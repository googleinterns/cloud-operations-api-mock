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
	"context"
	"log"
	"net"
	"time"

	"github.com/googleinterns/cloud-operations-api-mock/server/metric"
	"github.com/googleinterns/cloud-operations-api-mock/server/trace"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
)

// CloudMock is the struct we will expose to users to use in their tests.
// It contains the gRPC clients for users to call, as well as the connection
// info to allow a graceful shutdown after the tests run.
type CloudMock struct {
	conn                *grpc.ClientConn
	grpcServer          *grpc.Server
	mockTraceServer     *trace.MockTraceServer
	TraceServiceClient  cloudtrace.TraceServiceClient
	MetricServiceClient monitoring.MetricServiceClient
}

func startMockServer() (string, *grpc.Server, *trace.MockTraceServer) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("mock server failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	mockTrace := trace.NewMockTraceServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, mockTrace)
	monitoring.RegisterMetricServiceServer(grpcServer, metric.NewMockMetricServer())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("mock server failed to serve: %v", err)
		}
	}()

	return lis.Addr().String(), grpcServer, mockTrace
}

// NewCloudMock is the constructor for the CloudMock struct, it will return a
// pointer to a new CloudMock.
func NewCloudMock() *CloudMock {
	address, grpcServer, mockTrace := startMockServer()

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}

	traceClient := cloudtrace.NewTraceServiceClient(conn)
	metricClient := monitoring.NewMetricServiceClient(conn)
	return &CloudMock{
		conn,
		grpcServer,
		mockTrace,
		traceClient,
		metricClient,
	}
}

// ClientConn is a getter to retrieve the client connection. This is used
// to provide the exporters with the address of our mock server.
func (mock *CloudMock) ClientConn() *grpc.ClientConn {
	return mock.conn
}

// GetNumSpans returns the number of spans currently stored on the server.
func (mock *CloudMock) GetNumSpans() int {
	return mock.mockTraceServer.GetNumSpans()
}

// GetSpan returns the span that was stored in memory at the given index.
func (mock *CloudMock) GetSpan(index int) *cloudtrace.Span {
	return mock.mockTraceServer.GetSpan(index)
}

// SetDelay allows users to set the amount of time to delay before
// writing spans to memory.
func (mock *CloudMock) SetDelay(delay time.Duration) {
	mock.mockTraceServer.SetDelay(delay)
}

// SetOnUpload allows users to set the onUpload function on the mock server,
// which is called before BatchWriteSpans runs.
func (mock *CloudMock) SetOnUpload(onUpload func(ctx context.Context, spans []*cloudtrace.Span)) {
	mock.mockTraceServer.SetOnUpload(onUpload)
}

// Shutdown closes the connections and shuts down the gRPC server.
func (mock *CloudMock) Shutdown() {
	if err := mock.conn.Close(); err != nil {
		log.Fatalf("failed to close connection: %s", err)
	}
	mock.grpcServer.GracefulStop()
}
