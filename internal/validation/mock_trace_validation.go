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
	"reflect"
	"regexp"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	genprotoStatus "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// These restrictions can be found at
	// https://cloud.google.com/trace/docs/reference/v2/rpc/google.devtools.cloudtrace.v2
	maxAnnotationAttributes = 4
	maxAnnotationBytes      = 256
	maxAttributes           = 32
	maxAttributeKeyBytes    = 128
	maxAttributeValueBytes  = 256
	maxDisplayNameBytes     = 128
	maxLinks                = 128
	maxTimeEvents           = 32

	agent          = "g.co/agent"
	shortenedAgent = "agent"
)

var (
	// The exporter is responsible for mapping these special attributes to the correct
	// canonical Cloud Trace attributes (/http/method, /http/route, etc.)
	specialAttributes = map[string]struct{}{
		"http.method":      {},
		"http.route":       {},
		"http.status_code": {},
	}
	requiredFields = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
	spanNameRegex  = regexp.MustCompile("^projects/[^/]+/traces/[a-fA-F0-9]{32}/spans/[a-fA-F0-9]{16}$")
	agentRegex     = regexp.MustCompile(`^opentelemetry-[a-zA-Z]+ [0-9]+\.[0-9]+\.[0-9]+; google-cloud-trace-exporter [0-9]+\.[0-9]+\.[0-9]+$`)
)

// SpanData wraps all the span data on the server into a struct.
type SpanData struct {
	// If a batch has a bad span, we don't write batch to memory, but still want
	// info on them for summary, so need SpansSummary
	SpansSummary      []*cloudtrace.Span
	UploadedSpanNames map[string]struct{}
	UploadedSpans     []*cloudtrace.Span
	Mutex             sync.RWMutex
}

// ValidateSpans checks that the spans conform to the API requirements.
// That is, required fields are present, and optional fields are of the correct form.
// If any violations are detected, the errors will be added to the result table.
func ValidateSpans(requestName string, spanData *SpanData, spans ...*cloudtrace.Span) error {
	var overallError error
	currentRequestSpanNames := make(map[string]struct{})

	for _, span := range spans {
		var currentError error

		// Validate required fields are present and semantically make sense.
		if err := CheckForRequiredFields(requiredFields, reflect.ValueOf(span), requestName); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}
		if err := validateName(span.Name, spanData.UploadedSpanNames, currentRequestSpanNames); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}
		if err := validateTimeStamps(span); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}
		if err := validateDisplayName(span.DisplayName); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}

		// Validate that if optional fields are present, they conform to the API.
		if err := validateAttributes(span.Attributes, maxAttributes); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}
		if err := validateTimeEvents(span.TimeEvents); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}
		if err := validateLinks(span.Links); err != nil {
			addSpanToSummary(&spanData.SpansSummary, span, err)
			currentError = err
		}

		if currentError == nil {
			addSpanToSummary(&spanData.SpansSummary, span, nil)
		} else {
			overallError = currentError
		}
	}

	if overallError != nil {
		return overallError
	}

	return nil
}

// addSpanToSummary sets the span's status and adds it to the summary slice.
func addSpanToSummary(spanSummary *[]*cloudtrace.Span, span *cloudtrace.Span, err error) {
	setSpanStatus(span, err)
	*spanSummary = append(*spanSummary, span)
}

func setSpanStatus(span *cloudtrace.Span, err error) {
	if err == nil {
		span.Status = &genprotoStatus.Status{
			Code:    int32(codes.OK),
			Message: "OK",
		}
	} else {
		span.Status = &genprotoStatus.Status{
			Code:    int32(status.Convert(err).Code()),
			Message: status.Convert(err).Message(),
		}
	}
}

// AddSpans adds the given spans to the list of uploaded spans.
func AddSpans(spanData *SpanData, spans ...*cloudtrace.Span) {
	for _, span := range spans {
		spanData.UploadedSpans = append(spanData.UploadedSpans, span)
		spanData.UploadedSpanNames[span.Name] = struct{}{}
	}
}

// Delay will block for the specified amount of time.
// Used to delay writing spans to memory.
func Delay(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// AccessSpan returns the span at the given index if it is in range.
// If it is not in range, nil is returned.
func AccessSpan(index int, uploadedSpans []*cloudtrace.Span) *cloudtrace.Span {
	if index >= len(uploadedSpans) || index < 0 {
		return nil
	}
	return uploadedSpans[index]
}

// validateDisplayName verifies that the display name has at most 128 bytes.
func validateDisplayName(displayName *cloudtrace.TruncatableString) error {
	if len(displayName.Value) > maxDisplayNameBytes {
		return statusInvalidDisplayName
	}
	return nil
}

// validateName verifies that the span name is not a duplicate, and is of the form:
// projects/{project_id}/traces/{trace_id}/spans/{span_id}
// where trace_id is a 32-char hex encoding, and span_id is a 16-char hex encoding.
func validateName(name string, spanNames map[string]struct{}, currentRequestSpanNames map[string]struct{}) error {
	if _, ok := spanNames[name]; ok {
		return statusDuplicateSpanName
	}

	if _, ok := currentRequestSpanNames[name]; ok {
		return statusDuplicateSpanName
	}

	if !spanNameRegex.MatchString(name) {
		return statusInvalidSpanName
	}

	currentRequestSpanNames[name] = struct{}{}
	return nil
}

// validateTimeStamps verifies that the start time of a span is before its end time.
func validateTimeStamps(span *cloudtrace.Span) error {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return statusMalformedTimestamp
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return statusMalformedTimestamp
	}

	if !start.Before(end) {
		return statusInvalidInterval
	}
	return nil
}

// validateAttributes verifies that a span has at most 32 attributes, where each attribute is a dictionary.
// The key is a string with max length of 128 bytes, and the value can be a string, int64 or bool.
// If the value is a string, it has a max length of 256 bytes.
func validateAttributes(attributes *cloudtrace.Span_Attributes, maxAttributes int) error {
	if attributes == nil {
		return nil
	}
	if len(attributes.AttributeMap) > maxAttributes {
		return statusTooManyAttributes
	}

	containsAgent := false

	for k, v := range attributes.AttributeMap {
		if len(k) > maxAttributeKeyBytes {
			return statusInvalidAttributeKey
		}

		// Ensure that the special attributes have been translated properly.
		if _, ok := specialAttributes[k]; ok {
			return statusUnmappedSpecialAttribute
		}

		if val, ok := v.Value.(*cloudtrace.AttributeValue_StringValue); ok {
			if len(val.StringValue.Value) > maxAttributeValueBytes {
				return statusInvalidAttributeValue
			}

			// The span must contain the attribute "g.co/agent" or "agent".
			if k == agent || k == shortenedAgent {
				containsAgent = true
				if err := validateAgent(val.StringValue.Value); err != nil {
					return err
				}
			}
		}
	}

	if !containsAgent {
		return statusMissingAgentAttribute
	}

	return nil
}

// validateAgent checks that the g.co/agent or agent attribute is of the form
// opentelemetry-<language_code> <ot_version>; google-cloud-trace-exporter <exporter_version>
func validateAgent(agent string) error {
	if !agentRegex.MatchString(agent) {
		return statusInvalidAgentAttribute
	}
	return nil
}

// validateTimeEvents verifies that a span has at most 32 TimeEvents.
// A TimeEvent consists of a TimeStamp, and either an Annotation or a MessageEvent.
// An Annotation is a dictionary that maps a string description to a list of attributes.
// A MessageEvent describes messages sent between spans and must contain an ID and size.
func validateTimeEvents(events *cloudtrace.Span_TimeEvents) error {
	if events == nil {
		return nil
	}
	if len(events.TimeEvent) > maxTimeEvents {
		return statusTooManyTimeEvents
	}

	for _, event := range events.TimeEvent {
		if event.Time == nil {
			return statusTimeEventMissingTime
		}

		switch e := event.Value.(type) {
		case *cloudtrace.Span_TimeEvent_Annotation_:
			if len(e.Annotation.Description.Value) > maxAnnotationBytes {
				return statusInvalidAnnotation
			}

			if err := validateAttributes(e.Annotation.Attributes, maxAnnotationAttributes); err != nil {
				return err
			}
		case *cloudtrace.Span_TimeEvent_MessageEvent_:
			if e.MessageEvent.Id <= 0 || e.MessageEvent.UncompressedSizeBytes <= 0 {
				return statusInvalidMessageEvent
			}
		}
	}

	return nil
}

// validateLinks verifies that a span has at most 128 links, which are used to link the span to another span.
// A link contains a traceId, spanId, the type of the span, and at most 32 attributes.
func validateLinks(links *cloudtrace.Span_Links) error {
	if links == nil {
		return nil
	}
	if len(links.Link) > maxLinks {
		return statusTooManyLinks
	}

	for _, link := range links.Link {
		if link.SpanId == "" || link.TraceId == "" {
			return statusInvalidLink
		}
		if err := validateAttributes(link.Attributes, maxAttributes); err != nil {
			return err
		}
	}

	return nil
}
