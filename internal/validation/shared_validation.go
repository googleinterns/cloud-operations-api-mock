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
	"regexp"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

const (
	missingFieldMsg = "%v must contain the required %v field"
)

var (
	projectNameRegex = regexp.MustCompile("^projects/[^/]+$")
)

// CheckForRequiredFields verifies that the required fields for the given request are present.
func CheckForRequiredFields(requiredFields []string, reqReflect reflect.Value, requestName string) error {
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if reflect.Indirect(reqReflect).FieldByName(field).IsZero() {
			// Checking for required fields in MetricDescriptor field
			desc := ""
			if requestName == "MetricDescriptor" {
				desc = fmt.Sprintf("MetricDescriptor must contain the required %v field", field)
			} else {
				desc = fmt.Sprintf(missingFieldMsg, requestName, field)
			}

			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: desc,
			}
			br.FieldViolations = append(br.FieldViolations, v)
		}
	}

	if len(br.FieldViolations) > 0 {
		st, err := statusMissingField.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}
	return nil
}

// ValidateMissingFieldsErrDetails is used to test that the error details contain the correct missing fields.
func ValidateMissingFieldsErrDetails(err error, missingFields map[string]struct{}) bool {
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

// ValidateDuplicateErrDetails is used in tests to verify that the error returned
// contains the correct duplicated value.
func ValidateDuplicateErrDetails(err error, duplicateName string) bool {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		if t, ok := detail.(*errdetails.ErrorInfo); ok {
			if t.GetReason() != duplicateName {
				return false
			}
		}
	}
	return true
}

// ValidateProjectName verifies that the project name from the BatchWriteSpans request
// is of the form projects/[PROJECT_ID]
func ValidateProjectName(projectName string) error {
	if !projectNameRegex.MatchString(projectName) {
		return statusInvalidProjectName
	}
	return nil
}