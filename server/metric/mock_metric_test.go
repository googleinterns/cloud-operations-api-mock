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
	"strings"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const bufSize = 1024 * 1024

var (
	client     monitoring.MetricServiceClient
	conn       *grpc.ClientConn
	ctx        context.Context
	grpcServer *grpc.Server
	lis        *bufconn.Listener
)

func setup() {
	// Setup the in-memory server.
	lis = bufconn.Listen(bufSize)
	grpcServer = grpc.NewServer()
	monitoring.RegisterMetricServiceServer(grpcServer, NewMockMetricServer())
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
	in := &monitoring.CreateTimeSeriesRequest{
		Name:       "test create time series request",
		TimeSeries: []*monitoring.TimeSeries{&monitoring.TimeSeries{}},
	}
	want := &empty.Empty{}
	response, err := client.CreateTimeSeries(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateTimeSeries %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("CreateTimeSeries(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_ListTimeSeries(t *testing.T) {
	in := &monitoring.ListTimeSeriesRequest{
		Name:     "test list time series request",
		Filter:   "test filter",
		Interval: &monitoring.TimeInterval{},
		View:     monitoring.ListTimeSeriesRequest_HEADERS,
	}
	want := &monitoring.ListTimeSeriesResponse{
		TimeSeries:      []*monitoring.TimeSeries{},
		NextPageToken:   "",
		ExecutionErrors: []*status.Status{},
	}

	response, err := client.ListTimeSeries(ctx, in)
	if err != nil {
		t.Fatalf("failed to call ListTimeSeries %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("ListTimeSeries(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_GetMonitoredResourceDescriptor(t *testing.T) {
	in := &monitoring.GetMonitoredResourceDescriptorRequest{
		Name: "test get metric monitored resource descriptor",
	}
	want := &monitoredres.MonitoredResourceDescriptor{}
	response, err := client.GetMonitoredResourceDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call GetMonitoredResourceDescriptor %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("GetMonitoredResourceDescriptor(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_ListMonitoredResourceDescriptors(t *testing.T) {
	in := &monitoring.ListMonitoredResourceDescriptorsRequest{
		Name: "test list monitored resource descriptors",
	}
	want := &monitoring.ListMonitoredResourceDescriptorsResponse{
		ResourceDescriptors: []*monitoredres.MonitoredResourceDescriptor{},
	}
	response, err := client.ListMonitoredResourceDescriptors(ctx, in)
	if err != nil {
		t.Fatalf("failed to call ListMonitoredResourceDescriptors %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("ListMonitoredResourceDescriptors(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_GetMetricDescriptor(t *testing.T) {
	in := &monitoring.GetMetricDescriptorRequest{
		Name: "test get metric descriptor",
	}
	want := &metric.MetricDescriptor{}
	response, err := client.GetMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call GetMetricDescriptor %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("GetMetricDescriptor(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_CreateMetricDescriptor(t *testing.T) {
	in := &monitoring.CreateMetricDescriptorRequest{
		Name:             "test create metric descriptor",
		MetricDescriptor: &metric.MetricDescriptor{},
	}
	want := &metric.MetricDescriptor{}
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateMetricDescriptorRequest: %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("CreateMetricDescriptorRequest(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_DeleteMetricDescriptor(t *testing.T) {
	in := &monitoring.DeleteMetricDescriptorRequest{
		Name: "test create metric descriptor",
	}
	want := &empty.Empty{}

	response, err := client.DeleteMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call DeleteMetricDescriptorRequest: %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("DeleteMetricDescriptorRequest(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_ListMetricDescriptors(t *testing.T) {
	in := &monitoring.ListMetricDescriptorsRequest{
		Name: "test list metric decriptors request",
	}
	want := &monitoring.ListMetricDescriptorsResponse{
		MetricDescriptors: []*metric.MetricDescriptor{},
	}
	response, err := client.ListMetricDescriptors(ctx, in)
	if err != nil {
		t.Fatalf("failed to call ListMetricDescriptors %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("ListMetricDescriptors(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_GetMetricDescriptorError(t *testing.T) {
	in := &monitoring.GetMetricDescriptorRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.GetMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("GetMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("GetMetricDescriptor(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_GetMonitoredResourceDescriptorError(t *testing.T) {
	in := &monitoring.GetMonitoredResourceDescriptorRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.GetMonitoredResourceDescriptor(ctx, in)
	if err == nil {
		t.Errorf("GetMonitoredResourceDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("GetMonitoredResourceDescriptor(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_DeleteMetricDescriptorError(t *testing.T) {
	in := &monitoring.DeleteMetricDescriptorRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.DeleteMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("DeleteMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("DeleteMetricDescriptor(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListMetricDescriptorError(t *testing.T) {
	in := &monitoring.ListMetricDescriptorsRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.ListMetricDescriptors(ctx, in)
	if err == nil {
		t.Errorf("ListMetricDescriptors(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("ListMetricDescriptors(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_CreateMetricDescriptorError(t *testing.T) {
	in := &monitoring.CreateMetricDescriptorRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}, "MetricDescriptor": {}}
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("CreateMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("CreateMetricDescriptor(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListMonitoredResourceDescriptorsError(t *testing.T) {
	in := &monitoring.ListMonitoredResourceDescriptorsRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.ListMonitoredResourceDescriptors(ctx, in)
	if err == nil {
		t.Errorf("ListMonitoredResourceDescriptors(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("ListMonitoredResourceDescriptors(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListTimeSeriesError(t *testing.T) {
	in := &monitoring.ListTimeSeriesRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}, "Filter": {}, "View": {}, "Interval": {}}
	response, err := client.ListTimeSeries(ctx, in)
	if err == nil {
		t.Errorf("ListTimeSeries(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("ListTimeSeries(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_CreateTimeSeriesError(t *testing.T) {
	in := &monitoring.CreateTimeSeriesRequest{}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{"Name": {}, "TimeSeries": {}}
	response, err := client.CreateTimeSeries(ctx, in)
	if err == nil {
		t.Errorf("CreateTimeSeries(%q) == %q, expected error %q", in, response, want)
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("CreateTimeSeries(%q) returned error %q, expected error %q",
			in, err.Error(), want)
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}
