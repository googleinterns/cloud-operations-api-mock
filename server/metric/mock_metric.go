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
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/api/metric"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	"google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
)

type MockMetricServer struct {
	monitoring.UnimplementedMetricServiceServer
}

func (s *MockMetricServer) ListMonitoredResourceDescriptors(ctx context.Context, req *monitoring.ListMonitoredResourceDescriptorsRequest,
) (*monitoring.ListMonitoredResourceDescriptorsResponse, error) {
	return &monitoring.ListMonitoredResourceDescriptorsResponse{
		ResourceDescriptors: []*monitoredres.MonitoredResourceDescriptor{},
		NextPageToken:       "",
	}, nil
}

func (s *MockMetricServer) ListMetricDescriptors(ctx context.Context, req *monitoring.ListMetricDescriptorsRequest,
) (*monitoring.ListMetricDescriptorsResponse, error) {
	return &monitoring.ListMetricDescriptorsResponse{
		MetricDescriptors: []*metric.MetricDescriptor{},
		NextPageToken:     "",
	}, nil
}

func (s *MockMetricServer) ListTimeSeries(ctx context.Context, req *monitoring.ListTimeSeriesRequest,
) (*monitoring.ListTimeSeriesResponse, error) {
	return &monitoring.ListTimeSeriesResponse{
		TimeSeries:      []*monitoring.TimeSeries{},
		NextPageToken:   "",
		ExecutionErrors: []*status.Status{},
	}, nil
}