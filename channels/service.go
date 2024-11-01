package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	grpcCommonV1 "github.com/absmach/supermq/internal/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/supermq/internal/grpc/groups/v1"
	grpcThingsV1 "github.com/absmach/supermq/internal/grpc/things/v1"
	"github.com/absmach/supermq/pkg/apiutil"
	"github.com/absmach/supermq/pkg/authn"
	mgclients "github.com/absmach/supermq/pkg/clients"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	"golang.org/x/sync/errgroup"
)

var (
	errCreateChannelsPolicies  = errors.New("failed to create channels policies")
	errRollbackRepo            = errors.New("failed to rollback repo")
	errAddConnectionsThings    = errors.New("failed to add connections in things service")
	errRemoveConnectionsThings = errors.New("failed to remove connections from things service")
	errSetParentGroup          = errors.New("channel already have parent")
)

type service struct {
	repo       Repository
	policy     policies.Service
	idProvider magistrala.IDProvider
	things     grpcThingsV1.ThingsServiceClient
	groups     grpcGroupsV1.GroupsServiceClient
	roles.ProvisionManageService
}

var _ Service = (*service)(nil)

func New(repo Repository, policy policies.Service, idProvider magistrala.IDProvider, things grpcThingsV1.ThingsServiceClient, groups grpcGroupsV1.GroupsServiceClient, sidProvider magistrala.IDProvider) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.ChannelType, repo, policy, sidProvider, AvailableActions(), BuiltInRoles())
	if err != nil {
		return nil, err
	}

	return service{
		repo:                   repo,
		policy:                 policy,
		idProvider:             idProvider,
		things:                 things,
		groups:                 groups,
		ProvisionManageService: rpms,
	}, nil
}

func (svc service) CreateChannels(ctx context.Context, session authn.Session, chs ...Channel) (retChs []Channel, retErr error) {
	var reChs []Channel
	for _, c := range chs {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []Channel{}, err
			}
			c.ID = clientID
		}

		if c.Status != mgclients.DisabledStatus && c.Status != mgclients.EnabledStatus {
			return []Channel{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		reChs = append(reChs, c)
	}

	savedChs, err := svc.repo.Save(ctx, reChs...)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	chIDs := []string{}
	for _, c := range savedChs {
		chIDs = append(chIDs, c.ID)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := svc.repo.Remove(ctx, chIDs...); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{}

	for _, chID := range chIDs {
		optionalPolicies = append(optionalPolicies,
			policies.Policy{
				SubjectType: policies.DomainType,
				Subject:     session.DomainID,
				Relation:    policies.DomainRelation,
				ObjectType:  policies.ChannelType,
				Object:      chID,
			},
		)
	}
	if _, err := svc.AddNewEntitiesRoles(ctx, session.DomainID, session.UserID, chIDs, optionalPolicies, newBuiltInRoleMembers); err != nil {
		return []Channel{}, errors.Wrap(errCreateChannelsPolicies, err)
	}
	return savedChs, nil
}

func (svc service) UpdateChannel(ctx context.Context, session authn.Session, ch Channel) (Channel, error) {
	channel := Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Metadata:  ch.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	channel, err := svc.repo.Update(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (svc service) UpdateChannelTags(ctx context.Context, session authn.Session, ch Channel) (Channel, error) {
	channel := Channel{
		ID:        ch.ID,
		Tags:      ch.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	channel, err := svc.repo.UpdateTags(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}

func (svc service) EnableChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel := Channel{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := svc.changeChannelStatus(ctx, session.UserID, channel)
	if err != nil {
		return Channel{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return ch, nil
}

func (svc service) DisableChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel := Channel{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	ch, err := svc.changeChannelStatus(ctx, session.UserID, channel)
	if err != nil {
		return Channel{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	return ch, nil
}

func (svc service) ViewChannel(ctx context.Context, session authn.Session, id string) (Channel, error) {
	channel, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return channel, nil
}

func (svc service) ListChannels(ctx context.Context, session authn.Session, pm PageMetadata) (Page, error) {
	var ids []string
	var err error

	switch session.SuperAdmin {
	case true:
		pm.Domain = session.DomainID
	default:
		ids, err = svc.listChannelIDs(ctx, session.DomainUserID, pm.Permission)
		if err != nil {
			return Page{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	}
	if len(ids) == 0 && pm.Domain == "" {
		return Page{}, nil
	}
	pm.IDs = ids

	cp, err := svc.repo.RetrieveAll(ctx, pm)
	if err != nil {
		return Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(cp.Channels) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range cp.Channels {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, session.DomainUserID, &cp.Channels[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return Page{}, err
		}
	}
	return cp, nil
}

func (svc service) ListChannelsByThing(ctx context.Context, session authn.Session, thID string, pm PageMetadata) (Page, error) {
	return Page{}, nil
}

func (svc service) RemoveChannel(ctx context.Context, session authn.Session, id string) error {
	ok, err := svc.repo.DoesChannelHaveConnections(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if ok {
		if _, err := svc.things.RemoveChannelConnections(ctx, &grpcThingsV1.RemoveChannelConnectionsReq{ChannelId: id}); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
	}
	ch, err := svc.repo.ChangeStatus(ctx, Channel{ID: id, Status: mgclients.DeletedStatus})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	deletePolicies := []policies.Policy{
		{
			SubjectType: policies.DomainType,
			Subject:     session.DomainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		},
	}

	if ch.ParentGroup != "" {
		deletePolicies = append(deletePolicies, policies.Policy{
			SubjectType: policies.GroupType,
			Subject:     ch.ParentGroup,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		})
	}

	filterDeletePolicies := []policies.Policy{
		{
			SubjectType: policies.ChannelType,
			Subject:     id,
		},
		{
			ObjectType: policies.ChannelType,
			Object:     id,
		},
	}

	if err := svc.RemoveEntitiesRoles(ctx, session.DomainID, session.DomainUserID, []string{id}, filterDeletePolicies, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.repo.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) Connect(ctx context.Context, session authn.Session, chIDs, thIDs []string) (retErr error) {
	for _, chID := range chIDs {
		c, err := svc.repo.RetrieveByID(ctx, chID)
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if c.Status != mgclients.EnabledStatus {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("channel id %s is not in enabled state", chID))
		}
		if c.Domain != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("channel id %s has invalid domain id", chID))
		}
	}

	for _, thID := range thIDs {
		resp, err := svc.things.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: thID})
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if resp.GetEntity().GetStatus() != uint32(mgclients.EnabledStatus) {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("thing id %s is not in enabled state", thID))
		}
		if resp.GetEntity().GetDomainId() != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("thing id %s has invalid domain id", thID))
		}
	}

	conns := []Connection{}
	thConns := []*grpcCommonV1.Connection{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			conns = append(conns, Connection{
				ThingID:   thID,
				ChannelID: chID,
				DomainID:  session.DomainID,
			})
			thConns = append(thConns, &grpcCommonV1.Connection{
				ThingId:   thID,
				ChannelId: chID,
				DomainId:  session.DomainID,
			})
		}
	}

	for _, conn := range conns {
		err := svc.repo.CheckConnection(ctx, conn)

		switch {
		case err == nil:
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("channel %s and thing %s are already connected in domain %s", conn.ChannelID, conn.ThingID, conn.DomainID))
		case err != repoerr.ErrNotFound:
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
	}
	if _, err := svc.things.AddConnections(ctx, &grpcCommonV1.AddConnectionsReq{Connections: thConns}); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, errors.Wrap(errAddConnectionsThings, err))
	}

	if err := svc.repo.AddConnections(ctx, conns); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) Disconnect(ctx context.Context, session authn.Session, chIDs, thIDs []string) (retErr error) {
	for _, chID := range chIDs {
		c, err := svc.repo.RetrieveByID(ctx, chID)
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		if c.Domain != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("channel id %s has invalid domain id", chID))
		}
	}

	for _, thID := range thIDs {
		resp, err := svc.things.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: thID})
		if err != nil {
			return errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		if resp.GetEntity().GetDomainId() != session.DomainID {
			return errors.Wrap(svcerr.ErrCreateEntity, fmt.Errorf("thing id %s has invalid domain id", thID))
		}
	}

	conns := []Connection{}
	thConns := []*grpcCommonV1.Connection{}
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			conns = append(conns, Connection{
				ThingID:   thID,
				ChannelID: chID,
				DomainID:  session.DomainID,
			})
			thConns = append(thConns, &grpcCommonV1.Connection{
				ThingId:   thID,
				ChannelId: chID,
				DomainId:  session.DomainID,
			})
		}
	}

	if _, err := svc.things.RemoveConnections(ctx, &grpcCommonV1.RemoveConnectionsReq{Connections: thConns}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, errors.Wrap(errRemoveConnectionsThings, err))
	}

	if err := svc.repo.RemoveConnections(ctx, conns); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (retErr error) {
	ch, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	resp, err := svc.groups.RetrieveEntity(ctx, &grpcCommonV1.RetrieveEntityReq{Id: parentGroupID})
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if resp.GetEntity().GetDomainId() != session.DomainID {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s has invalid domain id", parentGroupID))
	}
	if resp.GetEntity().GetStatus() != uint32(mgclients.EnabledStatus) {
		return errors.Wrap(svcerr.ErrUpdateEntity, fmt.Errorf("parent group id %s is not in enabled state", parentGroupID))
	}

	var pols []policies.Policy
	if ch.ParentGroup != "" {
		return errors.Wrap(svcerr.ErrConflict, errSetParentGroup)
	}
	pols = append(pols, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     parentGroupID,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.ChannelType,
		Object:      id,
	})

	if err := svc.policy.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()
	ch = Channel{ID: id, ParentGroup: parentGroupID, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

	if err := svc.repo.SetParentGroup(ctx, ch); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (retErr error) {
	ch, err := svc.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if ch.ParentGroup != "" {
		var pols []policies.Policy
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     ch.ParentGroup,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.ChannelType,
			Object:      id,
		})

		if err := svc.policy.DeletePolicies(ctx, pols); err != nil {
			return errors.Wrap(svcerr.ErrAddPolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policy.AddPolicies(ctx, pols); errRollback != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
				}
			}
		}()

		ch := Channel{ID: id, UpdatedBy: session.UserID, UpdatedAt: time.Now()}

		if err := svc.repo.RemoveParentGroup(ctx, ch); err != nil {
			return err
		}
	}

	return nil
}

func (svc service) listChannelIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policy.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ChannelType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (svc service) retrievePermissions(ctx context.Context, userID string, channel *Channel) error {
	permissions, err := svc.listUserThingPermission(ctx, userID, channel.ID)
	if err != nil {
		return err
	}
	channel.Permissions = permissions
	return nil
}

func (svc service) listUserThingPermission(ctx context.Context, userID, thingID string) ([]string, error) {
	lp, err := svc.policy.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      thingID,
		ObjectType:  policies.ChannelType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return lp, nil
}

func (svc service) changeChannelStatus(ctx context.Context, userID string, channel Channel) (Channel, error) {
	dbchannel, err := svc.repo.RetrieveByID(ctx, channel.ID)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbchannel.Status == channel.Status {
		return Channel{}, errors.ErrStatusAlreadyAssigned
	}

	channel.UpdatedBy = userID

	channel, err = svc.repo.ChangeStatus(ctx, channel)
	if err != nil {
		return Channel{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return channel, nil
}
