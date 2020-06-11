package validation

import (
	"fmt"
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
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
	MissingFieldError     = status.New(codes.InvalidArgument, "Missing required field(s)")
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
	spanReflect := reflect.ValueOf(span)
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(spanReflect).FieldByName(field).IsZero(); isZero {
			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: fmt.Sprintf("Span must contain requried %v field", field),
			}
			br.FieldViolations = append(br.FieldViolations, v)
		}
	}

	if len(br.FieldViolations) == 0 {
		return nil
	} else {
		st, err := MissingFieldError.WithDetails(br)

		if err != nil {
			panic(fmt.Sprintf("Unexpected error attaching metadata: %v", err))
		}

		return st.Err()
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
