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

	"github.com/golang/protobuf/ptypes/empty"
	mocktrace "github.com/googleinterns/cloud-operations-api-mock/api"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"

	"golang.org/x/net/context"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

type MockTraceServer struct {
	cloudtrace.UnimplementedTraceServiceServer
	mocktrace.UnimplementedMockTraceServiceServer
	uploadedSpans     map[string]*cloudtrace.Span
	uploadedSpansLock sync.Mutex
}

func NewMockTraceServer() *MockTraceServer {
	uploadedSpans := make(map[string]*cloudtrace.Span)
	return &MockTraceServer{uploadedSpans: uploadedSpans}
}

func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	if err := validation.ValidateSpans("BatchWriteSpans", req.Spans...); err != nil {
		return nil, err
	}
	if err := validation.AddSpans(s.uploadedSpans, &s.uploadedSpansLock, req.Spans...); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	if err := validation.ValidateSpans("CreateSpan", span); err != nil {
		return nil, err
	}
	if err := validation.AddSpans(s.uploadedSpans, &s.uploadedSpansLock, span); err != nil {
		return nil, err
	}
	return span, nil
}

func (s *MockTraceServer) GetNumSpans(ctx context.Context, req *empty.Empty) (*mocktrace.GetNumSpansResponse, error) {
	s.uploadedSpansLock.Lock()
	defer s.uploadedSpansLock.Unlock()
	return &mocktrace.GetNumSpansResponse{
		NumSpans: int32(len(s.uploadedSpans)),
	}, nil
}
