package grpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	pb "github.com/superplanehq/superplane/pkg/protos/actions"
	"github.com/superplanehq/superplane/pkg/registry"
)

type ActionService struct {
	registry *registry.Registry
}

func NewActionService(registry *registry.Registry) *ActionService {
	return &ActionService{registry: registry}
}

func (s *ActionService) ListActions(ctx context.Context, req *pb.ListActionsRequest) (*pb.ListActionsResponse, error) {
	return &pb.ListActionsResponse{
		Actions: actions.SerializeActions(s.registry.ListActions()),
	}, nil
}

func (s *ActionService) DescribeAction(ctx context.Context, req *pb.DescribeActionRequest) (*pb.DescribeActionResponse, error) {
	action, err := s.registry.GetAction(req.Name)
	if err != nil {
		return nil, grpcerrors.NotFound(err, fmt.Sprintf("action %s not found", req.Name))
	}

	serialized := actions.SerializeAction(action)
	if serialized == nil {
		return nil, grpcerrors.Internal(errors.New("serialize action panicked"), "failed to serialize action")
	}

	return &pb.DescribeActionResponse{Action: serialized}, nil
}
