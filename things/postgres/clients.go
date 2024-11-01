// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"fmt"

	"github.com/absmach/supermq/pkg/apiutil"
	"github.com/absmach/supermq/pkg/clients"
	mgclients "github.com/absmach/supermq/pkg/clients"
	pgclients "github.com/absmach/supermq/pkg/clients/postgres"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/postgres"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	"github.com/absmach/supermq/things"
)

const (
	entityTableName      = "clients"
	entityIDColumnName   = "id"
	rolesTableNamePrefix = "things"
)

var _ things.Repository = (*thingsRepo)(nil)

type thingsRepo struct {
	clientsRepo
	rolesPostgres.Repository
}

// NewRepository instantiates a PostgreSQL
// implementation of Clients repository.
func NewRepository(db postgres.Database) things.Repository {
	repo := rolesPostgres.NewRepository(db, rolesTableNamePrefix, entityTableName, entityIDColumnName)

	return &thingsRepo{
		clientsRepo{
			pgclients.Repository{DB: db},
		},
		repo,
	}
}

type clientsRepo struct {
	pgclients.Repository
}

func (repo *clientsRepo) Save(ctx context.Context, cs ...mgclients.Client) (retClients []mgclients.Client, retErr error) {
	var dbClients []pgclients.DBClient
	for _, cli := range cs {
		dbcli, err := pgclients.ToDBClient(cli)
		if err != nil {
			return []mgclients.Client{}, errors.Wrap(repoerr.ErrCreateEntity, err)
		}
		dbClients = append(dbClients, dbcli)
	}

	q := `INSERT INTO clients (id, name, tags, domain_id, parent_group_id,  identity, secret, metadata, created_at, updated_at, updated_by, status)
	VALUES (:id, :name, :tags, :domain_id,  :parent_group_id, :identity, :secret, :metadata, :created_at, :updated_at, :updated_by, :status)
	RETURNING id, name, tags, identity, secret, metadata, COALESCE(domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, status, created_at, updated_at, updated_by`

	row, err := repo.DB.NamedQueryContext(ctx, q, dbClients)
	if err != nil {
		return []mgclients.Client{}, postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	defer row.Close()

	var clients []mgclients.Client

	for row.Next() {
		dbcli := pgclients.DBClient{}
		if err := row.StructScan(&dbcli); err != nil {
			return []mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		client, err := pgclients.ToClient(dbcli)
		if err != nil {
			return []mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}
		clients = append(clients, client)
	}

	return clients, nil
}

func (repo *clientsRepo) RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error) {
	q := fmt.Sprintf(`SELECT id, name, tags, COALESCE(domain_id, '') AS domain_id,  COALESCE(parent_group_id, '') AS parent_group_id, identity, secret, metadata, created_at, updated_at, updated_by, status
        FROM clients
        WHERE secret = :secret AND status = %d`, mgclients.EnabledStatus)

	dbc := pgclients.DBClient{
		Secret: key,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, q, dbc)
	if err != nil {
		return mgclients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	dbc = pgclients.DBClient{}
	if rows.Next() {
		if err = rows.StructScan(&dbc); err != nil {
			return mgclients.Client{}, postgres.HandleError(repoerr.ErrViewEntity, err)
		}

		client, err := pgclients.ToClient(dbc)
		if err != nil {
			return mgclients.Client{}, errors.Wrap(repoerr.ErrFailedOpDB, err)
		}

		return client, nil
	}

	return mgclients.Client{}, repoerr.ErrNotFound
}

func (repo *clientsRepo) RemoveThings(ctx context.Context, clientIDs ...[]string) error {
	q := "DELETE FROM clients AS c  WHERE c.id = ANY(:client_ids) ;"

	params := map[string]interface{}{
		"client_ids": clientIDs,
	}
	result, err := repo.DB.NamedExecContext(ctx, q, params)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}

func (repo *clientsRepo) RetrieveByIds(ctx context.Context, ids []string) (mgclients.ClientsPage, error) {
	if len(ids) == 0 {
		return clients.ClientsPage{}, nil
	}

	// To avoid adding c.Role in query adding Roles: mgclients.AllRole
	pm := mgclients.Page{IDs: ids, Role: mgclients.AllRole}
	query, err := pgclients.PageQuery(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	q := fmt.Sprintf(`SELECT c.id, c.name, c.tags, c.identity, c.metadata, COALESCE(c.domain_id, '') AS domain_id,  COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c %s ORDER BY c.created_at`, query)

	dbPage, err := pgclients.ToDBClientsPage(pm)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	rows, err := repo.DB.NamedQueryContext(ctx, q, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrFailedToRetrieveAllGroups, err)
	}
	defer rows.Close()

	var items []clients.Client
	for rows.Next() {
		dbc := pgclients.DBClient{}
		if err := rows.StructScan(&dbc); err != nil {
			return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		c, err := pgclients.ToClient(dbc)
		if err != nil {
			return clients.ClientsPage{}, err
		}

		items = append(items, c)
	}
	cq := fmt.Sprintf(`SELECT COUNT(*) FROM clients c %s;`, query)

	total, err := postgres.Total(ctx, repo.DB, cq, dbPage)
	if err != nil {
		return clients.ClientsPage{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}

	page := clients.ClientsPage{
		Clients: items,
		Page: clients.Page{
			Total:  total,
			Offset: pm.Offset,
			Limit:  total,
		},
	}

	return page, nil
}

func (repo *clientsRepo) AddConnections(ctx context.Context, conns []things.Connection) error {
	dbConns := toDBConnections(conns)

	q := `INSERT INTO connections (channel_id, domain_id, thing_id)
			VALUES (:channel_id, :domain_id, :thing_id);`

	if _, err := repo.DB.NamedExecContext(ctx, q, dbConns); err != nil {
		return postgres.HandleError(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (repo *clientsRepo) RemoveConnections(ctx context.Context, conns []things.Connection) (retErr error) {
	tx, err := repo.DB.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	defer func() {
		if retErr != nil {
			if errRollBack := tx.Rollback(); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollBack))
			}
		}
	}()

	query := `DELETE FROM connections WHERE channel_id = :channel_id AND domain_id = :domain_id AND thing_id = :thing_id`

	for _, conn := range conns {
		dbConn := toDBConnection(conn)
		if _, err := tx.NamedExec(query, dbConn); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, errors.Wrap(fmt.Errorf("failed to delete connection for channel_id: %s, domain_id: %s thing_id %s", conn.ChannelID, conn.DomainID, conn.ThingID), err))
		}
	}
	if err := tx.Commit(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientsRepo) SetParentGroup(ctx context.Context, th clients.Client) error {
	q := "UPDATE clients SET parent_group_id = :parent_group_id, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"

	params := map[string]interface{}{
		"parent_group_id": th.ParentGroup,
		"updated_at":      th.UpdatedAt,
		"updated_by":      th.UpdatedBy,
		"id":              th.ID,
	}
	result, err := repo.DB.NamedExecContext(ctx, q, params)
	if err != nil {
		return postgres.HandleError(repoerr.ErrUpdateEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (repo *clientsRepo) RemoveParentGroup(ctx context.Context, th clients.Client) error {
	q := "UPDATE clients SET parent_group_id = NULL, updated_at = :updated_at, updated_by = :updated_by WHERE id = :id"
	dbCh, err := pgclients.ToDBClient(th)
	if err != nil {
		return errors.Wrap(repoerr.ErrUpdateEntity, err)
	}
	result, err := repo.DB.NamedExecContext(ctx, q, dbCh)
	if err != nil {
		return postgres.HandleError(repoerr.ErrRemoveEntity, err)
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return repoerr.ErrNotFound
	}
	return nil
}

func (repo *clientsRepo) ThingConnectionsCount(ctx context.Context, id string) (uint64, error) {
	query := `SELECT COUNT(*) FROM connections WHERE thing_id = :thing_id`
	dbConn := dbConnection{ThingID: id}

	total, err := postgres.Total(ctx, repo.DB, query, dbConn)
	if err != nil {
		return 0, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	return total, nil
}

func (repo *clientsRepo) DoesThingHaveConnections(ctx context.Context, id string) (bool, error) {
	query := `SELECT 1 FROM connections WHERE thing_id = :thing_id`
	dbConn := dbConnection{ThingID: id}

	rows, err := repo.DB.NamedQueryContext(ctx, query, dbConn)
	if err != nil {
		return false, postgres.HandleError(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	return rows.Next(), nil
}

func (repo *clientsRepo) RemoveChannelConnections(ctx context.Context, channelID string) error {
	query := `DELETE FROM connections WHERE channel_id = :channel_id`

	dbConn := dbConnection{ChannelID: channelID}
	if _, err := repo.DB.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientsRepo) RemoveThingConnections(ctx context.Context, thingID string) error {
	query := `DELETE FROM connections WHERE thing_id = :thing_id`

	dbConn := dbConnection{ThingID: thingID}
	if _, err := repo.DB.NamedExecContext(ctx, query, dbConn); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (repo *clientsRepo) RetrieveParentGroupThings(ctx context.Context, parentGroupID string) ([]clients.Client, error) {
	query := `SELECT c.id, c.name, c.tags,  c.metadata, COALESCE(c.domain_id, '') AS domain_id, COALESCE(parent_group_id, '') AS parent_group_id, c.status,
					c.created_at, c.updated_at, COALESCE(c.updated_by, '') AS updated_by FROM clients c WHERE c.parent_group_id = :parent_group_id ;`

	params := map[string]interface{}{
		"parent_group_id": parentGroupID,
	}

	rows, err := repo.DB.NamedQueryContext(ctx, query, params)
	if err != nil {
		return []clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
	}
	defer rows.Close()

	var ths []clients.Client
	for rows.Next() {
		dbTh := pgclients.DBClient{}
		if err := rows.StructScan(&dbTh); err != nil {
			return []clients.Client{}, errors.Wrap(repoerr.ErrViewEntity, err)
		}

		th, err := pgclients.ToClient(dbTh)
		if err != nil {
			return []clients.Client{}, err
		}

		ths = append(ths, th)
	}
	return ths, nil
}

func (repo *clientsRepo) UnsetParentGroupFromThings(ctx context.Context, parentGroupID string) error {
	query := "UPDATE clients SET parent_group_id = NULL WHERE parent_group_id = :parent_group_id"

	params := map[string]interface{}{
		"parent_group_id": parentGroupID,
	}
	if _, err := repo.DB.NamedExecContext(ctx, query, params); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

type dbConnection struct {
	ThingID   string `db:"thing_id"`
	ChannelID string `db:"channel_id"`
	DomainID  string `db:"domain_id"`
}

func toDBConnections(conns []things.Connection) []dbConnection {
	var dbconns []dbConnection
	for _, conn := range conns {
		dbconns = append(dbconns, toDBConnection(conn))
	}
	return dbconns
}

func toDBConnection(conn things.Connection) dbConnection {
	return dbConnection{
		ThingID:   conn.ThingID,
		ChannelID: conn.ChannelID,
		DomainID:  conn.DomainID,
	}
}
