// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/supermq/things"
	pThings "github.com/absmach/supermq/things/private"
	"github.com/go-kit/kit/endpoint"
)

func authenticateEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authenticateReq)

		thingID, err := svc.Authenticate(ctx, req.ThingKey)
		if err != nil {
			return authenticateRes{}, err
		}
		return authenticateRes{
			authenticated: true,
			id:            thingID,
		}, err
	}
}

func retrieveEntityEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		thing, err := svc.RetrieveById(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: thing.ID, domain: thing.Domain, status: uint8(thing.Status)}, nil
	}
}

func retrieveEntitiesEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntitiesReq)
		tp, err := svc.RetrieveByIds(ctx, req.Ids)
		if err != nil {
			return retrieveEntitiesRes{}, err
		}
		thingsBasic := []thingBasic{}
		for _, thing := range tp.Clients {
			thingsBasic = append(thingsBasic, thingBasic{id: thing.ID, domain: thing.Domain, status: uint8(thing.Status)})
		}
		return retrieveEntitiesRes{
			total:  tp.Total,
			limit:  tp.Limit,
			offset: tp.Offset,
			things: thingsBasic,
		}, nil
	}
}

func addConnectionsEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectionsReq)

		var conns []things.Connection

		for _, c := range req.connections {
			conns = append(conns, things.Connection{
				ThingID:   c.thingID,
				ChannelID: c.channelID,
				DomainID:  c.domainID,
			})
		}
		if err := svc.AddConnections(ctx, conns); err != nil {
			return connectionsRes{ok: false}, err
		}

		return connectionsRes{ok: true}, nil
	}
}

func removeConnectionsEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectionsReq)

		var conns []things.Connection

		for _, c := range req.connections {
			conns = append(conns, things.Connection{
				ThingID:   c.thingID,
				ChannelID: c.channelID,
				DomainID:  c.domainID,
			})
		}
		if err := svc.RemoveConnections(ctx, conns); err != nil {
			return connectionsRes{ok: false}, err
		}

		return connectionsRes{ok: true}, nil
	}
}

func removeChannelConnectionsEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeChannelConnectionsReq)

		if err := svc.RemoveChannelConnections(ctx, req.channelID); err != nil {
			return removeChannelConnectionsRes{}, err
		}

		return removeChannelConnectionsRes{}, nil
	}
}

func unsetParentGroupFromThingsEndpoint(svc pThings.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unsetParentGroupFromThingsReq)

		if err := svc.UnsetParentGroupFromThings(ctx, req.parentGroupID); err != nil {
			return unsetParentGroupFromThingsRes{}, err
		}

		return unsetParentGroupFromThingsRes{}, nil
	}
}
