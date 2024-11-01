// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/absmach/mproxy/pkg/session"
	grpcChannelsV1 "github.com/absmach/supermq/internal/grpc/channels/v1"
	grpcThingsV1 "github.com/absmach/supermq/internal/grpc/things/v1"
	"github.com/absmach/supermq/pkg/apiutil"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var _ session.Handler = (*handler)(nil)

const protocol = "websocket"

// Log message formats.
const (
	LogInfoSubscribed   = "subscribed with client_id %s to topics %s"
	LogInfoUnsubscribed = "unsubscribed client_id %s from topics %s"
	LogInfoConnected    = "connected with client_id %s"
	LogInfoDisconnected = "disconnected client_id %s and username %s"
	LogInfoPublished    = "published with client_id %s to the topic %s"
)

// Error wrappers for MQTT errors.
var (
	errMalformedSubtopic        = errors.New("malformed subtopic")
	errClientNotInitialized     = errors.New("client is not initialized")
	errMalformedTopic           = errors.New("malformed topic")
	errMissingTopicPub          = errors.New("failed to publish due to missing topic")
	errMissingTopicSub          = errors.New("failed to subscribe due to missing topic")
	errFailedSubscribe          = errors.New("failed to subscribe")
	errFailedPublish            = errors.New("failed to publish")
	errFailedParseSubtopic      = errors.New("failed to parse subtopic")
	errFailedPublishToMsgBroker = errors.New("failed to publish to magistrala message broker")
)

var channelRegExp = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)

// Event implements events.Event interface.
type handler struct {
	pubsub   messaging.PubSub
	things   grpcThingsV1.ThingsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	authn    mgauthn.Authentication
	logger   *slog.Logger
}

// NewHandler creates new Handler entity.
func NewHandler(pubsub messaging.PubSub, logger *slog.Logger, authn mgauthn.Authentication, things grpcThingsV1.ThingsServiceClient, channels grpcChannelsV1.ChannelsServiceClient) session.Handler {
	return &handler{
		logger:   logger,
		pubsub:   pubsub,
		authn:    authn,
		things:   things,
		channels: channels,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the ws server.
func (h *handler) AuthConnect(ctx context.Context) error {
	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the ws server.
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	if topic == nil {
		return errMissingTopicPub
	}
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	var token string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		token = strings.ReplaceAll(string(s.Password), "Thing ", "")
	default:
		token = string(s.Password)
	}

	return h.authAccess(ctx, token, *topic, policies.PublishPermission)
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker.
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return errMissingTopicSub
	}

	var token string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		token = strings.ReplaceAll(string(s.Password), "Thing ", "")
	default:
		token = string(s.Password)
	}

	for _, topic := range *topics {
		if err := h.authAccess(ctx, token, topic, policies.SubscribePermission); err != nil {
			return err
		}
	}

	return nil
}

// Connect - after client successfully connected.
func (h *handler) Connect(ctx context.Context) error {
	return nil
}

// Publish - after client successfully published.
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(errFailedPublish, errClientNotInitialized)
	}
	h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))

	if len(*payload) == 0 {
		return errFailedMessagePublish
	}

	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	channelParts := channelRegExp.FindStringSubmatch(*topic)
	if len(channelParts) < 2 {
		return errors.Wrap(errFailedPublish, errMalformedTopic)
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		return errors.Wrap(errFailedParseSubtopic, err)
	}

	var clientID, clientType string
	switch {
	case strings.HasPrefix(string(s.Password), "Thing"):
		thingKey := extractThingKey(string(s.Password))
		authnRes, err := h.things.Authenticate(ctx, &grpcThingsV1.AuthnReq{ThingKey: thingKey})
		if err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}
		if !authnRes.Authenticated {
			return svcerr.ErrAuthentication
		}
		clientType = policies.ThingType
		clientID = authnRes.GetId()
	default:
		token := string(s.Password)
		authnSession, err := h.authn.Authenticate(ctx, extractBearerToken(token))
		if err != nil {
			return err
		}
		clientType = policies.UserType
		clientID = authnSession.DomainUserID
	}

	ar := &grpcChannelsV1.AuthzReq{
		Permission: policies.PublishPermission,
		ClientId:   clientID,
		ClientType: clientType,
		ChannelId:  chanID,
	}
	res, err := h.channels.Authorize(ctx, ar)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}

	msg := messaging.Message{
		Protocol: protocol,
		Channel:  chanID,
		Subtopic: subtopic,
		Payload:  *payload,
		Created:  time.Now().UnixNano(),
	}

	if clientType == policies.ThingType {
		msg.Publisher = clientID
	}

	if err := h.pubsub.Publish(ctx, msg.GetChannel(), &msg); err != nil {
		return errors.Wrap(errFailedPublishToMsgBroker, err)
	}

	return nil
}

// Subscribe - after client successfully subscribed.
func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(errFailedSubscribe, errClientNotInitialized)
	}
	h.logger.Info(fmt.Sprintf(LogInfoSubscribed, s.ID, strings.Join(*topics, ",")))
	return nil
}

// Unsubscribe - after client unsubscribed.
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(errFailedUnsubscribe, errClientNotInitialized)
	}

	h.logger.Info(fmt.Sprintf(LogInfoUnsubscribed, s.ID, strings.Join(*topics, ",")))
	return nil
}

// Disconnect - connection with broker or client lost.
func (h *handler) Disconnect(ctx context.Context) error {
	return nil
}

func (h *handler) authAccess(ctx context.Context, token, topic, action string) error {
	var clientID, clientType string
	switch {
	case strings.HasPrefix(token, "Thing"):
		thingKey := extractThingKey(token)
		authnRes, err := h.things.Authenticate(ctx, &grpcThingsV1.AuthnReq{ThingKey: thingKey})
		if err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}
		if !authnRes.Authenticated {
			return svcerr.ErrAuthentication
		}
		clientType = policies.ThingType
		clientID = authnRes.GetId()
	default:
		authnSession, err := h.authn.Authenticate(ctx, extractBearerToken(token))
		if err != nil {
			return err
		}
		clientType = policies.UserType
		clientID = authnSession.DomainUserID
	}

	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	if !channelRegExp.MatchString(topic) {
		return errMalformedTopic
	}

	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 1 {
		return errMalformedTopic
	}

	chanID := channelParts[1]

	ar := &grpcChannelsV1.AuthzReq{
		Permission: action,
		ClientId:   clientID,
		ClientType: clientType,
		ChannelId:  chanID,
	}
	res, err := h.channels.Authorize(ctx, ar)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return nil
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
	}
	subtopic = strings.ReplaceAll(subtopic, "/", ".")

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", errMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")
	return subtopic, nil
}

// extractThingKey returns value of the thing key. If there is no thing key - an empty value is returned.
func extractThingKey(topic string) string {
	if !strings.HasPrefix(topic, apiutil.ThingPrefix) {
		return ""
	}

	return strings.TrimPrefix(topic, apiutil.ThingPrefix)
}

// extractBearerToken
func extractBearerToken(token string) string {
	if !strings.HasPrefix(token, apiutil.BearerPrefix) {
		return ""
	}

	return strings.TrimPrefix(token, apiutil.BearerPrefix)
}
