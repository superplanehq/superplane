package grpc

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type HealthCheckServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (h *HealthCheckServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (h *HealthCheckServer) Watch(req *grpc_health_v1.HealthCheckRequest, srv grpc_health_v1.Health_WatchServer) error {
	return nil
}
