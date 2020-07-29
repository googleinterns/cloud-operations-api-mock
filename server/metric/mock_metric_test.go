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
	"fmt"
	"log"
	"math/rand"
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"

	"google.golang.org/genproto/googleapis/api/label"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	st "google.golang.org/grpc/status"
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

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func generateMetricDescriptor() *metric.MetricDescriptor {
	return &metric.MetricDescriptor{
		Type:        fmt.Sprintf("custom.googleapis.com/opentelemetry/%v", rand.Intn(10000)),
		DisplayName: "opentelemetry/test-instrument",
		Description: "test-description",
		Labels: []*label.LabelDescriptor{
			{
				Key:       "testkey",
				ValueType: label.LabelDescriptor_STRING,
			},
		},
		MetricKind: metric.MetricDescriptor_GAUGE,
		ValueType:  metric.MetricDescriptor_DOUBLE,
	}
}

func TestMockMetricServer_DuplicateMetricLabelError(t *testing.T) {
	setup()
	defer tearDown()

	md := &metric.MetricDescriptor{
		Type:        "custom.googleapis.com/opentelemetry/1",
		DisplayName: "opentelemetry/test-instrument",
		Description: "test-description",
		Labels: []*label.LabelDescriptor{
			{
				Key:       "testkey",
				ValueType: label.LabelDescriptor_STRING,
			},
			{
				Key:       "testkey",
				ValueType: label.LabelDescriptor_STRING,
			},
		},
		MetricKind: metric.MetricDescriptor_GAUGE,
		ValueType:  metric.MetricDescriptor_DOUBLE,
	}

	in := &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}
	want := codes.InvalidArgument
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("CreateMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}
}

func TestMockMetricServer_MissingValueTypeInMetricType(t *testing.T) {
	setup()
	defer tearDown()
	missingFields := map[string]struct{}{"ValueType": {}}
	md := &metric.MetricDescriptor{
		Type:        fmt.Sprintf("custom.googleapis.com/opentelemetry/%v", rand.Intn(10000)),
		DisplayName: "opentelemetry/test-instrument",
		Description: "test-description",
		Labels: []*label.LabelDescriptor{
			{
				Key:       "testkeymissingvaluetype",
				ValueType: label.LabelDescriptor_STRING,
			},
		},
		MetricKind: metric.MetricDescriptor_GAUGE,
	}
	in := &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}
	want := codes.InvalidArgument
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("CreateMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func generateTimeSeries(metricType string, resourceType string, metricKind metric.MetricDescriptor_MetricKind,
	startTime *timestamp.Timestamp, endTime *timestamp.Timestamp) *monitoring.TimeSeries {
	return &monitoring.TimeSeries{
		Metric:     &metric.Metric{Type: metricType},
		Resource:   &monitoredres.MonitoredResource{Type: resourceType},
		MetricKind: metricKind,
		Points: []*monitoring.Point{
			{
				Interval: &monitoring.TimeInterval{
					StartTime: startTime, EndTime: endTime,
				},
			},
		},
	}
}

func TestMockMetricServer_CreateTimeSeries_Gauge(t *testing.T) {
	setup()
	defer tearDown()

	metricType := "custom.googleapis.com/opentelemetry/test-1"

	// Create the corresponding MetricDescriptor.
	_, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name: "projects/test-project",
		MetricDescriptor: &metric.MetricDescriptor{
			Type:        metricType,
			DisplayName: "opentelemetry/test-instrument",
			Description: "test-description",
			Labels: []*label.LabelDescriptor{
				{
					Key:       "testkey",
					ValueType: label.LabelDescriptor_STRING,
				},
			},
			MetricKind: metric.MetricDescriptor_GAUGE,
			ValueType:  metric.MetricDescriptor_DOUBLE,
		},
	})
	if err != nil {
		t.Fatalf("failed to call CreateMetricDescriptor %v", err)
	}

	// Create the TimeSeries.
	pointTime := ptypes.TimestampNow()
	in := &monitoring.CreateTimeSeriesRequest{
		Name: "projects/test-project",
		TimeSeries: []*monitoring.TimeSeries{
			generateTimeSeries(metricType, "test-monitored-resource",
				metric.MetricDescriptor_GAUGE, pointTime, pointTime),
		},
	}

	_, err = client.CreateTimeSeries(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateTimeSeries %v", err)
	}
}

func TestMockMetricServer_CreateTimeSeries_RateLimit(t *testing.T) {
	setup()
	defer tearDown()
	metricType := "custom.googleapis.com/opentelemetry/test-1"

	// Create the corresponding MetricDescriptor.
	_, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name: "projects/test-project",
		MetricDescriptor: &metric.MetricDescriptor{
			Type:        metricType,
			DisplayName: "opentelemetry/test-instrument",
			Description: "test-description",
			Labels: []*label.LabelDescriptor{
				{
					Key:       "testkey",
					ValueType: label.LabelDescriptor_STRING,
				},
			},
			MetricKind: metric.MetricDescriptor_GAUGE,
			ValueType:  metric.MetricDescriptor_DOUBLE,
		},
	})
	if err != nil {
		t.Fatalf("failed to call CreateMetricDescriptor %v", err)
	}

	// Create a point for the time series.
	pointTime := ptypes.TimestampNow()
	in := &monitoring.CreateTimeSeriesRequest{
		Name: "projects/test-project",
		TimeSeries: []*monitoring.TimeSeries{
			generateTimeSeries(metricType, "test-monitored-resource",
				metric.MetricDescriptor_GAUGE, pointTime, pointTime),
		},
	}
	_, err = client.CreateTimeSeries(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateTimeSeries %v", err)
	}

	// Try to create another point for the same TimeSeries before 10 seconds have elapsed.
	want := codes.Aborted
	_, err = client.CreateTimeSeries(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Error("Expected rate limit exceeded, instead got success")
	}
}

func TestMockMetricServer_ListTimeSeries(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.ListTimeSeriesRequest{
		Name:     "projects/test-project",
		Filter:   "",
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
	setup()
	defer tearDown()

	in := &monitoring.GetMonitoredResourceDescriptorRequest{
		Name: "test-resource-type",
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
	setup()
	defer tearDown()

	in := &monitoring.ListMonitoredResourceDescriptorsRequest{
		Name: "projects/test-project",
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
	setup()
	defer tearDown()
	md := generateMetricDescriptor()

	in := &monitoring.GetMetricDescriptorRequest{
		Name: md.Type,
	}
	want := md

	if _, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}); err != nil {
		t.Fatalf("failed to create test metric descriptor with error: %v", err)
	}

	response, err := client.GetMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call GetMetricDescriptor %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("GetMetricDescriptor(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_CreateMetricDescriptor(t *testing.T) {
	setup()
	defer tearDown()
	md := generateMetricDescriptor()

	in := &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}
	want := md
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateMetricDescriptor: %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("CreateMetricDescriptor(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_DeleteMetricDescriptor(t *testing.T) {
	setup()
	defer tearDown()
	md := generateMetricDescriptor()

	in := &monitoring.DeleteMetricDescriptorRequest{
		Name: md.Type,
	}
	want := &empty.Empty{}

	if _, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}); err != nil {
		t.Fatalf("failed to create test metric descriptor with error: %v", err)
	}

	response, err := client.DeleteMetricDescriptor(ctx, in)
	if err != nil {
		t.Fatalf("failed to call DeleteMetricDescriptorRequest: %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("DeleteMetricDescriptorRequest(%q) == %q, want %q", in, response, want)
	}
}

func TestMockMetricServer_ListMetricDescriptors(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.ListMetricDescriptorsRequest{
		Name: "projects/test-project",
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

func TestMockMetricServer_GetMetricDescriptor_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.GetMetricDescriptorRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.GetMetricDescriptor(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("GetMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_GetMonitoredResourceDescriptor_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.GetMonitoredResourceDescriptorRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.GetMonitoredResourceDescriptor(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("GetMonitoredResourceDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_GetMetricDescriptor_NotFoundError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.GetMetricDescriptorRequest{
		Name: "test",
	}
	want := codes.NotFound
	response, err := client.GetMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("GetMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if s := st.Convert(err); s.Code() != want {
		t.Errorf("GetMetricDescriptor(%q) returned error %q, expected error %q",
			in, s.Message(), want)
	}
}

func TestMockMetricServer_DeleteMetricDescriptor_NotFoundError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.DeleteMetricDescriptorRequest{
		Name: "test",
	}
	want := codes.NotFound
	response, err := client.DeleteMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("DeleteMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if s := st.Convert(err); s.Code() != want {
		t.Errorf("DeleteMetricDescriptor(%q) returned error %q, expected error %q",
			in, s.Message(), want)
	}
}

func TestMockMetricServer_MetricDescriptor_DataRace(t *testing.T) {
	setup()
	defer tearDown()
	md1 := generateMetricDescriptor()
	md2 := generateMetricDescriptor()

	errChan := make(chan error)
	go func() {
		_, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
			Name:             "projects/test-project",
			MetricDescriptor: md1,
		})
		errChan <- err
	}()

	_, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md2,
	})
	if err != nil {
		t.Fatalf("failed to call CreateMetricDescriptor: %v", err)
	}

	// Wait for goroutine to finish.
	if err := <-errChan; err != nil {
		t.Fatalf("failed to call CreateMetricDescriptor: %v", err)
	}
}

func TestMockMetricServer_DuplicateMetricDescriptorError(t *testing.T) {
	setup()
	defer tearDown()
	duplicateDescriptorType := "custom.googleapis.com/opentelemetry/1"
	md := &metric.MetricDescriptor{
		Type:        duplicateDescriptorType,
		DisplayName: "opentelemetry/test-instrument",
		Description: "test-description",
		Labels: []*label.LabelDescriptor{
			{
				Key:       "testkey",
				ValueType: label.LabelDescriptor_STRING,
			},
		},
		MetricKind: metric.MetricDescriptor_GAUGE,
		ValueType:  metric.MetricDescriptor_DOUBLE,
	}

	if _, err := client.CreateMetricDescriptor(ctx, &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}); err != nil {
		t.Fatalf("failed to create test metric descriptor with error: %v", err)
	}

	in := &monitoring.CreateMetricDescriptorRequest{
		Name:             "projects/test-project",
		MetricDescriptor: md,
	}
	want := codes.AlreadyExists
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err == nil {
		t.Errorf("CreateMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateDuplicateErrDetails(err, duplicateDescriptorType); !valid {
		t.Errorf("expected duplicate metric descriptor type: %v", duplicateDescriptorType)
	}
}

func TestMockMetricServer_DeleteMetricDescriptor_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.DeleteMetricDescriptorRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.DeleteMetricDescriptor(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("DeleteMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListMetricDescriptor_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.ListMetricDescriptorsRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.ListMetricDescriptors(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("ListMetricDescriptors(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_CreateMetricDescriptor_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.CreateMetricDescriptorRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}, "MetricDescriptor": {}}
	response, err := client.CreateMetricDescriptor(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("CreateMetricDescriptor(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListMonitoredResourceDescriptors_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.ListMonitoredResourceDescriptorsRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}}
	response, err := client.ListMonitoredResourceDescriptors(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("ListMonitoredResourceDescriptors(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_ListTimeSeries_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.ListTimeSeriesRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}, "Filter": {}, "View": {}, "Interval": {}}
	response, err := client.ListTimeSeries(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("ListTimeSeries(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}

func TestMockMetricServer_CreateTimeSeries_MissingFieldsError(t *testing.T) {
	setup()
	defer tearDown()

	in := &monitoring.CreateTimeSeriesRequest{}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{"Name": {}, "TimeSeries": {}}
	response, err := client.CreateTimeSeries(ctx, in)
	if err == nil || st.Code(err) != want {
		t.Errorf("CreateTimeSeries(%q) == %q, expected error %q", in, response, want)
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("Expected missing fields %q", missingFields)
	}
}
