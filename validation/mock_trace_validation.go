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
	code          codes.Code
	missingFields []string
}

func (err *MissingFieldError) Error() string {
	return fmt.Sprintf("span is missing required fields: %v",
		strings.Join(err.missingFields, ", "))
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
			code:          codes.InvalidArgument,
			missingFields: missingFields,
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
