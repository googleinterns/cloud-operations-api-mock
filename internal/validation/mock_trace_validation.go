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
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidTimestamp   = status.Error(codes.InvalidArgument, "start time must be before end time")
	ErrMalformedTimestamp = status.Error(codes.InvalidArgument, "unable to parse timestamp")
	ErrDuplicateSpanName  = status.New(codes.AlreadyExists, "duplicate span name")
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
)

func ValidateSpans(requestName string, spans ...*cloudtrace.Span) error {
	for _, span := range spans {
		if err := CheckForRequiredFields(requiredFields, reflect.ValueOf(span), requestName); err != nil {
			return err
		}

		if err := validateTimeStamps(span); err != nil {
			return err
		}
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

func AddSpans(uploadedSpans map[string]*cloudtrace.Span, lock *sync.Mutex,
	delay time.Duration, ctx context.Context, spans ...*cloudtrace.Span) error {

	br := &errdetails.ErrorInfo{}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		lock.Lock()
		defer lock.Unlock()
		for _, span := range spans {
			if _, ok := uploadedSpans[span.Name]; ok {
				br.Reason = span.Name
				st, err := ErrDuplicateSpanName.WithDetails(br)
				if err != nil {
					panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
				}
				return st.Err()
			}
			uploadedSpans[span.Name] = span
		}
	}

	return nil
}

func ValidateDuplicateSpanNames(err error, duplicateName string) bool {
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
