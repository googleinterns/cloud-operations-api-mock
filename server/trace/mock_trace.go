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
	uploadedSpans     map[string]*cloudtrace.Span
	uploadedSpansLock sync.Mutex
	delay             time.Duration
	delayLock         sync.Mutex
}

// NewMockTraceServer creates a new MockTraceServer and returns a pointer to it.
func NewMockTraceServer() *MockTraceServer {
	uploadedSpans := make(map[string]*cloudtrace.Span)
	return &MockTraceServer{uploadedSpans: uploadedSpans}
}

// BatchWriteSpans creates and stores a list of spans on the server.
func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	if err := validation.ValidateProjectName(req.Name); err != nil {
		return nil, err
	}
	if err := validation.ValidateSpans("BatchWriteSpans", req.Spans...); err != nil {
		return nil, err
	}
	if err := validation.AddSpans(s.uploadedSpans, &s.uploadedSpansLock, s.delay, ctx, req.Spans...); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// CreateSpan creates and stores a single span on the server.
func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	if err := validation.ValidateSpans("CreateSpan", span); err != nil {
		return nil, err
	}
	if err := validation.AddSpans(s.uploadedSpans, &s.uploadedSpansLock, s.delay, ctx, span); err != nil {
		return nil, err
	}
	return span, nil
}

// GetNumSpans returns the number of spans currently stored on the server.
func (s *MockTraceServer) GetNumSpans(ctx context.Context, req *empty.Empty) (*mocktrace.GetNumSpansResponse, error) {
	s.uploadedSpansLock.Lock()
	defer s.uploadedSpansLock.Unlock()
	return &mocktrace.GetNumSpansResponse{
		NumSpans: int32(len(s.uploadedSpans)),
	}, nil
}

// SetDelay sets the amount of time to delay before writing the spans to memory.
func (s *MockTraceServer) SetDelay(ctx context.Context, req *mocktrace.SetDelayRequest) (*empty.Empty, error) {
	s.delayLock.Lock()
	defer s.delayLock.Unlock()
	s.delay = (time.Duration(req.Duration.Seconds) * time.Second) + time.Duration(req.Duration.Nanos)
	return &empty.Empty{}, nil
}
