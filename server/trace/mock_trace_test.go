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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	mocktrace "github.com/googleinterns/cloud-operations-api-mock/api"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const (
	bufSize        = 1024 * 1024
	spanNameFormat = "projects/%v/traces/%v/spans/%v"
)

var (
	traceClient cloudtrace.TraceServiceClient
	mockClient  mocktrace.MockTraceServiceClient
	conn        *grpc.ClientConn
	ctx         context.Context
	grpcServer  *grpc.Server
	lis         *bufconn.Listener
)

func setup() {
	// Setup the in-memory server.
	lis = bufconn.Listen(bufSize)
	grpcServer = grpc.NewServer()
	mockTraceServer := NewMockTraceServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, mockTraceServer)
	mocktrace.RegisterMockTraceServiceServer(grpcServer, mockTraceServer)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

	// Setup the connection and clients.
	ctx = context.Background()
	var err error
	conn, err = grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial bufnet: %v", err)
	}
	traceClient = cloudtrace.NewTraceServiceClient(conn)
	mockClient = mocktrace.NewMockTraceServiceClient(conn)
}

func tearDown() {
	conn.Close()
	grpcServer.GracefulStop()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func generateSpan() *cloudtrace.Span {
	spanName := generateSpanName("test-project")
	startTime, err := ptypes.TimestampProto(time.Now().Add(-(time.Second * time.Duration(30))))
	if err != nil {
		log.Fatalf("failed to create span with error: %v", err)
	}
	endTime := ptypes.TimestampNow()

	return &cloudtrace.Span{
		Name:   spanName,
		SpanId: spanName[len(spanName)-16:],
		DisplayName: &cloudtrace.TruncatableString{
			Value:              spanName,
			TruncatedByteCount: 0,
		},
		StartTime: startTime,
		EndTime:   endTime,
	}
}

func generateInvalidTimestampSpan() *cloudtrace.Span {
	invalidTimestampSpan := generateSpan()
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

func generateMissingFieldSpan(missingFields ...string) *cloudtrace.Span {
	span := generateSpan()
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

func generateHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateSpanName(projectName string) string {
	traceId, err := generateHex(16)
	if err != nil {
		log.Fatalf("failed to generate span name: %v", err)
	}
	spanId, err := generateHex(8)
	if err != nil {
		log.Fatalf("failed to generate span name: %v", err)
	}
	return fmt.Sprintf(spanNameFormat, projectName, traceId, spanId)
}

func TestMockTraceServer_BatchWriteSpans(t *testing.T) {
	setup()
	defer tearDown()

	spans := []*cloudtrace.Span{
		generateSpan(),
		generateSpan(),
		generateSpan(),
	}
	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: spans,
	}
	want := &empty.Empty{}

	response, err := traceClient.BatchWriteSpans(ctx, in)
	if err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
		return
	}

	if !proto.Equal(response, want) {
		t.Errorf("BatchWriteSpans(%v) == %v, want %v", in, response, want)
	}
}

func TestMockTraceServer_BatchWriteSpans_MissingField(t *testing.T) {
	setup()
	defer tearDown()

	missingFieldsSpan := []*cloudtrace.Span{
		generateMissingFieldSpan("Name", "StartTime"),
		generateMissingFieldSpan(),
	}
	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: missingFieldsSpan,
	}
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{
		"Name":      {},
		"StartTime": {},
	}

	responseSpan, err := traceClient.BatchWriteSpans(ctx, in)
	if err == nil {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("BatchWriteSpans(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_BatchWriteSpans_InvalidTimestamp(t *testing.T) {
	setup()
	defer tearDown()

	invalidTimestampSpans := []*cloudtrace.Span{
		generateInvalidTimestampSpan(),
		generateInvalidTimestampSpan(),
	}

	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: invalidTimestampSpans,
	}
	want := validation.StatusInvalidTimestamp

	responseSpan, err := traceClient.BatchWriteSpans(ctx, in)
	if err == nil {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("BatchWriteSpans(%v) returned error %v, expected error %v",
			in, err.Error(), want)
	}
}

func TestMockTraceServer_BatchWriteSpans_DuplicateName(t *testing.T) {
	setup()
	defer tearDown()

	duplicateSpan := generateSpan()
	spans := []*cloudtrace.Span{
		duplicateSpan,
		duplicateSpan,
	}
	in := &cloudtrace.BatchWriteSpansRequest{Name: "projects/test-project", Spans: spans}
	want := validation.StatusDuplicateSpanName

	responseSpan, err := traceClient.BatchWriteSpans(ctx, in)
	if err == nil {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want.Err())
		return
	}

	if valid := validation.ValidateDuplicateSpanNames(err, duplicateSpan.Name); !valid {
		t.Errorf("expected duplicate spanName: %v", duplicateSpan.Name)
	}
}

func TestMockTraceServer_CreateSpan(t *testing.T) {
	setup()
	defer tearDown()

	span := generateSpan()
	in, want := span, span

	responseSpan, err := traceClient.CreateSpan(ctx, in)
	if err != nil {
		t.Fatalf("failed to call CreateSpan: %v", err)
		return
	}

	if !proto.Equal(responseSpan, want) {
		t.Errorf("CreateSpan(%v) == %v, want %v",
			in, responseSpan, want)
	}
}

func TestMockTraceServer_CreateSpan_MissingFields(t *testing.T) {
	setup()
	defer tearDown()

	in := generateMissingFieldSpan("SpanId", "EndTime")
	want := validation.ErrMissingField.Err()
	missingFields := map[string]struct{}{
		"SpanId":  {},
		"EndTime": {},
	}

	responseSpan, err := traceClient.CreateSpan(ctx, in)
	if err == nil {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if valid := validation.ValidateErrDetails(err, missingFields); !valid {
		t.Errorf("CreateSpan(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_CreateSpan_InvalidTimestamp(t *testing.T) {
	setup()
	defer tearDown()

	in := generateInvalidTimestampSpan()
	want := validation.StatusInvalidTimestamp

	responseSpan, err := traceClient.CreateSpan(ctx, in)
	if err == nil {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if !strings.Contains(err.Error(), want.Error()) {
		t.Errorf("CreateSpan(%v) returned error %v, expected error %v",
			in, err.Error(), want)
	}
}

func TestMockTraceServer_CreateSpan_DuplicateName(t *testing.T) {
	setup()
	defer tearDown()

	const duplicateSpanName = "test-span-1"
	in := generateSpan(duplicateSpanName)
	want := validation.ErrDuplicateSpanName

	_, err := traceClient.CreateSpan(ctx, in)
	if err != nil {
		t.Errorf("CreateSpan(%v) returned error %v", in, err)
	}

	resp, err := traceClient.CreateSpan(ctx, in)
	if err == nil {
		t.Errorf("CreateSpan(%v) returned %v, expected error %v", in, resp, want)
	}

	if valid := validation.ValidateDuplicateSpanNames(err, duplicateSpanName); !valid {
		t.Errorf("expected duplicate spanName: %v", duplicateSpanName)
	}
}

func TestMockTraceServer_GetNumSpans(t *testing.T) {
	setup()
	defer tearDown()

	var wg sync.WaitGroup
	const expectedNumSpans = 6
	spans1 := []*cloudtrace.Span{
		generateSpan(),
		generateSpan(),
		generateSpan(),
	}

	spans2 := []*cloudtrace.Span{
		generateSpan(),
		generateSpan(),
		generateSpan(),
	}

	go func() {
		wg.Add(1)
		defer wg.Done()
		_, err := traceClient.BatchWriteSpans(ctx, &cloudtrace.BatchWriteSpansRequest{
			Name:  "projects/test-project",
			Spans: spans1,
		})
		if err != nil {
			t.Fatalf("failed to call BatchWriteSpans: %v", err)
		}
	}()

	_, err := traceClient.BatchWriteSpans(ctx, &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: spans2,
	})
	wg.Wait()

	numSpansResp, err := mockClient.GetNumSpans(ctx, &empty.Empty{})
	if err != nil {
		t.Fatalf("failed to call GetNumSpans: %v", err)
		return
	}

	if numSpansResp.NumSpans != expectedNumSpans {
		t.Errorf("GetNumSpans() == %v, expected %v", numSpansResp.NumSpans, expectedNumSpans)
	}
}
