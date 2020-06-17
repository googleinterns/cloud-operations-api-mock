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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/validation"

	"golang.org/x/net/context"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

type MockTraceServer struct {
	cloudtrace.UnimplementedTraceServiceServer
}

func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	for _, span := range req.Spans {
		if err := validation.IsSpanValid(span, "BatchWriteSpans"); err != nil {
			return nil, err
		}
	}
	return &empty.Empty{}, nil
}

func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	if err := validation.IsSpanValid(span, "CreateSpan"); err != nil {
		return nil, err
	}
	return span, nil
}
