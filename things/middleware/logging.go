// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	mgclients "github.com/absmach/supermq/pkg/clients"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/things"
)

var _ things.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger *slog.Logger
	svc    things.Service
	rmMW.RoleManagerLoggingMiddleware
}

func LoggingMiddleware(svc things.Service, logger *slog.Logger) things.Service {
	return &loggingMiddleware{
		logger:                       logger,
		svc:                          svc,
		RoleManagerLoggingMiddleware: rmMW.NewRoleManagerLoggingMiddleware("things", svc, logger),
	}
}

// Things service
func (lm *loggingMiddleware) CreateThings(ctx context.Context, session authn.Session, clients ...mgclients.Client) (cs []mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn(fmt.Sprintf("Create %d things failed", len(clients)), args...)
			return
		}
		lm.logger.Info(fmt.Sprintf("Create %d things completed successfully", len(clients)), args...)
	}(time.Now())
	return lm.svc.CreateThings(ctx, session, clients...)
}

func (lm *loggingMiddleware) ViewClient(ctx context.Context, session authn.Session, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("View thing failed", args...)
			return
		}
		lm.logger.Info("View thing completed successfully", args...)
	}(time.Now())
	return lm.svc.ViewClient(ctx, session, id)
}

func (lm *loggingMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm mgclients.Page) (cp mgclients.ClientsPage, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("user_id", reqUserID),
			slog.Group("page",
				slog.Uint64("limit", pm.Limit),
				slog.Uint64("offset", pm.Offset),
				slog.Uint64("total", cp.Total),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("List things failed", args...)
			return
		}
		lm.logger.Info("List things completed successfully", args...)
	}(time.Now())
	return lm.svc.ListClients(ctx, session, reqUserID, pm)
}

func (lm *loggingMiddleware) UpdateClient(ctx context.Context, session authn.Session, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", client.ID),
				slog.String("name", client.Name),
				slog.Any("metadata", client.Metadata),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update thing failed", args...)
			return
		}
		lm.logger.Info("Update thing completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClient(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateClientTags(ctx context.Context, session authn.Session, client mgclients.Client) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
				slog.Any("tags", c.Tags),
			),
		}
		if err != nil {
			args := append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update thing tags failed", args...)
			return
		}
		lm.logger.Info("Update thing tags completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClientTags(ctx, session, client)
}

func (lm *loggingMiddleware) UpdateClientSecret(ctx context.Context, session authn.Session, oldSecret, newSecret string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", c.ID),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Update thing secret failed", args...)
			return
		}
		lm.logger.Info("Update thing secret completed successfully", args...)
	}(time.Now())
	return lm.svc.UpdateClientSecret(ctx, session, oldSecret, newSecret)
}

func (lm *loggingMiddleware) EnableClient(ctx context.Context, session authn.Session, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Enable thing failed", args...)
			return
		}
		lm.logger.Info("Enable thing completed successfully", args...)
	}(time.Now())
	return lm.svc.EnableClient(ctx, session, id)
}

func (lm *loggingMiddleware) DisableClient(ctx context.Context, session authn.Session, id string) (c mgclients.Client, err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.Group("thing",
				slog.String("id", id),
				slog.String("name", c.Name),
			),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Disable thing failed", args...)
			return
		}
		lm.logger.Info("Disable thing completed successfully", args...)
	}(time.Now())
	return lm.svc.DisableClient(ctx, session, id)
}

func (lm *loggingMiddleware) DeleteClient(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Delete thing failed", args...)
			return
		}
		lm.logger.Info("Delete thing completed successfully", args...)
	}(time.Now())
	return lm.svc.DeleteClient(ctx, session, id)
}

func (lm *loggingMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("parent_group_id", parentGroupID),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Set parent group to thing failed", args...)
			return
		}
		lm.logger.Info("Set parent group to thing completed successfully", args...)
	}(time.Now())
	return lm.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (lm *loggingMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	defer func(begin time.Time) {
		args := []any{
			slog.String("duration", time.Since(begin).String()),
			slog.String("thing_id", id),
		}
		if err != nil {
			args = append(args, slog.String("error", err.Error()))
			lm.logger.Warn("Remove parent group from thing failed", args...)
			return
		}
		lm.logger.Info("Remove parent group from thing completed successfully", args...)
	}(time.Now())
	return lm.svc.RemoveParentGroup(ctx, session, id)
}
