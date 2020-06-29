// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudmock

import (
	"log"
	"net"

	mocktrace "github.com/googleinterns/cloud-operations-api-mock/api"
	"github.com/googleinterns/cloud-operations-api-mock/server/metric"
	"github.com/googleinterns/cloud-operations-api-mock/server/trace"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
)

type CloudMock struct {
	conn                   *grpc.ClientConn
	grpcServer             *grpc.Server
	TraceServiceClient     cloudtrace.TraceServiceClient
	MockTraceServiceClient mocktrace.MockTraceServiceClient
	MetricServiceClient    monitoring.MetricServiceClient
}

func startMockServer() (string, *grpc.Server) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	mockTrace := trace.NewMockTraceServer()

	cloudtrace.RegisterTraceServiceServer(grpcServer, mockTrace)
	mocktrace.RegisterMockTraceServiceServer(grpcServer, mockTrace)
	monitoring.RegisterMetricServiceServer(grpcServer, &metric.MockMetricServer{})

	log.Printf("Listening on %s\n", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("mock server failed to serve: %v", err)
		}
	}()

	return lis.Addr().String(), grpcServer
}

func NewCloudMock() *CloudMock {
	address, grpcServer := startMockServer()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}

	traceClient := cloudtrace.NewTraceServiceClient(conn)
	mockTraceClient := mocktrace.NewMockTraceServiceClient(conn)
	metricClient := monitoring.NewMetricServiceClient(conn)
	return &CloudMock{
		conn,
		grpcServer,
		traceClient,
		mockTraceClient,
		metricClient,
	}
}

func (mock *CloudMock) ClientConn() *grpc.ClientConn {
	return mock.conn
}

func (mock *CloudMock) Shutdown() {
	if err := mock.conn.Close(); err != nil {
		log.Fatalf("failed to close connection: %s", err)
	}
	mock.grpcServer.GracefulStop()
}
