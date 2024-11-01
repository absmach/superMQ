// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authorizeReq struct {
	domainID   string
	channelID  string
	clientID   string
	clientType string
	permission string
}
type removeThingConnectionsReq struct {
	thingID string
}

type unsetParentGroupFromChannelsReq struct {
	parentGroupID string
}
