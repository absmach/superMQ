// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

type thingBasic struct {
	id     string
	domain string
	status uint8
}

type authenticateRes struct {
	id            string
	authenticated bool
}

type retrieveEntitiesRes struct {
	total  uint64
	limit  uint64
	offset uint64
	things []thingBasic
}

type retrieveEntityRes thingBasic

type connectionsReq struct {
	connections []connection
}

type connection struct {
	thingID   string
	channelID string
	domainID  string
}
type connectionsRes struct {
	ok bool
}

type removeChannelConnectionsRes struct{}

type unsetParentGroupFromThingsRes struct{}
