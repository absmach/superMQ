// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

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

var (
	// errFailedMessagePublish indicates that message publishing failed.
	errFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// errFailedUnsubscribe indicates that client couldn't unsubscribe from specified channel.
	errFailedUnsubscribe = errors.New("failed to unsubscribe from a channel")

	// ErrEmptyTopic indicate absence of thingKey in the request.
	ErrEmptyTopic = errors.New("empty topic")
)

// Service specifies web socket service API.
type Service interface {
	// Subscribe subscribes message from the broker using the thingKey for authorization,
	// and the channelID for subscription. Subtopic is optional.
	// If the subscription is successful, nil is returned otherwise error is returned.
	Subscribe(ctx context.Context, thingKey, chanID, subtopic string, client *Client) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	things   grpcThingsV1.ThingsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	pubsub   messaging.PubSub
}

// New instantiates the WS adapter implementation.
func New(things grpcThingsV1.ThingsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, pubsub messaging.PubSub) Service {
	return &adapterService{
		things:   things,
		channels: channels,
		pubsub:   pubsub,
	}
}

func (svc *adapterService) Subscribe(ctx context.Context, thingKey, chanID, subtopic string, c *Client) error {
	if chanID == "" || thingKey == "" {
		return svcerr.ErrAuthentication
	}

	thingID, err := svc.authorize(ctx, thingKey, chanID, policies.SubscribePermission)
	if err != nil {
		return svcerr.ErrAuthorization
	}

	c.id = thingID

	subject := fmt.Sprintf("%s.%s", chansPrefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}

	subCfg := messaging.SubscriberConfig{
		ID:      thingID,
		Topic:   subject,
		Handler: c,
	}
	if err := svc.pubsub.Subscribe(ctx, subCfg); err != nil {
		return ErrFailedSubscription
	}

	return nil
}

// authorize checks if the thingKey is authorized to access the channel
// and returns the thingID if it is.
func (svc *adapterService) authorize(ctx context.Context, thingKey, chanID, action string) (string, error) {
	authnReq := &grpcThingsV1.AuthnReq{
		ThingKey: thingKey,
	}
	authnRes, err := svc.things.Authenticate(ctx, authnReq)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if !authnRes.GetAuthenticated() {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}

	authzReq := &grpcChannelsV1.AuthzReq{
		ClientType: policies.ThingType,
		ClientId:   authnRes.GetId(),
		Permission: action,
		ChannelId:  chanID,
	}
	authzRes, err := svc.channels.Authorize(ctx, authzReq)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !authzRes.GetAuthorized() {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return authnRes.GetId(), nil
}
