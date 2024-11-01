// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

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

type ctxKey string

const (
	protocol                = "http"
	clientIDCtxKey   ctxKey = "client_id"
	clientTypeCtxKey ctxKey = "client_type"
)

// Log message formats.
const (
	logInfoConnected = "connected with thing_key %s"
	logInfoPublished = "published with client_type %s client_id %s to the topic %s"
)

// Error wrappers for MQTT errors.
var (
	errMalformedSubtopic        = errors.New("malformed subtopic")
	errClientNotInitialized     = errors.New("client is not initialized")
	errMalformedTopic           = errors.New("malformed topic")
	errMissingTopicPub          = errors.New("failed to publish due to missing topic")
	errFailedPublish            = errors.New("failed to publish")
	errFailedParseSubtopic      = errors.New("failed to parse subtopic")
	errFailedPublishToMsgBroker = errors.New("failed to publish to magistrala message broker")
)

var channelRegExp = regexp.MustCompile(`^\/?channels\/([\w\-]+)\/messages(\/[^?]*)?(\?.*)?$`)

// Event implements events.Event interface.
type handler struct {
	publisher messaging.Publisher
	things    grpcThingsV1.ThingsServiceClient
	channels  grpcChannelsV1.ChannelsServiceClient
	authn     mgauthn.Authentication
	logger    *slog.Logger
}

// NewHandler creates new Handler entity.
func NewHandler(publisher messaging.Publisher, authn mgauthn.Authentication, things grpcThingsV1.ThingsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, logger *slog.Logger) session.Handler {
	return &handler{
		publisher: publisher,
		authn:     authn,
		things:    things,
		channels:  channels,
		logger:    logger,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the HTTP server.
func (h *handler) AuthConnect(ctx context.Context) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	var tok string
	switch {
	case string(s.Password) == "":
		return errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey)
	case strings.HasPrefix(string(s.Password), "Thing"):
		tok = extractThingKey(string(s.Password))
	default:
		tok = string(s.Password)
	}

	h.logger.Info(fmt.Sprintf(logInfoConnected, tok))
	return nil
}

// AuthPublish is not used in HTTP service.
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	return nil
}

// AuthSubscribe is not used in HTTP service.
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Connect - after client successfully connected.
func (h *handler) Connect(ctx context.Context) error {
	return nil
}

// Publish - after client successfully published.
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	if topic == nil {
		return errMissingTopicPub
	}
	topic = &strings.Split(*topic, "?")[0]
	s, ok := session.FromContext(ctx)
	if !ok {
		return errors.Wrap(errFailedPublish, errClientNotInitialized)
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

	chanID, subtopic, err := parseTopic(*topic)
	if err != nil {
		return err
	}

	msg := messaging.Message{
		Protocol: protocol,
		Channel:  chanID,
		Subtopic: subtopic,
		Payload:  *payload,
		Created:  time.Now().UnixNano(),
	}

	ar := &grpcChannelsV1.AuthzReq{
		ClientId:   clientID,
		ClientType: clientType,
		ChannelId:  msg.Channel,
		Permission: policies.PublishPermission,
	}
	res, err := h.channels.Authorize(ctx, ar)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}

	if clientType == policies.ThingType {
		msg.Publisher = clientID
	}

	if err := h.publisher.Publish(ctx, msg.Channel, &msg); err != nil {
		return errors.Wrap(errFailedPublishToMsgBroker, err)
	}

	h.logger.Info(fmt.Sprintf(logInfoPublished, clientType, clientID, *topic))

	return nil
}

// Subscribe - not used for HTTP.
func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Unsubscribe - not used for HTTP.
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Disconnect - not used for HTTP.
func (h *handler) Disconnect(ctx context.Context) error {
	return nil
}

func parseTopic(topic string) (string, string, error) {
	// Topics are in the format:
	// channels/<channel_id>/messages/<subtopic>/.../ct/<content_type>
	channelParts := channelRegExp.FindStringSubmatch(topic)
	if len(channelParts) < 2 {
		return "", "", errors.Wrap(errFailedPublish, errMalformedTopic)
	}

	chanID := channelParts[1]
	subtopic := channelParts[2]

	subtopic, err := parseSubtopic(subtopic)
	if err != nil {
		return "", "", errors.Wrap(errFailedParseSubtopic, err)
	}

	return chanID, subtopic, nil
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
