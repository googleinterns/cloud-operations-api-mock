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

type MockMetricServer struct {
	monitoring.UnimplementedMetricServiceServer
	uploadedMetricDescriptors     map[string]*metric.MetricDescriptor
	uploadedMetricDescriptorsLock sync.Mutex
}

func NewMockMetricServer() *MockMetricServer {
	uploadedMetricDescriptors := make(map[string]*metric.MetricDescriptor)
	return &MockMetricServer{uploadedMetricDescriptors: uploadedMetricDescriptors}
}

func (s *MockMetricServer) GetMonitoredResourceDescriptor(ctx context.Context, req *monitoring.GetMonitoredResourceDescriptorRequest,
) (*monitoredres.MonitoredResourceDescriptor, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}
	return &monitoredres.MonitoredResourceDescriptor{}, nil
}

func (s *MockMetricServer) ListMonitoredResourceDescriptors(ctx context.Context, req *monitoring.ListMonitoredResourceDescriptorsRequest,
) (*monitoring.ListMonitoredResourceDescriptorsResponse, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}
	return &monitoring.ListMonitoredResourceDescriptorsResponse{
		ResourceDescriptors: []*monitoredres.MonitoredResourceDescriptor{},
		NextPageToken:       "",
	}, nil
}

func (s *MockMetricServer) GetMetricDescriptor(ctx context.Context, req *monitoring.GetMetricDescriptorRequest,
) (*metric.MetricDescriptor, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}

	metricDescriptor, err := validation.AccessMetricDescriptor(&s.uploadedMetricDescriptorsLock, s.uploadedMetricDescriptors, req.Name)
	if err != nil {
		return nil, err
	}

	return metricDescriptor, nil
}

func (s *MockMetricServer) CreateMetricDescriptor(ctx context.Context, req *monitoring.CreateMetricDescriptorRequest,
) (*metric.MetricDescriptor, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}

	if err := validation.AddMetricDescriptor(&s.uploadedMetricDescriptorsLock, s.uploadedMetricDescriptors, req.Name, req.MetricDescriptor); err != nil {
		return nil, err
	}

	return req.MetricDescriptor, nil
}

func (s *MockMetricServer) DeleteMetricDescriptor(ctx context.Context, req *monitoring.DeleteMetricDescriptorRequest,
) (*empty.Empty, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}

	if err := validation.RemoveMetricDescriptor(&s.uploadedMetricDescriptorsLock, s.uploadedMetricDescriptors, req.Name); err != nil {
		return nil, err
	}

	return &empty.Empty{}, nil
}

func (s *MockMetricServer) ListMetricDescriptors(ctx context.Context, req *monitoring.ListMetricDescriptorsRequest,
) (*monitoring.ListMetricDescriptorsResponse, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}
	return &monitoring.ListMetricDescriptorsResponse{
		MetricDescriptors: []*metric.MetricDescriptor{},
		NextPageToken:     "",
	}, nil
}

func (s *MockMetricServer) CreateTimeSeries(ctx context.Context, req *monitoring.CreateTimeSeriesRequest,
) (*empty.Empty, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

func (s *MockMetricServer) ListTimeSeries(ctx context.Context, req *monitoring.ListTimeSeriesRequest,
) (*monitoring.ListTimeSeriesResponse, error) {
	if err := validation.IsValidRequest(req); err != nil {
		return nil, err
	}
	return &monitoring.ListTimeSeriesResponse{
		TimeSeries:      []*monitoring.TimeSeries{},
		NextPageToken:   "",
		ExecutionErrors: []*status.Status{},
	}, nil
}
