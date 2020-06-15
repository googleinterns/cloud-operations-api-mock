package validation

import (
	"reflect"

	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes"
)

var (
	InvalidTimestampErr   = status.Error(codes.InvalidArgument, "start time must be before end time")
	MalformedTimestampErr = status.Error(codes.InvalidArgument, "unable to parse timestamp")
	MissingFieldError     = status.New(codes.InvalidArgument, "Missing required field(s)")
	requiredFields        = []string{"Name", "SpanId", "DisplayName", "StartTime", "EndTime"}
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
	return CheckForRequiredFields(requiredFields, reflect.ValueOf(span), "Span")
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
