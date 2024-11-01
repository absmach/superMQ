// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/clients"
	mgclients "github.com/absmach/supermq/pkg/clients"
	"github.com/absmach/supermq/pkg/roles"
)

type Connection struct {
	ThingID   string
	ChannelID string
	DomainID  string
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
//
//go:generate mockery --name Service --output=./mocks --filename service.go  --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateThings creates new client. In case of the failed registration, a
	// non-nil error value is returned.
	CreateThings(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, error)

	// ViewClient retrieves client info for a given client ID and an authorized token.
	ViewClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// ListClients retrieves clients list for a valid auth token.
	ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error)

	// UpdateClient updates the client's name and metadata.
	UpdateClient(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientTags updates the client's tags.
	UpdateClientTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error)

	// UpdateClientSecret updates the client's secret
	UpdateClientSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error)

	// EnableClient logically enableds the client identified with the provided ID
	EnableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// DisableClient logically disables the client identified with the provided ID
	DisableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error)

	// DeleteClient deletes client with given ID.
	DeleteClient(ctx context.Context, session authn.Session, id string) error

	SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error

	RemoveParentGroup(ctx context.Context, session authn.Session, id string) error

	roles.RoleManager
}

// Cache contains thing caching interface.
//
//go:generate mockery --name Cache --output=./mocks --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	// Save stores pair thing secret, thing id.
	Save(ctx context.Context, thingSecret, thingID string) error

	// ID returns thing ID for given thing secret.
	ID(ctx context.Context, thingSecret string) (string, error)

	// Removes thing from cache.
	Remove(ctx context.Context, thingID string) error
}

// Repository is the interface that wraps the basic methods for
// a client repository.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	mgclients.Repository

	// Save persists the client account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, client ...mgclients.Client) ([]mgclients.Client, error)

	// RetrieveBySecret retrieves a client based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (mgclients.Client, error)

	RemoveThings(ctx context.Context, clientIDs ...[]string) error

	RetrieveByIds(ctx context.Context, ids []string) (mgclients.ClientsPage, error)

	AddConnections(ctx context.Context, conns []Connection) error

	RemoveConnections(ctx context.Context, conns []Connection) error

	ThingConnectionsCount(ctx context.Context, id string) (uint64, error)

	DoesThingHaveConnections(ctx context.Context, id string) (bool, error)

	RemoveChannelConnections(ctx context.Context, channelID string) error

	RemoveThingConnections(ctx context.Context, thingID string) error

	// SetParentGroup set parent group id to a given channel id
	SetParentGroup(ctx context.Context, th clients.Client) error

	// RemoveParentGroup remove parent group id fr given chanel id
	RemoveParentGroup(ctx context.Context, th clients.Client) error

	RetrieveParentGroupThings(ctx context.Context, parentGroupID string) ([]clients.Client, error)

	UnsetParentGroupFromThings(ctx context.Context, parentGroupID string) error

	roles.Repository
}
