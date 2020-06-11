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
	"log"
	"reflect"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
)

var (
	InvalidTimestampErr   = status.Error(codes.InvalidArgument, "start time must be before end time")
	MalformedTimestampErr = status.Error(codes.InvalidArgument, "unable to parse timestamp")
	MissingFieldErr       = status.New(codes.InvalidArgument, "missing required fields")
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
)

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
	var missingFields []*errdetails.BadRequest_FieldViolation
	spanReflect := reflect.ValueOf(span)

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(spanReflect).FieldByName(field).IsZero(); isZero {
			missingFields = append(missingFields, &errdetails.BadRequest_FieldViolation{
				Field: field,
			})
		}
	}

	if len(missingFields) == 0 {
		return nil
	} else {
		badRequest := &errdetails.BadRequest{}
		badRequest.FieldViolations = missingFields
		result, err := MissingFieldErr.WithDetails(badRequest)
		if err != nil {
			log.Print("Unable to add field violations")
			return MissingFieldErr.Err()
		}
		return result.Err()
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
