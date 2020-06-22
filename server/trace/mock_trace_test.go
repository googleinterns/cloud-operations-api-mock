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
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/validation"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
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
	setup()
	defer tearDown()

	spans := []*cloudtrace.Span{
		generateSpan("test-span-1"),
		generateSpan("test-span-2"),
		generateSpan("test-span-3"),
	}
	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "test-project",
		Spans: spans,
	}
	want := &empty.Empty{}

	response, err := client.BatchWriteSpans(ctx, in)
	if err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
	}

	if !proto.Equal(response, want) {
		t.Errorf("BatchWriteSpans(%v) == %v, want %v", in, response, want)
	}
}

func TestMockTraceServer_BatchWriteSpans_MissingField(t *testing.T) {
	setup()
	defer tearDown()

	missingFieldsSpan := []*cloudtrace.Span{
		generateMissingFieldSpan("test-span-1", "Name", "StartTime"),
		generateMissingFieldSpan("test-span-2"),
	}
	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "test-project",
		Spans: missingFieldsSpan,
	}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{
		"Name":      {},
		"StartTime": {},
	}

	responseSpan, err := client.BatchWriteSpans(ctx, in)
	if err == nil {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
	}

	if valid := validateErrDetails(err, missingFields); !valid {
		t.Errorf("BatchWriteSpans(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_BatchWriteSpans_InvalidTimestamp(t *testing.T) {
	setup()
	defer tearDown()

	invalidTimestampSpans := []*cloudtrace.Span{
		generateInvalidTimestampSpan("test-span-1"),
		generateInvalidTimestampSpan("test-span-2"),
	}

	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "test-project",
		Spans: invalidTimestampSpans,
	}
	want := validation.ErrInvalidTimestamp

	responseSpan, err := client.BatchWriteSpans(ctx, in)
	if err == nil {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
	} else {
		if !strings.Contains(err.Error(), want.Error()) {
			t.Errorf("BatchWriteSpans(%v) returned error %v, expected error %v",
				in, err.Error(), want)
		}
	}
}

func TestMockTraceServer_CreateSpan(t *testing.T) {
	setup()
	defer tearDown()

	span := generateSpan("test-span-1")
	in, want := span, span

	responseSpan, err := client.CreateSpan(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateSpan: %v", err)
	}

	if !proto.Equal(responseSpan, want) {
		t.Errorf("CreateSpan(%v) == %v, want %v",
			in, responseSpan, want)
	}
}

func TestMockTraceServer_CreateSpan_MissingFields(t *testing.T) {
	setup()
	defer tearDown()

	in := generateMissingFieldSpan("test-span-1", "SpanId", "EndTime")
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{
		"SpanId":  {},
		"EndTime": {},
	}

	responseSpan, err := client.CreateSpan(ctx, in)
	if err == nil {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
	}

	if valid := validateErrDetails(err, missingFields); !valid {
		t.Errorf("CreateSpan(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_CreateSpan_InvalidTimestamp(t *testing.T) {
	setup()
	defer tearDown()

	in := generateInvalidTimestampSpan("test-span-1")
	want := validation.ErrInvalidTimestamp

	responseSpan, err := client.CreateSpan(ctx, in)
	if err == nil {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
	} else {
		if !strings.Contains(err.Error(), want.Error()) {
			t.Errorf("CreateSpan(%v) returned error %v, expected error %v",
				in, err.Error(), want)
		}
	}
}

func validateErrDetails(err error, missingFields map[string]struct{}) bool {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if t, ok := detail.(*errdetails.BadRequest); ok {
			for _, violation := range t.GetFieldViolations() {
				if _, ok := missingFields[violation.GetField()]; !ok {
					return false
				}
			}
		}
	}
	return true
}
