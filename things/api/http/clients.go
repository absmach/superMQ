// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"log/slog"

	"github.com/absmach/supermq/internal/api"
	"github.com/absmach/supermq/pkg/apiutil"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	roleManagerHttp "github.com/absmach/supermq/pkg/roles/rolemanager/api"
	"github.com/absmach/supermq/things"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func clientsHandler(svc things.Service, authn mgauthn.Authentication, r *chi.Mux, logger *slog.Logger) *chi.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	d := roleManagerHttp.NewDecoder("thingID")

	r.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))

		r.Route("/{domainID}/things", func(r chi.Router) {
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				createClientEndpoint(svc),
				decodeCreateClientReq,
				api.EncodeResponse,
				opts...,
			), "create_thing").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listClientsEndpoint(svc),
				decodeListClients,
				api.EncodeResponse,
				opts...,
			), "list_things").ServeHTTP)

			r.Post("/bulk", otelhttp.NewHandler(kithttp.NewServer(
				createClientsEndpoint(svc),
				decodeCreateClientsReq,
				api.EncodeResponse,
				opts...,
			), "create_things").ServeHTTP)
			r = roleManagerHttp.EntityAvailableActionsRouter(svc, d, r, opts)

			r.Route("/{thingID}", func(r chi.Router) {
				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					viewClientEndpoint(svc),
					decodeViewClient,
					api.EncodeResponse,
					opts...,
				), "view_thing").ServeHTTP)

				r.Patch("/", otelhttp.NewHandler(kithttp.NewServer(
					updateClientEndpoint(svc),
					decodeUpdateClient,
					api.EncodeResponse,
					opts...,
				), "update_thing").ServeHTTP)

				r.Patch("/tags", otelhttp.NewHandler(kithttp.NewServer(
					updateClientTagsEndpoint(svc),
					decodeUpdateClientTags,
					api.EncodeResponse,
					opts...,
				), "update_thing_tags").ServeHTTP)

				r.Patch("/secret", otelhttp.NewHandler(kithttp.NewServer(
					updateClientSecretEndpoint(svc),
					decodeUpdateClientCredentials,
					api.EncodeResponse,
					opts...,
				), "update_thing_credentials").ServeHTTP)

				r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
					enableClientEndpoint(svc),
					decodeChangeClientStatus,
					api.EncodeResponse,
					opts...,
				), "enable_thing").ServeHTTP)

				r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
					disableClientEndpoint(svc),
					decodeChangeClientStatus,
					api.EncodeResponse,
					opts...,
				), "disable_thing").ServeHTTP)

				r.Post("/parent", otelhttp.NewHandler(kithttp.NewServer(
					setThingParentGroupEndpoint(svc),
					decodeSetThingParentGroupStatus,
					api.EncodeResponse,
					opts...,
				), "set_thing_parent_group").ServeHTTP)

				r.Delete("/parent", otelhttp.NewHandler(kithttp.NewServer(
					removeThingParentGroupEndpoint(svc),
					decodeRemoveThingParentGroupStatus,
					api.EncodeResponse,
					opts...,
				), "remove_thing_parent_group").ServeHTTP)

				r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
					deleteClientEndpoint(svc),
					decodeDeleteClientReq,
					api.EncodeResponse,
					opts...,
				), "delete_thing").ServeHTTP)
				roleManagerHttp.EntityRoleMangerRouter(svc, d, r, opts)
			})
		})

		r.Get("/{domainID}/users/{userID}/things", otelhttp.NewHandler(kithttp.NewServer(
			listClientsEndpoint(svc),
			decodeListClients,
			api.EncodeResponse,
			opts...,
		), "list_user_things").ServeHTTP)
	})
	return r
}
