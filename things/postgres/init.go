// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	rolesPostgres "github.com/absmach/supermq/pkg/roles/repo/postgres"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	migrate "github.com/rubenv/sql-migrate"
)

func Migration() (*migrate.MemoryMigrationSource, error) {
	thingsRolesMigration, err := rolesPostgres.Migration(rolesTableNamePrefix, entityTableName, entityIDColumnName)
	if err != nil {
		return &migrate.MemoryMigrationSource{}, errors.Wrap(repoerr.ErrRoleMigration, err)
	}

	thingsMigration := &migrate.MemoryMigrationSource{
		Migrations: []*migrate.Migration{
			{
				Id: "clients_01",
				// VARCHAR(36) for colums with IDs as UUIDS have a maximum of 36 characters
				// STATUS 0 to imply enabled and 1 to imply disabled
				Up: []string{
					`CREATE TABLE IF NOT EXISTS clients (
						id			       VARCHAR(36) PRIMARY KEY,
						name		       VARCHAR(1024),
						domain_id	       VARCHAR(36) NOT NULL,
						parent_group_id    VARCHAR(36) DEFAULT NULL,
						identity	       VARCHAR(254),
						secret		       VARCHAR(4096) NOT NULL,
						tags		       TEXT[],
						metadata	       JSONB,
						created_at	       TIMESTAMP,
						updated_at	       TIMESTAMP,
						updated_by         VARCHAR(254),
						status		       SMALLINT NOT NULL DEFAULT 0 CHECK (status >= 0),
						UNIQUE		       (domain_id, secret),
						UNIQUE		       (domain_id, name),
						UNIQUE		       (domain_id, id)
					)`,
					`CREATE TABLE IF NOT EXISTS connections (
						channel_id    VARCHAR(36),
						domain_id 	  VARCHAR(36),
						thing_id      VARCHAR(36),
						FOREIGN KEY (thing_id, domain_id) REFERENCES clients (id, domain_id) ON DELETE CASCADE ON UPDATE CASCADE,
						PRIMARY KEY (channel_id, domain_id, thing_id)
					)`,
				},
				Down: []string{
					`DROP TABLE IF EXISTS clients`,
					`DROP TABLE IF EXISTS connections`,
				},
			},
		},
	}

	thingsMigration.Migrations = append(thingsMigration.Migrations, thingsRolesMigration.Migrations...)

	return thingsMigration, nil
}
