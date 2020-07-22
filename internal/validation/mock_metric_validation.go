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
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes"

	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

const (
	// Time series constraints can be found at https://cloud.google.com/monitoring/quotas#custom_metrics_quotas.
	maxTimeSeriesPerRequest      = 200
	maxTimeSeriesLabelKeyBytes   = 100
	maxTimeSeriesLabelValueBytes = 1024
)

// ValidRequiredFields verifies that the given request contains the required fields.
func ValidRequiredFields(req interface{}) error {
	reqReflect := reflect.ValueOf(req)
	requiredFields := []string{"Name"}
	requestName := ""

	switch req.(type) {
	case *monitoring.CreateMetricDescriptorRequest:
		requiredFields = append(requiredFields, "MetricDescriptor")
		requestName = "CreateMetricDescriptorRequest"
	case *monitoring.GetMetricDescriptorRequest:
		requestName = "GetMetricDescriptorRequest"
	case *monitoring.DeleteMetricDescriptorRequest:
		requestName = "DeleteMetricDescriptorRequest"
	case *monitoring.GetMonitoredResourceDescriptorRequest:
		requestName = "GetMonitoredResourceDescriptorRequest"
	case *monitoring.ListMetricDescriptorsRequest:
		requestName = "ListMetricDescriptorsRequest"
	case *monitoring.ListMonitoredResourceDescriptorsRequest:
		requestName = "ListMonitoredResourceDescriptorsRequest"
	case *monitoring.ListTimeSeriesRequest:
		requestName = "ListTimeSeriesRequest"
	case *monitoring.CreateTimeSeriesRequest:
		requiredFields = append(requiredFields, "TimeSeries")
		requestName = "CreateTimeSeriesRequest"
	}

	return CheckForRequiredFields(requiredFields, reqReflect, requestName)
}

// AddMetricDescriptor adds a new MetricDescriptor to the map if a duplicate type does not already exist.
// If a duplicate is detected, an error is returned.
func AddMetricDescriptor(uploadedMetricDescriptors map[string]*metric.MetricDescriptor, metricType string, metricDescriptor *metric.MetricDescriptor) error {
	if _, ok := uploadedMetricDescriptors[metricType]; ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = metricType
		st, err := statusDuplicateMetricDescriptorType.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}

	uploadedMetricDescriptors[metricType] = metricDescriptor

	return nil
}

// AccessMetricDescriptor attempts to retrieve the given metric descriptor.
// If it does / exist, an error is returned.
func AccessMetricDescriptor(uploadedMetricDescriptors map[string]*metric.MetricDescriptor, metricType string) (*metric.MetricDescriptor, error) {
	if _, ok := uploadedMetricDescriptors[metricType]; !ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = metricType
		st, err := statusMetricDescriptorNotFound.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return nil, st.Err()
	}

	return uploadedMetricDescriptors[metricType], nil
}

// RemoveMetricDescriptor attempts to delete the given metric descriptor.
// If it does not exist, an error is returned.
func RemoveMetricDescriptor(uploadedMetricDescriptors map[string]*metric.MetricDescriptor, metricType string) error {
	if _, ok := uploadedMetricDescriptors[metricType]; !ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = metricType
		st, err := statusMetricDescriptorNotFound.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}

	delete(uploadedMetricDescriptors, metricType)

	return nil
}

// ValidateCreateTimeSeries checks that the given TimeSeries conform to the API requirements.
func ValidateCreateTimeSeries(timeSeries []*monitoring.TimeSeries, descriptors map[string]*metric.MetricDescriptor) error {
	if len(timeSeries) > maxTimeSeriesPerRequest {
		return statusTooManyTimeSeries
	}

	for _, ts := range timeSeries {
		// Check that required fields for time series are present.
		if ts.Metric == nil || len(ts.Points) != 1 || ts.Resource == nil {
			return statusInvalidTimeSeries
		}

		// Check that the metric labels follow the constraints.
		for k, v := range ts.Metric.Labels {
			if len(k) > maxTimeSeriesLabelKeyBytes {
				return statusInvalidTimeSeriesLabelKey
			}

			if len(v) > maxTimeSeriesLabelValueBytes {
				return statusInvalidTimeSeriesLabelValue
			}
		}

		if err := validateMetricKind(ts, descriptors); err != nil {
			return err
		}

		if err := validateValueType(ts.ValueType, ts.Points[0]); err != nil {
			return err
		}

		if err := validatePoint(ts.MetricKind, ts.Points[0]); err != nil {
			return err
		}
	}

	return nil
}

// validateMetricKind check that if metric_kind is present,
// it is the same as the metricKind of the associated metric.
func validateMetricKind(timeSeries *monitoring.TimeSeries, descriptors map[string]*metric.MetricDescriptor) error {
	descriptor := descriptors[timeSeries.Metric.Type]
	if descriptor == nil {
		return statusMissingMetricDescriptor
	}
	if descriptor.MetricKind != timeSeries.MetricKind {
		return statusInvalidTimeSeriesMetricKind
	}

	return nil
}

// validateValueType checks that if TimeSeries' "value_type" is present,
// it is the same as the type of the data in the "points" field.
func validateValueType(valueType metric.MetricDescriptor_ValueType, point *monitoring.Point) error {
	if valueType == metric.MetricDescriptor_BOOL {
		if _, ok := point.Value.Value.(*monitoring.TypedValue_BoolValue); !ok {
			return statusInvalidTimeSeriesValueType
		}
	}
	if valueType == metric.MetricDescriptor_INT64 {
		if _, ok := point.Value.Value.(*monitoring.TypedValue_Int64Value); !ok {
			return statusInvalidTimeSeriesValueType
		}
	}
	if valueType == metric.MetricDescriptor_DOUBLE {
		if _, ok := point.Value.Value.(*monitoring.TypedValue_DoubleValue); !ok {
			return statusInvalidTimeSeriesValueType
		}
	}
	if valueType == metric.MetricDescriptor_STRING {
		if _, ok := point.Value.Value.(*monitoring.TypedValue_StringValue); !ok {
			return statusInvalidTimeSeriesValueType
		}
	}
	if valueType == metric.MetricDescriptor_DISTRIBUTION {
		if _, ok := point.Value.Value.(*monitoring.TypedValue_DistributionValue); !ok {
			return statusInvalidTimeSeriesValueType
		}
	}

	return nil
}

// validatePoint checks that the data for the point is valid.
// The specifics checks depend on the metric_kind.
func validatePoint(metricKind metric.MetricDescriptor_MetricKind, point *monitoring.Point) error {
	startTime, err := ptypes.Timestamp(point.Interval.StartTime)
	// If metric_kind is GAUGE, the startTime is optional.
	if err != nil && metricKind != metric.MetricDescriptor_GAUGE {
		return statusMalformedTimestamp
	}
	endTime, err := ptypes.Timestamp(point.Interval.EndTime)
	if err != nil {
		return statusMalformedTimestamp
	}

	// If metric_kind is GAUGE, if start time is supplied, it must equal the end time.
	if metricKind == metric.MetricDescriptor_GAUGE {
		// If start time is nil, ptypes.Timestamp will return time.Unix(0, 0).UTC().
		if startTime != time.Unix(0, 0).UTC() && !startTime.Equal(endTime) {
			return statusInvalidTimeSeriesPointGauge
		}
	}

	// TODO (ejwang): also need checks for metric.MetricDescriptor_DELTA and metric.MetricDescriptor_CUMULATIVE,
	//  however they involve comparing against previous time series, so we'll need to store time series in memory.
	//  Leaving this for another PR since this PR is already quite large.

	return nil
}
