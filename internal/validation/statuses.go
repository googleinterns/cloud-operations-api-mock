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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// Trace statuses.
	statusInvalidSpanName = status.Error(codes.InvalidArgument,
		"span name must be of the form projects/{project_id}/traces/{trace_id}/spans/{span_id}")
	statusInvalidProjectName = status.Error(codes.InvalidArgument,
		"project name must be of the form projects/{project_id}")
	statusInvalidTimestamp = status.Error(codes.InvalidArgument,
		"start time must be before end time")
	statusMalformedTimestamp = status.Error(codes.InvalidArgument,
		"unable to parse timestamp")
	statusTimeEventMissingTime = status.Error(codes.InvalidArgument,
		"time events' time field cannot be empty")
	statusInvalidMessageEvent = status.Error(codes.InvalidArgument,
		"message events must contain a type, ID and uncompressed size in bytes")
	statusInvalidLink = status.Error(codes.InvalidArgument,
		"links must contain a Span ID and trace ID")
	statusInvalidDisplayName = status.Error(codes.InvalidArgument,
		fmt.Sprintf("displayName has max length of %v bytes", maxDisplayNameBytes))
	statusTooManyAttributes = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v attributes", maxAttributes))
	statusInvalidAttributeKey = status.Error(codes.InvalidArgument,
		fmt.Sprintf("attribute keys have a max length of %v bytes", maxAttributeKeyBytes))
	statusInvalidAttributeValue = status.Error(codes.InvalidArgument,
		fmt.Sprintf("attribute values have a max length of %v bytes", maxAttributeValueBytes))
	statusInvalidAgentAttribute = status.Error(codes.InvalidArgument,
		"agent attribute must be of the form opentelemetry-<language_code> <ot_version>; google-cloud-trace-exporter <exporter_version>")
	statusMissingAgentAttribute    = status.Error(codes.InvalidArgument, "attributes must contain either g.co/agent or agent")
	statusUnmappedSpecialAttribute = status.Error(codes.InvalidArgument,
		"http.method, http.route and http.status_code should be translated to /http/method, /http/route and /http/status_code respectively")
	statusTooManyTimeEvents = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v time events", maxTimeEvents))
	statusInvalidAnnotation = status.Error(codes.InvalidArgument,
		fmt.Sprintf("annotation descriptions have a max length of %v bytes", maxAnnotationBytes))
	statusTooManyLinks = status.Error(codes.InvalidArgument,
		fmt.Sprintf("a span can have at most %v links", maxLinks))
	statusDuplicateSpanName = status.New(codes.AlreadyExists, "duplicate span name")

	// Metric statuses.
	statusDuplicateMetricDescriptorType = status.New(codes.AlreadyExists, "metric descriptor of same type already exists")
	statusMetricDescriptorNotFound      = status.New(codes.NotFound, "metric descriptor of given type does not exist")

	// Shared statuses.
	statusMissingField = status.New(codes.InvalidArgument, "missing required field(s)")
)
