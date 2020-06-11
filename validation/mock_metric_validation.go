package validation

import (
	"fmt"
	"reflect"

	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

const (
	missingFieldMsg = "%v must contain requried %v field"
)

func IsValidGetMetricDescriptorRequest(req *monitoring.GetMetricDescriptorRequest) error {
	reqReflect := reflect.ValueOf(req)
	requiredFields := []string{"Name"}
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(reqReflect).FieldByName(field).IsZero(); isZero {
			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: fmt.Sprintf(missingFieldMsg, "GetMetricDescriptorRequest", field),
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

func IsValidDeleteMetricDescriptorRequest(req *monitoring.DeleteMetricDescriptorRequest) error {
	reqReflect := reflect.ValueOf(req)
	requiredFields := []string{"Name"}
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(reqReflect).FieldByName(field).IsZero(); isZero {
			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: fmt.Sprintf(missingFieldMsg, "DeleteMetricDescriptorRequest", field),
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

func IsValidCreateMetricDescriptorRequest(req *monitoring.CreateMetricDescriptorRequest) error {
	reqReflect := reflect.ValueOf(req)
	requiredFields := []string{"Name", "MetricDescriptor"}
	br := &errdetails.BadRequest{}

	for _, field := range requiredFields {
		if isZero := reflect.Indirect(reqReflect).FieldByName(field).IsZero(); isZero {
			v := &errdetails.BadRequest_FieldViolation{
				Field:       field,
				Description: fmt.Sprintf(missingFieldMsg, "CreateMetricDescriptorRequest", field),
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
