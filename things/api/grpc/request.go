// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type authenticateReq struct {
	ThingID  string
	ThingKey string
}

type retrieveEntitiesReq struct {
	Ids []string
}

type retrieveEntityReq struct {
	Id string
}

type removeChannelConnectionsReq struct {
	channelID string
}

type unsetParentGroupFromThingsReq struct {
	parentGroupID string
}
