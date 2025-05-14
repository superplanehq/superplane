package grpcmock

import (
	"net"

	schedulepb "github.com/superplanehq/superplane/pkg/protos/periodic_scheduler"
	pplpb "github.com/superplanehq/superplane/pkg/protos/plumber.pipeline"
	repoproxypb "github.com/superplanehq/superplane/pkg/protos/repo_proxy"
	"google.golang.org/grpc"
)

var started = false
var registry *ServiceRegistry

type ServiceRegistry struct {
	RepoProxyService *RepoProxyService
	PipelineService  *PipelineService
	SchedulerService *SchedulerService
}

func Start() (*ServiceRegistry, error) {
	if started {
		return registry, nil
	}

	started = true

	// #nosec
	lis, err := net.Listen("tcp", "0.0.0.0:50052")
	if err != nil {
		return nil, err
	}

	registry = &ServiceRegistry{
		RepoProxyService: NewRepoProxyService(),
		PipelineService:  NewPipelineService(),
		SchedulerService: NewSchedulerService(),
	}

	grpcServer := grpc.NewServer()
	repoproxypb.RegisterRepoProxyServiceServer(grpcServer, registry.RepoProxyService)
	pplpb.RegisterPipelineServiceServer(grpcServer, registry.PipelineService)
	schedulepb.RegisterPeriodicServiceServer(grpcServer, registry.SchedulerService)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	return registry, nil
}
