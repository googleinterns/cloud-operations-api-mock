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
	"sync"

	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
)

// IsValidRequest verifies that the given request is valid.
// This means required fields are present and all fields semantically make sense.
func IsValidRequest(req interface{}) error {
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

// AddMetricDescriptor adds a new MetricDescriptor to the map if a duplicate does not already exist.
// If a duplciate is detected, an error is returned.
func AddMetricDescriptor(lock *sync.Mutex, uploadedMetricDescriptors map[string]*metric.MetricDescriptor, name string, metricDescriptor *metric.MetricDescriptor) error {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := uploadedMetricDescriptors[name]; ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = name
		st, err := statusDuplicateMetricDescriptorName.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}

	uploadedMetricDescriptors[name] = metricDescriptor

	return nil
}

// AccessMetricDescriptor attempts to retrieve the given metric descriptor.
// If it does not exist, an error is returned.
func AccessMetricDescriptor(lock *sync.Mutex, uploadedMetricDescriptors map[string]*metric.MetricDescriptor, name string) (*metric.MetricDescriptor, error) {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := uploadedMetricDescriptors[name]; !ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = name
		st, err := statusMetricDescriptorNotFound.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return nil, st.Err()
	}

	return uploadedMetricDescriptors[name], nil
}

// RemoveMetricDescriptor attempts to delete the given metric descriptor.
// If it does not exist, an error is returned.
func RemoveMetricDescriptor(lock *sync.Mutex, uploadedMetricDescriptors map[string]*metric.MetricDescriptor, name string) error {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := uploadedMetricDescriptors[name]; !ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = name
		st, err := statusMetricDescriptorNotFound.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}

	delete(uploadedMetricDescriptors, name)

	return nil
}
