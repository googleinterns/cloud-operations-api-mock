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
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
)

var (
	InvalidTimestampErr   = status.Error(codes.InvalidArgument, "start time must be before end time")
	MalformedTimestampErr = status.Error(codes.InvalidArgument, "unable to parse timestamp")
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
)

type MissingFieldError struct {
	Code          codes.Code
	MissingFields []string
}

func (err *MissingFieldError) Error() string {
	return fmt.Sprintf("span is missing required fields: %v",
		strings.Join(err.MissingFields, ", "))
}

func IsSpanValid(span *cloudtrace.Span) error {
	if err := validateRequiredFields(span); err != nil {
		return err
	}

	if err := validateTimeStamps(span); err != nil {
		return err
	}

	return nil
}

func validateRequiredFields(span *cloudtrace.Span) error {
	var missingFields []string
	spanReflect := reflect.ValueOf(span)

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(spanReflect).FieldByName(field).IsZero(); isZero {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) == 0 {
		return nil
	} else {
		return &MissingFieldError{
			Code:          codes.InvalidArgument,
			MissingFields: missingFields,
		}
	}
}

func validateTimeStamps(span *cloudtrace.Span) error {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return MalformedTimestampErr
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return MalformedTimestampErr
	}

	if start.Before(end) {
		return nil
	} else {
		return InvalidTimestampErr
	}
}
