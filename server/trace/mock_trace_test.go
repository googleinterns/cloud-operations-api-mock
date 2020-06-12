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

package trace

import (
	"context"
	"log"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/validation"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const bufSize = 1024 * 1024

var (
	client     cloudtrace.TraceServiceClient
	conn       *grpc.ClientConn
	ctx        context.Context
	grpcServer *grpc.Server
	lis        *bufconn.Listener
)

func setup() {
	// Setup the in-memory server.
	lis = bufconn.Listen(bufSize)
	grpcServer = grpc.NewServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, &MockTraceServer{})
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
	client = cloudtrace.NewTraceServiceClient(conn)
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

func generateSpan(spanName string) *cloudtrace.Span {
	startTime, err := ptypes.TimestampProto(time.Now().Add(-(time.Second * time.Duration(30))))
	if err != nil {
		log.Fatalf("failed to create span with error: %v", err)
	}
	endTime := ptypes.TimestampNow()

	return &cloudtrace.Span{
		Name:   spanName,
		SpanId: spanName,
		DisplayName: &cloudtrace.TruncatableString{
			Value:              spanName,
			TruncatedByteCount: 0,
		},
		StartTime: startTime,
		EndTime:   endTime,
	}
}

func generateInvalidTimestampSpan(spanName string) *cloudtrace.Span {
	invalidTimestampSpan := generateSpan(spanName)
	end, err := ptypes.Timestamp(invalidTimestampSpan.EndTime)
	if err != nil {
		log.Fatalf("failed to create span with error: %v", err)
	}
	start, err := ptypes.TimestampProto(end.Add(time.Second * time.Duration(30)))
	if err != nil {
		log.Fatalf("failed to create span with error: %v", err)
	}
	invalidTimestampSpan.StartTime = start
	return invalidTimestampSpan
}

func generateMissingFieldSpan(spanName string, missingFields ...string) *cloudtrace.Span {
	span := generateSpan(spanName)
	for _, field := range missingFields {
		removeField(span, field)
	}
	return span
}

func removeField(span *cloudtrace.Span, field string) {
	spanField := reflect.Indirect(reflect.ValueOf(span)).FieldByName(field)
	zeroValue := reflect.Zero(spanField.Type())
	spanField.Set(zeroValue)
}

func TestMockTraceServer_BatchWriteSpans(t *testing.T) {
	spans := []*cloudtrace.Span{
		generateSpan("test-span-1"),
		generateSpan("test-span-2"),
		generateSpan("test-span-3"),
	}

	cases := []struct {
		in   *cloudtrace.BatchWriteSpansRequest
		want *empty.Empty
	}{
		{
			&cloudtrace.BatchWriteSpansRequest{
				Name:  "test-project",
				Spans: spans,
			},
			&empty.Empty{},
		},
	}

	for _, c := range cases {
		response, err := client.BatchWriteSpans(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call BatchWriteSpans: %v", err)
		}

		if !proto.Equal(response, c.want) {
			t.Errorf("BatchWriteSpans(%q) == %q, want %q", c.in, response, c.want)
		}
	}
}

func TestMockTraceServer_BatchWriteSpans_Error(t *testing.T) {
	missingFieldsSpan := []*cloudtrace.Span{
		generateMissingFieldSpan("test-span-1", "Name", "StartTime"),
		generateMissingFieldSpan("test-span-2"),
	}

	invalidTimestampSpans := []*cloudtrace.Span{
		generateInvalidTimestampSpan("test-span-3"),
		generateInvalidTimestampSpan("test-span-4"),
	}

	cases := []struct {
		in            *cloudtrace.BatchWriteSpansRequest
		want          error
		missingFields map[string]struct{}
	}{
		{
			&cloudtrace.BatchWriteSpansRequest{
				Name:  "test-project",
				Spans: missingFieldsSpan,
			},
			validation.MissingFieldErr.Err(),
			map[string]struct{}{
				"Name":      {},
				"StartTime": {},
			},
		},
		{
			&cloudtrace.BatchWriteSpansRequest{
				Name:  "test-project",
				Spans: invalidTimestampSpans,
			},
			validation.InvalidTimestampErr,
			nil,
		},
	}

	for _, c := range cases {
		responseSpan, err := client.BatchWriteSpans(ctx, c.in)
		if err == nil {
			t.Errorf("CreateSpan(%q) == %q, expected error %q",
				c.in, responseSpan, c.want)
			continue
		}

		if !strings.Contains(err.Error(), c.want.Error()) {
			t.Errorf("CreateSpan(%q) returned error %q, expected error %q",
				c.in, err.Error(), c.want)
		}
		if c.missingFields != nil {
			if valid := validateErrDetails(err, c.missingFields); !valid {
				t.Errorf("Expected missing fields %q", c.missingFields)
			}
		}
	}
}

func TestMockTraceServer_CreateSpan(t *testing.T) {
	span := generateSpan("test-span-1")
	cases := []struct {
		in   *cloudtrace.Span
		want *cloudtrace.Span
	}{
		{span, span},
	}

	for _, c := range cases {
		responseSpan, err := client.CreateSpan(ctx, c.in)
		if err != nil {
			t.Fatalf("failed to call CreateSpan: %v", err)
		}

		if !proto.Equal(responseSpan, c.want) {
			t.Errorf("CreateSpan(%q) == %q, want %q", c.in, responseSpan, c.want)
		}
	}
}

func TestMockTraceServer_CreateSpan_Error(t *testing.T) {
	cases := []struct {
		in            *cloudtrace.Span
		want          error
		missingFields map[string]struct{}
	}{
		{
			generateMissingFieldSpan("test-span-1", "SpanId", "EndTime"),
			validation.MissingFieldErr.Err(),
			map[string]struct{}{
				"SpanId":  {},
				"EndTime": {},
			},
		},
		{
			generateInvalidTimestampSpan("test-span-2"),
			validation.InvalidTimestampErr,
			nil,
		},
	}

	for _, c := range cases {
		responseSpan, err := client.CreateSpan(ctx, c.in)
		if err == nil {
			t.Errorf("CreateSpan(%q) == %q, expected error %q",
				c.in, responseSpan, c.want)
			continue
		}

		if !strings.Contains(err.Error(), c.want.Error()) {
			t.Errorf("CreateSpan(%q) returned error %q, expected error %q",
				c.in, err.Error(), c.want)
		}
		if c.missingFields != nil {
			if valid := validateErrDetails(err, c.missingFields); !valid {
				t.Errorf("Expected missing fields %q", c.missingFields)
			}
		}
	}
}

func validateErrDetails(err error, missingFields map[string]struct{}) bool {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		switch t := detail.(type) {
		case *errdetails.BadRequest:
			for _, violation := range t.GetFieldViolations() {
				if _, ok := missingFields[violation.GetField()]; !ok {
					return false
				}
			}
		}
	}
	return true
}
