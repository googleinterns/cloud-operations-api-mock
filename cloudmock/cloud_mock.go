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

	"github.com/googleinterns/cloud-operations-api-mock/server/metric"
	"github.com/googleinterns/cloud-operations-api-mock/server/trace"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
)

func StartMockServer(address string) *grpc.Server {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, &trace.MockTraceServer{})
	monitoring.RegisterMetricServiceServer(grpcServer, &metric.MockMetricServer{})

	log.Printf("Listening on %s\n", address)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("mock server failed to serve: %v", err)
		}
	}()

	return grpcServer
}

func ShutdownMockServer(grpcServer *grpc.Server) {
	grpcServer.GracefulStop()
}

func StartMockClients(address string) (*grpc.ClientConn, cloudtrace.TraceServiceClient, monitoring.MetricServiceClient) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}

	traceClient := cloudtrace.NewTraceServiceClient(conn)
	metricClient := monitoring.NewMetricServiceClient(conn)
	return conn, traceClient, metricClient
}

func ShutdownMockClients(conn *grpc.ClientConn) {
	if err := conn.Close(); err != nil {
		log.Fatalf("failed to close connection: %s", err)
	}
}
