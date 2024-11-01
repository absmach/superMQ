// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	mgauth "github.com/absmach/supermq/auth"
	channels "github.com/absmach/supermq/channels/private"
	grpcChannelsV1 "github.com/absmach/supermq/internal/grpc/channels/v1"
	"github.com/absmach/supermq/pkg/apiutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ grpcChannelsV1.ChannelsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	grpcChannelsV1.UnimplementedChannelsServiceServer

	authorize                    kitgrpc.Handler
	removeThingConnections       kitgrpc.Handler
	unsetParentGroupFromChannels kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc channels.Service) grpcChannelsV1.ChannelsServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			authorizeEndpoint(svc),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		removeThingConnections: kitgrpc.NewServer(
			removeThingConnectionsEndpoint(svc),
			decodeRemoveThingConnectionsRequest,
			encodeRemoveThingConnectionsResponse,
		),
		unsetParentGroupFromChannels: kitgrpc.NewServer(
			unsetParentGroupFromChannelsEndpoint(svc),
			decodeUnsetParentGroupFromChannelsRequest,
			encodeUnsetParentGroupFromChannelsResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *grpcChannelsV1.AuthzReq) (*grpcChannelsV1.AuthzRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.AuthzRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.AuthzReq)

	return authorizeReq{
		domainID:   req.GetDomainId(),
		clientID:   req.GetClientId(),
		clientType: req.GetClientType(),
		channelID:  req.GetChannelId(),
		permission: req.GetPermission(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &grpcChannelsV1.AuthzRes{Authorized: res.authorized}, nil
}

func (s *grpcServer) RemoveThingConnections(ctx context.Context, req *grpcChannelsV1.RemoveThingConnectionsReq) (*grpcChannelsV1.RemoveThingConnectionsRes, error) {
	_, res, err := s.removeThingConnections.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.RemoveThingConnectionsRes), nil
}

func decodeRemoveThingConnectionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.RemoveThingConnectionsReq)

	return removeThingConnectionsReq{
		thingID: req.GetThingId(),
	}, nil
}

func encodeRemoveThingConnectionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(removeThingConnectionsRes)
	return &grpcChannelsV1.RemoveThingConnectionsRes{}, nil
}

func (s *grpcServer) UnsetParentGroupFromChannels(ctx context.Context, req *grpcChannelsV1.UnsetParentGroupFromChannelsReq) (*grpcChannelsV1.UnsetParentGroupFromChannelsRes, error) {
	_, res, err := s.unsetParentGroupFromChannels.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*grpcChannelsV1.UnsetParentGroupFromChannelsRes), nil
}

func decodeUnsetParentGroupFromChannelsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcChannelsV1.UnsetParentGroupFromChannelsReq)

	return unsetParentGroupFromChannelsReq{
		parentGroupID: req.GetParentGroupId(),
	}, nil
}

func encodeUnsetParentGroupFromChannelsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	_ = grpcRes.(unsetParentGroupFromChannelsRes)
	return &grpcChannelsV1.UnsetParentGroupFromChannelsRes{}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, mgauth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
