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

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrMissingField = status.New(codes.InvalidArgument, "missing required field(s)")
)

const (
	missingFieldMsg = "%v must contain required %v field"
)

func CheckForRequiredFields(requiredFields []string, reqReflect reflect.Value, requestName string) error {
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if reflect.Indirect(reqReflect).FieldByName(field).IsZero() {
			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: fmt.Sprintf(missingFieldMsg, requestName, field),
			}
			br.FieldViolations = append(br.FieldViolations, v)
		}
	}

	if len(br.FieldViolations) > 0 {
		st, err := ErrMissingField.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}
	return nil
}

func ValidateErrDetails(err error, missingFields map[string]struct{}) bool {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if t, ok := detail.(*errdetails.BadRequest); ok {
			for _, violation := range t.GetFieldViolations() {
				if _, ok := missingFields[violation.GetField()]; !ok {
					return false
				}
			}
		}
	}
	return true
}
