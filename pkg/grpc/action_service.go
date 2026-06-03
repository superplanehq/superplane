package grpc

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
		return nil, status.Errorf(codes.NotFound, "action %s not found", req.Name)
	}

	outputChannels := action.OutputChannels(nil)
	channels := make([]*pb.OutputChannel, len(outputChannels))
	for i, channel := range outputChannels {
		channels[i] = &pb.OutputChannel{
			Name: channel.Name,
		}
	}

	configFields := action.Configuration()
	configuration := make([]*configpb.Field, len(configFields))
	for i, field := range configFields {
		configuration[i] = actions.ConfigurationFieldToProto(field)
	}

	var exampleOutput *structpb.Struct
	if output := action.ExampleOutput(); output != nil {
		exampleOutput, _ = structpb.NewStruct(output)
	}

	return &pb.DescribeActionResponse{
		Action: &pb.Action{
			Name:           action.Name(),
			Label:          action.Label(),
			Description:    action.Description(),
			Icon:           action.Icon(),
			Color:          action.Color(),
			OutputChannels: channels,
			Configuration:  configuration,
			ExampleOutput:  exampleOutput,
		},
	}, nil
}
