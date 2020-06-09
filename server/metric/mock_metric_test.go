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

package metric

import (
	"context"
	"log"
	"net"
	"os"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const bufSize = 1024 * 1024

var (
	client monitoring.MetricServiceClient
	conn   *grpc.ClientConn
	ctx    context.Context
	grpcServer *grpc.Server
	lis    *bufconn.Listener
)

func setup() {
	// Setup the in-memory server.
	lis = bufconn.Listen(bufSize)
	grpcServer = grpc.NewServer()
	monitoring.RegisterMetricServiceServer(grpcServer, &MockMetricServer{})
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

	// Setup the connection and client.
	ctx = context.Background()
	var err error
	conn, err = grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial bufnet: %v", err)
	}
	client = monitoring.NewMetricServiceClient(conn)
}

func tearDown() {
	conn.Close()
	grpcServer.GracefulStop()
}

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	tearDown()
	os.Exit(retCode)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestMockMetricServer_CreateTimeSeries(t *testing.T) {
	cases := []struct {
		in *monitoring.CreateTimeSeriesRequest
		want *empty.Empty
	}{
		{
			&monitoring.CreateTimeSeriesRequest{
				Name: "test create time series request",
			},
			&empty.Empty{},
		},
	}

	for _, c := range cases {
		response, err := client.CreateTimeSeries(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call CreateTimeSeries %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("CreateTimeSeries(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_ListTimeSeries(t *testing.T) {
	cases := []struct {
		in *monitoring.ListTimeSeriesRequest
		want *monitoring.ListTimeSeriesResponse
	}{
		{
			&monitoring.ListTimeSeriesRequest{
				Name: "test list time series request",
			},
			&monitoring.ListTimeSeriesResponse{
				TimeSeries:      []*monitoring.TimeSeries{},
				NextPageToken:   "",
				ExecutionErrors: []*status.Status{},
			},
		},
	}

	for _, c := range cases {
		response, err := client.ListTimeSeries(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call ListTimeSeries %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("ListTimeSeries(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_GetMonitoredResourceDescriptor(t *testing.T) {
	cases := []struct {
		in *monitoring.GetMonitoredResourceDescriptorRequest
		want *monitoredres.MonitoredResourceDescriptor
	}{
		{
			&monitoring.GetMonitoredResourceDescriptorRequest{
				Name: "test get metric monitored resource descriptor",
			},
			&monitoredres.MonitoredResourceDescriptor{},
		},
	}

	for _, c := range cases {
		response, err := client.GetMonitoredResourceDescriptor(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call GetMonitoredResourceDescriptor %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("GetMonitoredResourceDescriptor(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}


func TestMockMetricServer_ListMonitoredResourceDescriptors(t *testing.T) {
	cases := []struct {
		in *monitoring.ListMonitoredResourceDescriptorsRequest
		want *monitoring.ListMonitoredResourceDescriptorsResponse
	}{
		{
			&monitoring.ListMonitoredResourceDescriptorsRequest{
				Name: "test list monitored resource descriptors",
			},
			&monitoring.ListMonitoredResourceDescriptorsResponse{
				ResourceDescriptors: []*monitoredres.MonitoredResourceDescriptor{},
			},
		},
	}

	for _, c := range cases {
		response, err := client.ListMonitoredResourceDescriptors(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call ListMonitoredResourceDescriptors %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("ListMonitoredResourceDescriptors(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_GetMetricDescriptor(t *testing.T) {
	cases := []struct {
		in *monitoring.GetMetricDescriptorRequest
		want *metric.MetricDescriptor
	}{
		{
			&monitoring.GetMetricDescriptorRequest{
				Name: "test get metric descriptor",
			},
			&metric.MetricDescriptor{},
		},
	}

	for _, c := range cases {
		response, err := client.GetMetricDescriptor(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call GetMetricDescriptor %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("GetMetricDescriptor(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_CreateMetricDescriptor(t *testing.T) {
	cases := []struct {
		in *monitoring.CreateMetricDescriptorRequest
		want *metric.MetricDescriptor
	}{
		{
			&monitoring.CreateMetricDescriptorRequest{
				Name: "test create metric descriptor",
				MetricDescriptor: &metric.MetricDescriptor{},
			},
			&metric.MetricDescriptor{},
		},
	}

	for _, c := range cases {
		response, err := client.CreateMetricDescriptor(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call CreateMetricDescriptorRequest: %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("CreateMetricDescriptorRequest(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_DeleteMetricDescriptor(t *testing.T) {
	cases := []struct {
		in *monitoring.DeleteMetricDescriptorRequest
		want *empty.Empty
	}{
		{
			&monitoring.DeleteMetricDescriptorRequest{
				Name: "test create metric descriptor",
			},
			&empty.Empty{},
		},
	}

	for _, c := range cases {
		response, err := client.DeleteMetricDescriptor(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call DeleteMetricDescriptorRequest: %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("DeleteMetricDescriptorRequest(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockMetricServer_ListMetricDescriptors(t *testing.T) {
	cases := []struct {
		in *monitoring.ListMetricDescriptorsRequest
		want *monitoring.ListMetricDescriptorsResponse
	}{
		{
			&monitoring.ListMetricDescriptorsRequest{
				Name: "test list metric decriptors request",
			},
			&monitoring.ListMetricDescriptorsResponse{
				MetricDescriptors: []*metric.MetricDescriptor{},
				NextPageToken:     "",
			},
		},
	}

	for _, c := range cases {
		response, err := client.ListMetricDescriptors(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call ListMetricDescriptors %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("ListMetricDescriptors(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}