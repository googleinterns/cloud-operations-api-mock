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
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxAnnotationAttributes = 4
	maxAnnotationBytes      = 256
	maxAttributes           = 32
	maxAttributeKeyBytes    = 128
	maxAttributeValueBytes  = 256
	maxDisplayNameBytes     = 128
	maxLinks                = 128
	maxTimeEvents           = 32
)

var (
	requiredFields   = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
	spanNameRegex    = regexp.MustCompile("^projects/(.*)/traces/[a-fA-F0-9]{32}/spans/[a-fA-F0-9]{16}")
	projectNameRegex = regexp.MustCompile("^projects/(.*)")

	StatusInvalidSpanName = status.Error(codes.InvalidArgument,
		"span name must be of the form projects/{project_id}/traces/{trace_id}/spans/{span_id}")
	StatusInvalidProjectName = status.Error(codes.InvalidArgument,
		"project name must be of the form projects/{project_id}")
	StatusInvalidTimestamp = status.Error(codes.InvalidArgument,
		"start time must be before end time")
	StatusMalformedTimestamp = status.Error(codes.InvalidArgument,
		"unable to parse timestamp")
	StatusTimeEventMissingTime = status.Error(codes.InvalidArgument,
		"time events' time field cannot be empty")
	StatusInvalidMessageEvent = status.Error(codes.InvalidArgument,
		"message events must contain a type, ID and uncompressed size in bytes")
	StatusInvalidLink = status.Error(codes.InvalidArgument,
		"links must contain a Span ID and trace ID")

	StatusInvalidDisplayName = status.Error(codes.InvalidArgument,
		fmt.Sprintf("displayName has max length of %v bytes", maxDisplayNameBytes))
	StatusTooManyAttributes = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v attributes", maxAttributes))
	StatusInvalidAttributeKey = status.Error(codes.InvalidArgument,
		fmt.Sprintf("attribute keys have a max length of %v bytes", maxAttributeKeyBytes))
	StatusInvalidAttributeValue = status.Error(codes.InvalidArgument,
		fmt.Sprintf("attribute values have a max length of %v bytes", maxAttributeValueBytes))
	StatusTooManyTimeEvents = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v time events", maxTimeEvents))
	StatusInvalidAnnotation = status.Error(codes.InvalidArgument,
		fmt.Sprintf("annotation descriptions have a max length of %v bytes", maxAnnotationBytes))
	StatusTooManyAnnotationAttributes = status.Error(codes.InvalidArgument,
		fmt.Sprintf("annotations can have at most %v attributes", maxAnnotationAttributes))
	StatusTooManyLinks = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v links", maxLinks))

	StatusDuplicateSpanName = status.New(codes.AlreadyExists, "duplicate span name")
)

// ValidateSpans checks that the spans conform to the API requirements.
// That is, required fields are present, and optional fields are of the correct form.
func ValidateSpans(requestName string, spans ...*cloudtrace.Span) error {
	for _, span := range spans {
		// Validate required fields are present and semantically make sense.
		if err := CheckForRequiredFields(requiredFields, reflect.ValueOf(span), requestName); err != nil {
			return err
		}
		if err := validateName(span.Name); err != nil {
			return err
		}
		if err := validateTimeStamps(span); err != nil {
			return err
		}
		if err := validateDisplayName(span.DisplayName); err != nil {
			return err
		}

		// Validate that if optional fields are present, they conform to the API.
		if err := validateAttributes(span.Attributes, maxAttributes); err != nil {
			return err
		}
		if err := validateTimeEvents(span.TimeEvents); err != nil {
			return err
		}
		if err := validateLinks(span.Links); err != nil {
			return err
		}
	}
	return nil
}

func validateTimeStamps(span *cloudtrace.Span) error {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return StatusMalformedTimestamp
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return StatusMalformedTimestamp
	}

	if !start.Before(end) {
		return StatusInvalidTimestamp
	}
	return nil
}

// AddSpans adds the given spans to the map of uploaded spans as long as there are no duplicate names.
// If a duplicate span name is detected, it will not be written, and an error is returned.
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
				st, err := StatusDuplicateSpanName.WithDetails(br)
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

// ValidateProjectName verifies that the project name from the BatchWriteSpans request
// is of the form projects/[PROJECT_ID]
func ValidateProjectName(projectName string) error {
	if !projectNameRegex.MatchString(projectName) {
		return StatusInvalidProjectName
	}
	return nil
}

// A display name can be at most 128 bytes.
func validateDisplayName(displayName *cloudtrace.TruncatableString) error {
	if len(displayName.Value) > maxDisplayNameBytes {
		return StatusInvalidDisplayName
	}
	return nil
}

// The span name must be of the form projects/{project_id}/traces/{trace_id}/spans/{span_id} where
// trace_id is a 32-char hex encoding, span_id is a 16-char hex encoding.
func validateName(name string) error {
	if !spanNameRegex.MatchString(name) {
		return StatusInvalidSpanName
	}
	return nil
}

// A span can have at most 32 attributes, where each attribute is a dictionary.
// The key is a string with max length of 128 bytes, and the value can be a string, int64 or bool.
// If the value is a string, it has a max length of 256 bytes.
func validateAttributes(attributes *cloudtrace.Span_Attributes, maxAttributes int) error {
	if attributes == nil {
		return nil
	}
	if len(attributes.AttributeMap) > maxAttributes {
		return StatusTooManyAttributes
	}

	for k, v := range attributes.AttributeMap {
		if len(k) > maxAttributeKeyBytes {
			return StatusInvalidAttributeKey
		}
		if val, ok := v.Value.(*cloudtrace.AttributeValue_StringValue); ok {
			if len(val.StringValue.Value) > maxAttributeValueBytes {
				return StatusInvalidAttributeValue
			}
		}
	}

	return nil
}

// A span can have at most 32 TimeEvents. A TimeEvent consists of a TimeStamp,
// and either an Annotation or a MessageEvent.
// An Annotation is a dictionary that maps a string description to a list of attributes.
// A MessageEvent describes messages sent between spans and must contain an ID and size.
func validateTimeEvents(events *cloudtrace.Span_TimeEvents) error {
	if events == nil {
		return nil
	}
	if len(events.TimeEvent) > maxTimeEvents {
		return StatusTooManyTimeEvents
	}

	for _, event := range events.TimeEvent {
		if event.Time == nil {
			return StatusTimeEventMissingTime
		}

		switch e := event.Value.(type) {
		case *cloudtrace.Span_TimeEvent_Annotation_:
			if len(e.Annotation.Description.Value) > maxAnnotationBytes {
				return StatusInvalidAnnotation
			}

			if err := validateAttributes(e.Annotation.Attributes, maxAnnotationAttributes); err != nil {
				return StatusTooManyAnnotationAttributes
			}
		case *cloudtrace.Span_TimeEvent_MessageEvent_:
			if e.MessageEvent.Id <= 0 || e.MessageEvent.UncompressedSizeBytes <= 0 {
				return StatusInvalidMessageEvent
			}
		}
	}

	return nil
}

// A span can have at most 128 links, which links the span to another span.
// A link contains a traceId, spanId, the type of the span, and at most 32 attributes.
func validateLinks(links *cloudtrace.Span_Links) error {
	if links == nil {
		return nil
	}
	if len(links.Link) > maxLinks {
		return StatusTooManyLinks
	}

	for _, link := range links.Link {
		if link.SpanId == "" || link.TraceId == "" {
			return StatusInvalidLink
		}
		if err := validateAttributes(link.Attributes, maxAttributes); err != nil {
			return err
		}
	}

	return nil
}

// ValidateDuplicateSpanNames is used in tests to verify that the error returned
// contains the correct span name (that is, the duplicate span name).
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
