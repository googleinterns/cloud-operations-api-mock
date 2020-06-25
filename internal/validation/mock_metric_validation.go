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
	"reflect"
	"sync"
	"fmt"

	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrDuplicateMetricDescriptorName = status.New(codes.AlreadyExists, "metric descriptor with same name already exists")
)

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

func AddMetricDescriptor(lock *sync.Mutex, uploadedMetricDescriptors map[string]*metric.MetricDescriptor, name string, metricDescriptor *metric.MetricDescriptor) error {
	lock.Lock()
	defer lock.Unlock()

	if _, ok := uploadedMetricDescriptors[name]; ok {
		br := &errdetails.ErrorInfo{}
		br.Reason = name
		st, err := ErrDuplicateMetricDescriptorName.WithDetails(br)
		if err != nil {
			panic(fmt.Sprintf("unexpected error attaching metadata: %v", err))
		}
		return st.Err()
	}

	uploadedMetricDescriptors[name] = metricDescriptor

	return nil
}