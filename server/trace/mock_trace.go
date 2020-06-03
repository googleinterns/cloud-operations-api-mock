package trace

import (
	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/devtools/cloudtrace/v2"
)

type MockTraceServer struct {
        cloudtrace.UnimplementedTraceServiceServer
}

func (s *MockTraceServer) BatchWriteSpans(ctx context.Context, req *cloudtrace.BatchWriteSpansRequest) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (s *MockTraceServer) CreateSpan(ctx context.Context, span *cloudtrace.Span) (*cloudtrace.Span, error) {
	return span, nil
}
