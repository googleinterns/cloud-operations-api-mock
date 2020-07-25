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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	mocktrace "github.com/googleinterns/cloud-operations-api-mock/api"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

// MockTraceServer implements all of the RPCs pertaining to
// tracing that can be called by the client.
type MockTraceServer struct {
	cloudtrace.UnimplementedTraceServiceServer
	mocktrace.UnimplementedMockTraceServiceServer
	spanData     *validation.SpanData
	delay        time.Duration
	delayLock    sync.Mutex
	onUpload     func(ctx context.Context, spans []*cloudtrace.Span)
	onUploadLock sync.Mutex
}

// NewMockTraceServer creates a new MockTraceServer and returns a pointer to it.
func NewMockTraceServer() *MockTraceServer {
	uploadedSpanNames := make(map[string]struct{})
	spanData := &validation.SpanData{
		UploadedSpanNames: uploadedSpanNames,
	}
	return &MockTraceServer{spanData: spanData}
}

// BatchWriteSpans creates and stores a list of spans on the server.
// If ANY of the spans in the request are invalid, ALL spans will be dropped.
func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	if err := validation.ValidateProjectName(req.Name); err != nil {
		return nil, err
	}

	if err := validation.Delay(ctx, s.delay); err != nil {
		return nil, err
	}

	if s.onUpload != nil {
		s.onUpload(ctx, req.Spans)
	}

	s.spanData.Mutex.Lock()
	defer s.spanData.Mutex.Unlock()
	if err := validation.ValidateSpans("BatchWriteSpans", s.spanData, req.Spans...); err != nil {
		return nil, err
	}

	validation.AddSpans(s.spanData, req.Spans...)

	return &empty.Empty{}, nil
}

// CreateSpan validates a single span, and returns the Span proto.
func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	// Validate the span and add to result table.
	s.spanData.Mutex.Lock()
	defer s.spanData.Mutex.Unlock()
	if err := validation.ValidateSpans("CreateSpan", s.spanData, span); err != nil {
		return nil, err
	}
	return span, nil
}

// GetNumSpans returns the number of spans currently stored on the server.
// Used by the library implementation to avoid having to make a network call.
func (s *MockTraceServer) GetNumSpans() int {
	s.spanData.Mutex.Lock()
	defer s.spanData.Mutex.Unlock()
	return len(s.spanData.UploadedSpans)
}

// ListSpans returns a list of all the spans currently stored on the server.
func (s *MockTraceServer) ListSpans(ctx context.Context, req *empty.Empty) (*mocktrace.ListSpansResponse, error) {
	s.spanData.Mutex.Lock()
	defer s.spanData.Mutex.Unlock()
	return &mocktrace.ListSpansResponse{
		Spans: s.spanData.UploadedSpans,
	}, nil
}

// GetSpan returns the span that was stored in memory at the given index.
// If the index is out of bounds, nil is returned.
func (s *MockTraceServer) GetSpan(index int) *cloudtrace.Span {
	s.spanData.Mutex.Lock()
	defer s.spanData.Mutex.Unlock()
	span := validation.AccessSpan(index, s.spanData.UploadedSpans)
	return span
}

// SetDelay sets the amount of time to delay before writing the spans to memory.
func (s *MockTraceServer) SetDelay(delay time.Duration) {
	s.delayLock.Lock()
	defer s.delayLock.Unlock()
	s.delay = delay
}

// SetOnUpload sets the onUpload function which is called before BatchWriteSpans runs.
func (s *MockTraceServer) SetOnUpload(onUpload func(ctx context.Context, spans []*cloudtrace.Span)) {
	s.onUploadLock.Lock()
	defer s.onUploadLock.Unlock()
	s.onUpload = onUpload
}

func (s *MockTraceServer) ResultTable() *[]*cloudtrace.Span {
	return &s.spanData.SpansSummary
}
