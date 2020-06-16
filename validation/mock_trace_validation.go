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

package validation

import (
	"reflect"

	"github.com/golang/protobuf/ptypes"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidTimestamp   = status.Error(codes.InvalidArgument, "start time must be before end time")
	ErrMalformedTimestamp = status.Error(codes.InvalidArgument, "unable to parse timestamp")
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
)

func IsSpanValid(span *cloudtrace.Span, requestName string) error {
	if err := CheckForRequiredFields(requiredFields, reflect.ValueOf(span), requestName); err != nil {
		return err
	}

	if err := validateTimeStamps(span); err != nil {
		return err
	}

	return nil
}

func validateTimeStamps(span *cloudtrace.Span) error {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return ErrMalformedTimestamp
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return ErrMalformedTimestamp
	}

	if !start.Before(end) {
		return ErrInvalidTimestamp
	}
	return nil
}
