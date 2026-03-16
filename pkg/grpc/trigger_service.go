package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/triggers"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
)

type TriggerService struct {
	registry *registry.Registry
}

func NewTriggerService(registry *registry.Registry) *TriggerService {
	return &TriggerService{registry: registry}
}

func (s *TriggerService) ListTriggers(ctx context.Context, req *pb.ListTriggersRequest) (*pb.ListTriggersResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return triggers.ListTriggers(ctx, s.registry, organizationID)
}

func (s *TriggerService) DescribeTrigger(ctx context.Context, req *pb.DescribeTriggerRequest) (*pb.DescribeTriggerResponse, error) {
	organizationID := ctx.Value(authorization.OrganizationContextKey).(string)
	return triggers.DescribeTrigger(ctx, s.registry, organizationID, req.Name)
}
