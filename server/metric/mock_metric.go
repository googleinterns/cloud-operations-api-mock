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

package metric

import (
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/googleinterns/cloud-operations-api-mock/internal/validation"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
)

// MockMetricServer implements all of the RPCs pertaining to
// tracing that can be called by the client. It also contains the
// uploaded data.
type MockMetricServer struct {
	monitoring.UnimplementedMetricServiceServer
	metricDescriptorData *validation.MetricDescriptorData
	uploadedPoints       map[string]*validation.PreviousPoint
	uploadedPointsLock   sync.Mutex
}

// NewMockMetricServer creates a new MockMetricServer and returns a pointer to it.
func NewMockMetricServer() *MockMetricServer {
	uploadedMetricDescriptors := make(map[string]*metric.MetricDescriptor)
	metricDescriptorData := &validation.MetricDescriptorData{
		UploadedMetricDescriptors: uploadedMetricDescriptors,
	}
	uploadedPoints := make(map[string]*validation.PreviousPoint)
	return &MockMetricServer{
		metricDescriptorData: metricDescriptorData,
		uploadedPoints:       uploadedPoints,
	}
}

// GetMonitoredResourceDescriptor returns the requested monitored resource descriptor if it exists.
func (s *MockMetricServer) GetMonitoredResourceDescriptor(ctx context.Context, req *monitoring.GetMonitoredResourceDescriptorRequest,
) (*monitoredres.MonitoredResourceDescriptor, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}
	return &monitoredres.MonitoredResourceDescriptor{}, nil
}

// ListMonitoredResourceDescriptors list all the requested monitored resource descriptors
// that are picked up by the given query.
func (s *MockMetricServer) ListMonitoredResourceDescriptors(ctx context.Context, req *monitoring.ListMonitoredResourceDescriptorsRequest,
) (*monitoring.ListMonitoredResourceDescriptorsResponse, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}
	return &monitoring.ListMonitoredResourceDescriptorsResponse{
		ResourceDescriptors: []*monitoredres.MonitoredResourceDescriptor{},
		NextPageToken:       "",
	}, nil
}

// GetMetricDescriptor returns the requested metric descriptor.
// If it doesn't esxist, an error is returned.
func (s *MockMetricServer) GetMetricDescriptor(ctx context.Context, req *monitoring.GetMetricDescriptorRequest,
) (*metric.MetricDescriptor, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}

	s.metricDescriptorData.MetricDescriptorsLock.Lock()
	defer s.metricDescriptorData.MetricDescriptorsLock.Unlock()
	metricDescriptor, err := validation.AccessMetricDescriptor(s.metricDescriptorData.UploadedMetricDescriptors, req.Name)
	if err != nil {
		return nil, err
	}

	return metricDescriptor, nil
}

// CreateMetricDescriptor stores the given metric descriptor in memory.
// If it already exists, an error is returned.
func (s *MockMetricServer) CreateMetricDescriptor(ctx context.Context, req *monitoring.CreateMetricDescriptorRequest,
) (*metric.MetricDescriptor, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		addMetricDescriptorToSummary(&s.metricDescriptorData.MetricDescriptorSummary, req.MetricDescriptor, err.Error())
		return nil, err
	}

	if err := validation.ValidateProjectName(req.Name); err != nil {
		addMetricDescriptorToSummary(&s.metricDescriptorData.MetricDescriptorSummary, req.MetricDescriptor, err.Error())
		return nil, err
	}

	if err := validation.ValidateCreateMetricDescriptor(req.MetricDescriptor); err != nil {
		addMetricDescriptorToSummary(&s.metricDescriptorData.MetricDescriptorSummary, req.MetricDescriptor, err.Error())
		return nil, err
	}

	s.metricDescriptorData.MetricDescriptorsLock.Lock()
	defer s.metricDescriptorData.MetricDescriptorsLock.Unlock()
	if err := validation.AddMetricDescriptor(s.metricDescriptorData.UploadedMetricDescriptors, req.MetricDescriptor.Type, req.MetricDescriptor); err != nil {
		addMetricDescriptorToSummary(&s.metricDescriptorData.MetricDescriptorSummary, req.MetricDescriptor, err.Error())
		return nil, err
	}

	addMetricDescriptorToSummary(&s.metricDescriptorData.MetricDescriptorSummary, req.MetricDescriptor, "OK")
	return req.MetricDescriptor, nil
}

// DeleteMetricDescriptor deletes the given metric descriptor from memory.
// If it doesn't exist, an error is returned.
func (s *MockMetricServer) DeleteMetricDescriptor(ctx context.Context, req *monitoring.DeleteMetricDescriptorRequest,
) (*empty.Empty, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}

	s.metricDescriptorData.MetricDescriptorsLock.Lock()
	defer s.metricDescriptorData.MetricDescriptorsLock.Unlock()
	if err := validation.RemoveMetricDescriptor(s.metricDescriptorData.UploadedMetricDescriptors, req.Name); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

// ListMetricDescriptors lists all the metric descriptors that are picked up by the given query.
func (s *MockMetricServer) ListMetricDescriptors(ctx context.Context, req *monitoring.ListMetricDescriptorsRequest,
) (*monitoring.ListMetricDescriptorsResponse, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}
	return &monitoring.ListMetricDescriptorsResponse{
		MetricDescriptors: []*metric.MetricDescriptor{},
		NextPageToken:     "",
	}, nil
}

// CreateTimeSeries stores the given time series in memory.
// If it already exists, an error is returned.
func (s *MockMetricServer) CreateTimeSeries(ctx context.Context, req *monitoring.CreateTimeSeriesRequest,
) (*empty.Empty, error) {
	s.uploadedPointsLock.Lock()
	if err := validation.ValidateRateLimit(req.TimeSeries, s.uploadedPoints); err != nil {
		s.uploadedPointsLock.Unlock()
		return nil, err
	}
	s.uploadedPointsLock.Unlock()
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}
	if err := validation.ValidateProjectName(req.Name); err != nil {
		return nil, err
	}
	if err := validation.ValidateCreateTimeSeries(req.TimeSeries, s.metricDescriptorData.UploadedMetricDescriptors, s.uploadedPoints); err != nil {
		return nil, err
	}
	s.uploadedPointsLock.Lock()
	defer s.uploadedPointsLock.Unlock()
	validation.AddPoint(req.TimeSeries, s.uploadedPoints)
	return &empty.Empty{}, nil
}

// ListTimeSeries lists all time series that are picked up by the given query.
func (s *MockMetricServer) ListTimeSeries(ctx context.Context, req *monitoring.ListTimeSeriesRequest,
) (*monitoring.ListTimeSeriesResponse, error) {
	if err := validation.ValidateRequiredFields(req); err != nil {
		return nil, err
	}
	return &monitoring.ListTimeSeriesResponse{
		TimeSeries:      []*monitoring.TimeSeries{},
		NextPageToken:   "",
		ExecutionErrors: []*status.Status{},
	}, nil
}

// addMetricDescriptorToSummary adds the given metric descriptor and status to the summary.
func addMetricDescriptorToSummary(summary *[]*validation.DescriptorStatus, metricDescriptor *metric.MetricDescriptor, err string) {
	metricDescriptorStatus := &validation.DescriptorStatus{
		MetricDescriptor: metricDescriptor,
		Status:           err,
	}
	*summary = append(*summary, metricDescriptorStatus)
}

// MetricDescriptorSummary returns the metric descriptor data to display in the summary page.
func (s *MockMetricServer) MetricDescriptorSummary() []*validation.DescriptorStatus {
	return s.metricDescriptorData.MetricDescriptorSummary
}
