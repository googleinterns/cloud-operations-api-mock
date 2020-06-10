package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

const (
	invalidTimestampMsg   = "start time must be before end time"
	malformedTimestampMsg = "unable to parse timestamp"
	missingFieldMsg       = "span is missing required fields: %v"
)

var (
	requiredMessageFields = []string{"DisplayName", "StartTime", "EndTime"}
	requiredStringFields  = []string{"Name", "SpanId"}
)

func IsSpanValid(span *cloudtrace.Span) (bool, string) {
	if isValid, msg := validateRequiredFields(span); !isValid {
		return isValid, msg
	}

	if isValid, msg := validateTimeStamps(span); !isValid {
		return isValid, msg
	}

	return true, ""
}

func validateRequiredFields(span *cloudtrace.Span) (bool, string) {
	var missingFields []string
	spanReflect := reflect.ValueOf(span)

	for _, field := range requiredStringFields {
		fieldValue := reflect.Indirect(spanReflect).FieldByName(field).String()
		if fieldValue == "" {
			missingFields = append(missingFields, field)
		}
	}

	for _, field := range requiredMessageFields {
		fieldValue := reflect.Indirect(spanReflect).FieldByName(field)
		if fieldValue.IsNil() {
			missingFields = append(missingFields, field)
		}
	}

	formattedErrMsg := fmt.Sprintf(missingFieldMsg, strings.Join(missingFields, ", "))
	return len(missingFields) == 0, formattedErrMsg
}

func validateTimeStamps(span *cloudtrace.Span) (bool, string) {
	start, err := ptypes.Timestamp(span.StartTime)
	if err != nil {
		return false, malformedTimestampMsg
	}
	end, err := ptypes.Timestamp(span.EndTime)
	if err != nil {
		return false, malformedTimestampMsg
	}

	return start.Before(end), invalidTimestampMsg
}
