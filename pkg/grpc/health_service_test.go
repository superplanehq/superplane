package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/health/grpc_health_v1"
)

func Test__Check(t *testing.T) {
	healthServer := &HealthCheckServer{}

	resp, err := healthServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("Health check failed with error: %v", err)
	}

	if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("Expected health status %v, got %v", grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	}
}

func Test__Watch(t *testing.T) {
	healthServer := &HealthCheckServer{}

	err := healthServer.Watch(&grpc_health_v1.HealthCheckRequest{}, nil)
	if err != nil {
		t.Fatalf("Health watch failed with error: %v", err)
	}
}
