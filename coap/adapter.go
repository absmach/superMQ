// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package coap contains the domain concept definitions needed to support
// Magistrala CoAP adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"context"
	"fmt"

	grpcChannelsV1 "github.com/absmach/supermq/internal/grpc/channels/v1"
	grpcThingsV1 "github.com/absmach/supermq/internal/grpc/things/v1"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

const chansPrefix = "channels"

// Service specifies CoAP service API.
type Service interface {
	// Publish publishes message to specified channel.
	// Key is used to authorize publisher.
	Publish(ctx context.Context, key string, msg *messaging.Message) error

	// Subscribes to channel with specified id, subtopic and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(ctx context.Context, key, chanID, subptopic, token string) error
}

var _ Service = (*adapterService)(nil)

// Observers is a map of maps,.
type adapterService struct {
	things   grpcThingsV1.ThingsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	pubsub   messaging.PubSub
}

// New instantiates the CoAP adapter implementation.
func New(things grpcThingsV1.ThingsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, pubsub messaging.PubSub) Service {
	as := &adapterService{
		things:   things,
		channels: channels,
		pubsub:   pubsub,
	}

	return as
}

func (svc *adapterService) Publish(ctx context.Context, key string, msg *messaging.Message) error {
	authnRes, err := svc.things.Authenticate(ctx, &grpcThingsV1.AuthnReq{
		ThingKey: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		ClientId:   authnRes.GetId(),
		ClientType: policies.ThingType,
		Permission: policies.PublishPermission,
		ChannelId:  msg.GetChannel(),
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	msg.Publisher = authnRes.GetId()

	return svc.pubsub.Publish(ctx, msg.GetChannel(), msg)
}

func (svc *adapterService) Subscribe(ctx context.Context, key, chanID, subtopic string, c Client) error {
	authnRes, err := svc.things.Authenticate(ctx, &grpcThingsV1.AuthnReq{
		ThingKey: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		ClientId:   authnRes.GetId(),
		ClientType: policies.ThingType,
		Permission: policies.SubscribePermission,
		ChannelId:  chanID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	subCfg := messaging.SubscriberConfig{
		ID:      c.Token(),
		Topic:   subject,
		Handler: c,
	}
	return svc.pubsub.Subscribe(ctx, subCfg)
}

func (svc *adapterService) Unsubscribe(ctx context.Context, key, chanID, subtopic, token string) error {
	authnRes, err := svc.things.Authenticate(ctx, &grpcThingsV1.AuthnReq{
		ThingKey: key,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.Authenticated {
		return svcerr.ErrAuthentication
	}

	authzRes, err := svc.channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		DomainId:   "",
		ClientId:   authnRes.GetId(),
		ClientType: policies.ThingType,
		Permission: policies.SubscribePermission,
		ChannelId:  chanID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.Authorized {
		return svcerr.ErrAuthorization
	}

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	return svc.pubsub.Unsubscribe(ctx, token, subject)
}
