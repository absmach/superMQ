// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"fmt"
	"net"

	grpcThingsV1 "github.com/absmach/supermq/internal/grpc/things/v1"
	grpcapi "github.com/absmach/supermq/things/api/grpc"
	"github.com/absmach/supermq/things/private/mocks"
	"google.golang.org/grpc"
)

const port = 7000

var (
	thingID   = "testID"
	thingKey  = "testKey"
	channelID = "testID"
	invalid   = "invalid"
)

func startGRPCServer(svc *mocks.Service, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	grpcThingsV1.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}
