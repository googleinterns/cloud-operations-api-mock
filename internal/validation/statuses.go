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
