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
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"

	"golang.org/x/net/context"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockTraceServer struct {
	cloudtrace.UnimplementedTraceServiceServer
}

const (
	invalidTimestampMsg   = "start time must be before end time"
	malformedTimestampMsg = "unable to parse timestamp"
	missingFieldMsg       = "span is missing required fields: %v"
)

var (
	requiredMessageFields = []string{"DisplayName", "StartTime", "EndTime"}
	requiredStringFields  = []string{"Name", "SpanId"}
)

func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	for _, span := range req.Spans {
		if isValid, errMsg := isSpanValid(span); !isValid {
			return nil, status.Errorf(codes.InvalidArgument, errMsg)
		}
	}
	return &empty.Empty{}, nil
}

func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	if isValid, errMsg := isSpanValid(span); !isValid {
		return nil, status.Errorf(codes.InvalidArgument, errMsg)
	}
	return span, nil
}

func isSpanValid(span *cloudtrace.Span) (bool, string) {
	if isValid, msg := validateRequiredFields(span); !isValid {
		return isValid, msg
	}

	if isValid, msg := validateTimeStamps(span); !isValid {
		return isValid, msg
	}

	return true, ""
}

func validateRequiredFields(span *cloudtrace.Span) (bool, string) {
	var missingFields []string
	spanReflect := reflect.ValueOf(span)

	for _, field := range requiredStringFields {
		fieldValue := reflect.Indirect(spanReflect).FieldByName(field).String()
		if fieldValue == "" {
			missingFields = append(missingFields, field)
		}
	}

	for _, field := range requiredMessageFields {
		fieldValue := reflect.Indirect(spanReflect).FieldByName(field)
		if fieldValue.IsNil() {
			missingFields = append(missingFields, field)
		}
	}

	formattedErrMsg := fmt.Sprintf(missingFieldMsg, strings.Join(missingFields, ", "))
	return len(missingFields) == 0, formattedErrMsg
}

func validateTimeStamps(span *cloudtrace.Span) (bool, string) {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return false, malformedTimestampMsg
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return false, malformedTimestampMsg
	}

	return start.Before(end), invalidTimestampMsg
}
