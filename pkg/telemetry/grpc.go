package telemetry

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

func criticalGRPCFilter(info *stats.RPCTagInfo) bool {
	return IsCriticalGRPCMethod(info.FullMethodName)
}

func CriticalGRPCServerStatsHandler() stats.Handler {
	return otelgrpc.NewServerHandler(
		otelgrpc.WithFilter(criticalGRPCFilter),
	)
}

func CriticalGRPCClientStatsHandler() stats.Handler {
	return otelgrpc.NewClientHandler(
		otelgrpc.WithFilter(criticalGRPCFilter),
	)
}

func GRPCGatewayDialOptions() []grpc.DialOption {
	if !TracingEnabled() {
		return nil
	}

	return []grpc.DialOption{
		grpc.WithStatsHandler(CriticalGRPCClientStatsHandler()),
	}
}
