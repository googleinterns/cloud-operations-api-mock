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
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const (
	bufSize        = 1024 * 1024
	spanNameFormat = "projects/%v/traces/%v/spans/%v"
)

type TestMock struct {
	conn        *grpc.ClientConn
	ctx         context.Context
	grpcServer  *grpc.Server
	traceServer *MockTraceServer
	traceClient cloudtrace.TraceServiceClient
}

var (
	lis *bufconn.Listener
)

func setup() *TestMock {
	// Setup the in-memory server.
	lis = bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	traceServer := NewMockTraceServer()
	cloudtrace.RegisterTraceServiceServer(grpcServer, traceServer)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("server exited with error: %v", err)
		}
	}()

	// Setup the connection and clients.
	ctx := context.Background()
	var err error
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to dial bufnet: %v", err)
	}
	traceClient := cloudtrace.NewTraceServiceClient(conn)

	return &TestMock{
		conn,
		ctx,
		grpcServer,
		traceServer,
		traceClient,
	}
}

func (mock *TestMock) tearDown() {
	if err := mock.conn.Close(); err != nil {
		log.Fatalf("failed to close connection: %v", err)
	}
	mock.grpcServer.GracefulStop()
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
	traceID, err := generateHex(16)
	if err != nil {
		log.Fatalf("failed to generate span name: %v", err)
	}
	spanID, err := generateHex(8)
	if err != nil {
		log.Fatalf("failed to generate span name: %v", err)
	}
	return fmt.Sprintf(spanNameFormat, projectName, traceID, spanID)
}

func TestMockTraceServer_BatchWriteSpans(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

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

	response, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
		return
	}

	if !proto.Equal(response, want) {
		t.Errorf("BatchWriteSpans(%v) == %v, want %v", in, response, want)
	}
}

func TestMockTraceServer_BatchWriteSpans_MissingField(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	missingFieldsSpan := []*cloudtrace.Span{
		generateMissingFieldSpan("Name", "StartTime"),
		generateMissingFieldSpan(),
	}
	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: missingFieldsSpan,
	}
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{
		"Name":      {},
		"StartTime": {},
	}

	responseSpan, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err == nil || status.Code(err) != want {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("BatchWriteSpans(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_BatchWriteSpans_InvalidTimestamp(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	invalidTimestampSpans := []*cloudtrace.Span{
		generateInvalidTimestampSpan(),
		generateInvalidTimestampSpan(),
	}

	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: invalidTimestampSpans,
	}
	want := codes.InvalidArgument

	responseSpan, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err == nil || status.Code(err) != want {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
	}
}

func TestMockTraceServer_BatchWriteSpans_DuplicateName(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	duplicateSpan := generateSpan()
	spans := []*cloudtrace.Span{
		duplicateSpan,
		duplicateSpan,
	}
	in := &cloudtrace.BatchWriteSpansRequest{Name: "projects/test-project", Spans: spans}
	want := codes.AlreadyExists

	responseSpan, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err == nil || status.Code(err) != want {
		t.Errorf("BatchWriteSpans(%v) == %v, expected error %v",
			in, responseSpan, want)
	}

	if valid := validation.ValidateDuplicateErrDetails(err, duplicateSpan.Name); !valid {
		t.Errorf("expected duplicate spanName: %v", duplicateSpan.Name)
	}
}

func TestMockTraceServer_BatchWriteSpans_InvalidProjectName(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	spans := []*cloudtrace.Span{generateSpan()}
	requests := []*cloudtrace.BatchWriteSpansRequest{
		{Name: "invalid-project-name", Spans: spans},
		{Name: "projects/traces/foo/spans/bar", Spans: spans},
	}
	want := codes.InvalidArgument

	for _, request := range requests {
		_, err := mock.traceClient.BatchWriteSpans(mock.ctx, request)
		if err == nil || status.Code(err) != want {
			t.Errorf("BatchWriteSpans(%v) returned successfully, expected error %v",
				request, want)
		}
	}
}

func TestMockTraceServer_CreateSpan(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	span := generateSpan()
	in, want := span, span

	responseSpan, err := mock.traceClient.CreateSpan(mock.ctx, in)
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
	mock := setup()
	defer mock.tearDown()

	in := generateMissingFieldSpan("SpanId", "EndTime")
	want := codes.InvalidArgument
	missingFields := map[string]struct{}{
		"SpanId":  {},
		"EndTime": {},
	}

	responseSpan, err := mock.traceClient.CreateSpan(mock.ctx, in)
	if err == nil || status.Code(err) != want {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}

	if valid := validation.ValidateMissingFieldsErrDetails(err, missingFields); !valid {
		t.Errorf("CreateSpan(%v) expected missing fields %v", in, missingFields)
	}
}

func TestMockTraceServer_CreateSpan_InvalidTimestamp(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	in := generateInvalidTimestampSpan()
	want := codes.InvalidArgument

	responseSpan, err := mock.traceClient.CreateSpan(mock.ctx, in)
	if err == nil || status.Code(err) != want {
		t.Errorf("CreateSpan(%v) == %v, expected error %v",
			in, responseSpan, want)
		return
	}
}

func TestMockTraceServer_GetNumSpans(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

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

	errChan := make(chan error)
	go func() {
		_, err := mock.traceClient.BatchWriteSpans(mock.ctx, &cloudtrace.BatchWriteSpansRequest{
			Name:  "projects/test-project",
			Spans: spans1,
		})
		errChan <- err
	}()

	_, err := mock.traceClient.BatchWriteSpans(mock.ctx, &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: spans2,
	})
	if err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
	}

	// Wait for goroutine to finish.
	if err := <-errChan; err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
	}

	numSpans := mock.traceServer.GetNumSpans()
	if numSpans != expectedNumSpans {
		t.Errorf("GetNumSpans() == %v, expected %v", numSpans, expectedNumSpans)
	}
}

func TestMockTraceServer_GetSpan(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	createdSpan := generateSpan()

	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: []*cloudtrace.Span{createdSpan},
	}

	_, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err != nil {
		t.Errorf("failed to call BatchWriteSpans: %v", err)
	}

	index := 0
	retrievedSpan := mock.traceServer.GetSpan(index)
	if !proto.Equal(createdSpan, retrievedSpan) {
		t.Errorf("GetSpan(%v) == %v, expected %v", index, retrievedSpan, in)
	}

	invalidIndex := 1
	retrievedSpan = mock.traceServer.GetSpan(invalidIndex)
	if retrievedSpan != nil {
		t.Errorf("GetSpan(%v) == %v, expected nil", invalidIndex, retrievedSpan)
	}
}

func TestMockTraceServer_SetDelay(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	const delay = 100 * time.Millisecond
	mock.traceServer.SetDelay(delay)

	in := &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: []*cloudtrace.Span{generateSpan()},
	}

	// Test that the creation of the span is delayed the correct amount.
	start := time.Now()
	_, err := mock.traceClient.BatchWriteSpans(mock.ctx, in)
	if err != nil {
		t.Errorf("failed to call BatchWriteSpans: %v", err)
	}
	duration := time.Since(start).Milliseconds()

	if duration < delay.Milliseconds() {
		t.Errorf("Expected delay of %v", delay)
	}

	// Test timing out for BatchWriteSpans.
	const timeout = 20 * time.Millisecond

	ctx, cancel := context.WithTimeout(mock.ctx, timeout)
	_, err = mock.traceClient.BatchWriteSpans(ctx, &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: []*cloudtrace.Span{generateSpan()},
	})
	if err == nil || status.Code(err) != codes.DeadlineExceeded {
		t.Error("BatchWriteSpans expected timeout")
	}
	cancel()
}

func TestMockTraceServer_SetOnUpload(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	const numSpans = 3
	spans := []*cloudtrace.Span{
		generateSpan(),
		generateSpan(),
		generateSpan(),
	}

	// onUpload will just store the spans in a slice,
	// just to test that it's actually called.
	var exportedSpans []*cloudtrace.Span
	onUpload := func(ctx context.Context, spans []*cloudtrace.Span) {
		exportedSpans = append(exportedSpans, spans...)
	}
	mock.traceServer.SetOnUpload(onUpload)

	// Now call BatchWriteSpans, which should execute onUpload and store spans in the slice.
	_, err := mock.traceClient.BatchWriteSpans(mock.ctx, &cloudtrace.BatchWriteSpansRequest{
		Name:  "projects/test-project",
		Spans: spans,
	})
	if err != nil {
		t.Fatalf("failed to call BatchWriteSpans: %v", err)
	}
	if len(exportedSpans) != numSpans {
		t.Errorf("Expected %v spans, got %v instead", numSpans, len(exportedSpans))
	}
}

func TestMockTraceServer_Attributes(t *testing.T) {
	mock := setup()
	defer mock.tearDown()

	span := generateSpan()

	// Invalid attributes, contains an unmapped special attribute and
	// does not contain the agent.
	attributeMap := map[string]*cloudtrace.AttributeValue{
		"http.method": {},
	}
	span.Attributes = &cloudtrace.Span_Attributes{
		AttributeMap: attributeMap,
	}
	_, err := mock.traceClient.CreateSpan(mock.ctx, span)
	if err == nil || status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected invalid attributes, got success")
	}

	// Remove invalid attribute and add agent.
	delete(attributeMap, "http.method")
	attributeMap["g.co/agent"] = &cloudtrace.AttributeValue{
		Value: &cloudtrace.AttributeValue_StringValue{
			StringValue: &cloudtrace.TruncatableString{
				Value: "opentelemetry-go 1.0.0; google-cloud-trace-exporter 1.0.1"},
		},
	}
	_, err = mock.traceClient.CreateSpan(mock.ctx, span)
	if err != nil {
		t.Fatalf("expected success, got error %v", err)
	}
}
