// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/authz"
	mgauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/clients"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	rmMW "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/pkg/svcutil"
	"github.com/absmach/supermq/things"
)

var (
	errView                   = errors.New("not authorized to view thing")
	errUpdate                 = errors.New("not authorized to update thing")
	errUpdateTags             = errors.New("not authorized to update thing tags")
	errUpdateSecret           = errors.New("not authorized to update thing secret")
	errEnable                 = errors.New("not authorized to enable thing")
	errDisable                = errors.New("not authorized to disable thing")
	errDelete                 = errors.New("not authorized to delete thing")
	errSetParentGroup         = errors.New("not authorized to set parent group to thing")
	errRemoveParentGroup      = errors.New("not authorized to remove parent group from thing")
	errDomainCreateThings     = errors.New("not authorized to create thing in domain")
	errGroupSetChildThings    = errors.New("not authorized to set child thing for group")
	errGroupRemoveChildThings = errors.New("not authorized to remove child thing for group")
)

var _ things.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc    things.Service
	repo   things.Repository
	authz  mgauthz.Authorization
	opp    svcutil.OperationPerm
	extOpp svcutil.ExternalOperationPerm
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(entityType string, svc things.Service, authz mgauthz.Authorization, repo things.Repository, thingsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission, extOpPerm map[svcutil.ExternalOperation]svcutil.Permission) (things.Service, error) {
	opp := things.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(thingsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}
	ram, err := rmMW.NewRoleManagerAuthorizationMiddleware(policies.ThingType, svc, authz, rolesOpPerm)
	if err != nil {
		return nil, err
	}
	extOpp := things.NewExternalOperationPerm()
	if err := extOpp.AddOperationPermissionMap(extOpPerm); err != nil {
		return nil, err
	}
	if err := extOpp.Validate(); err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		repo:                               repo,
		opp:                                opp,
		extOpp:                             extOpp,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateThings(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, error) {
	if err := am.extAuthorize(ctx, things.DomainOpCreateThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return []clients.Client{}, errors.Wrap(err, errDomainCreateThings)
	}

	return am.svc.CreateThings(ctx, session, client...)
}

func (am *authorizationMiddleware) ViewClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpViewThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errView)
	}
	return am.svc.ViewClient(ctx, session, id)
}

func (am *authorizationMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		session.SuperAdmin = true
	}

	return am.svc.ListClients(ctx, session, reqUserID, pm)
}

func (am *authorizationMiddleware) UpdateClient(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpUpdateThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      client.ID,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdate)
	}

	return am.svc.UpdateClient(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateClientTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpUpdateThingTags, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      client.ID,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdateTags)
	}

	return am.svc.UpdateClientTags(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateClientSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpUpdateThingSecret, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdateSecret)
	}
	return am.svc.UpdateClientSecret(ctx, session, id, key)
}

func (am *authorizationMiddleware) EnableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpEnableThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errEnable)
	}

	return am.svc.EnableClient(ctx, session, id)
}

func (am *authorizationMiddleware) DisableClient(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, things.OpDisableThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errDisable)
	}
	return am.svc.DisableClient(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteClient(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, things.OpDeleteThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errDelete)
	}

	return am.svc.DeleteClient(ctx, session, id)
}

func (am *authorizationMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error {
	if err := am.authorize(ctx, things.OpSetParentGroup, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errSetParentGroup)
	}

	if err := am.extAuthorize(ctx, things.GroupOpSetChildThing, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.GroupType,
		Object:      parentGroupID,
	}); err != nil {
		return errors.Wrap(err, errGroupSetChildThings)
	}
	return am.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (am *authorizationMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, things.OpRemoveParentGroup, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ThingType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errRemoveParentGroup)
	}

	th, err := am.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if th.ParentGroup != "" {
		if err := am.extAuthorize(ctx, things.GroupOpSetChildThing, authz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.GroupType,
			Object:      th.ParentGroup,
		}); err != nil {
			return errors.Wrap(err, errGroupRemoveChildThings)
		}
		return am.svc.RemoveParentGroup(ctx, session, id)
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, req authz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) extAuthorize(ctx context.Context, extOp svcutil.ExternalOperation, req authz.PolicyReq) error {
	perm, err := am.extOpp.GetPermission(extOp)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, userID string) error {
	if err := am.authz.Authorize(ctx, mgauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}); err != nil {
		return err
	}
	return nil
}
