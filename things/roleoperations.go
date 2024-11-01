// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/svcutil"
)

// Internal Operations

const (
	OpViewThing svcutil.Operation = iota
	OpUpdateThing
	OpUpdateThingTags
	OpUpdateThingSecret
	OpEnableThing
	OpDisableThing
	OpDeleteThing
	OpSetParentGroup
	OpRemoveParentGroup
	OpConnectToChannel
	OpDisconnectFromChannel
)

var expectedOperations = []svcutil.Operation{
	OpViewThing,
	OpUpdateThing,
	OpUpdateThingTags,
	OpUpdateThingSecret,
	OpEnableThing,
	OpDisableThing,
	OpDeleteThing,
	OpSetParentGroup,
	OpRemoveParentGroup,
	OpConnectToChannel,
	OpDisconnectFromChannel,
}

var operationNames = []string{
	"OpViewThing",
	"OpUpdateThing",
	"OpUpdateThingTags",
	"OpUpdateThingSecret",
	"OpEnableThing",
	"OpDisableThing",
	"OpDeleteThing",
	"OpSetParentGroup",
	"OpRemoveParentGroup",
	"OpConnectToChannel",
	"OpDisconnectFromChannel",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

// External Operations
const (
	DomainOpCreateThing svcutil.ExternalOperation = iota
	DomainOpListThing
	GroupOpSetChildThing
	GroupsOpRemoveChildThing
	ChannelsOpConnectChannel
	ChannelsOpDisconnectChannel
)

var expectedExternalOperations = []svcutil.ExternalOperation{
	DomainOpCreateThing,
	DomainOpListThing,
	GroupOpSetChildThing,
	GroupsOpRemoveChildThing,
	ChannelsOpConnectChannel,
	ChannelsOpDisconnectChannel,
}

var externalOperationNames = []string{
	"DomainOpCreateThing",
	"DomainOpListThing",
	"GroupOpSetChildThing",
	"GroupsOpRemoveChildThing",
	"ChannelsOpConnectChannel",
	"ChannelsOpDisconnectChannel",
}

func NewExternalOperationPerm() svcutil.ExternalOperationPerm {
	return svcutil.NewExternalOperationPerm(expectedExternalOperations, externalOperationNames)
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	updatePermission           = "update_permission"
	readPermission             = "read_permission"
	deletePermission           = "delete_permission"
	setParentGroupPermission   = "set_parent_group_permission"
	connectToChannelPermission = "connect_to_channel_permission"

	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpViewThing:             readPermission,
		OpUpdateThing:           updatePermission,
		OpUpdateThingTags:       updatePermission,
		OpUpdateThingSecret:     updatePermission,
		OpEnableThing:           updatePermission,
		OpDisableThing:          updatePermission,
		OpDeleteThing:           deletePermission,
		OpSetParentGroup:        setParentGroupPermission,
		OpRemoveParentGroup:     setParentGroupPermission,
		OpConnectToChannel:      connectToChannelPermission,
		OpDisconnectFromChannel: connectToChannelPermission,
	}
	return opPerm
}

func NewRolesOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		roles.OpAddRole:                manageRolePermission,
		roles.OpRemoveRole:             manageRolePermission,
		roles.OpUpdateRoleName:         manageRolePermission,
		roles.OpRetrieveRole:           manageRolePermission,
		roles.OpRetrieveAllRoles:       manageRolePermission,
		roles.OpRoleAddActions:         manageRolePermission,
		roles.OpRoleListActions:        manageRolePermission,
		roles.OpRoleCheckActionsExists: manageRolePermission,
		roles.OpRoleRemoveActions:      manageRolePermission,
		roles.OpRoleRemoveAllActions:   manageRolePermission,
		roles.OpRoleAddMembers:         addRoleUsersPermission,
		roles.OpRoleListMembers:        viewRoleUsersPermission,
		roles.OpRoleCheckMembersExists: viewRoleUsersPermission,
		roles.OpRoleRemoveMembers:      removeRoleUsersPermission,
		roles.OpRoleRemoveAllMembers:   manageRolePermission,
	}
	return opPerm
}

const (
	// External Permission
	// Domains
	domainCreateThingPermission = "thing_create_permission"
	domainListThingPermission   = "list_things_permission"
	// Groups
	groupSetChildThingPermission    = "thing_create_permission"
	groupRemoveChildThingPermission = "thing_create_permission"
	// Channels
	channelsConnectThingPermission    = "connect_to_thing_permission"
	channelsDisconnectThingPermission = "connect_to_thing_permission"
)

func NewExternalOperationPermissionMap() map[svcutil.ExternalOperation]svcutil.Permission {
	extOpPerm := map[svcutil.ExternalOperation]svcutil.Permission{
		DomainOpCreateThing:         domainCreateThingPermission,
		DomainOpListThing:           domainListThingPermission,
		GroupOpSetChildThing:        groupSetChildThingPermission,
		GroupsOpRemoveChildThing:    groupRemoveChildThingPermission,
		ChannelsOpConnectChannel:    channelsConnectThingPermission,
		ChannelsOpDisconnectChannel: channelsDisconnectThingPermission,
	}
	return extOpPerm
}
