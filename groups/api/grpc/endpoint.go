// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	groups "github.com/absmach/supermq/groups/private"
	"github.com/go-kit/kit/endpoint"
)

func retrieveEntityEndpoint(svc groups.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveEntityReq)
		thing, err := svc.RetrieveById(ctx, req.Id)
		if err != nil {
			return retrieveEntityRes{}, err
		}

		return retrieveEntityRes{id: thing.ID, domain: thing.Domain, status: uint8(thing.Status)}, nil
	}
}
