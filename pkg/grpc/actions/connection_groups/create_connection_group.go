package connectiongroups

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateConnectionGroup(ctx context.Context, canvasID string, req *pb.CreateConnectionGroupRequest) (*pb.CreateConnectionGroupResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvasByIDOnly(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	logger := logging.ForCanvas(canvas)

	//
	// Validate request
	//
	if req.ConnectionGroup == nil || req.ConnectionGroup.Metadata == nil || req.ConnectionGroup.Metadata.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "connection group name is required")
	}

	connections, err := actions.ValidateConnections(canvas.ID.String(), req.ConnectionGroup.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	spec, err := validateSpec(req.ConnectionGroup.Spec)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	//
	// Create connection group
	//
	connectionGroup, err := canvas.CreateConnectionGroup(
		req.ConnectionGroup.Metadata.Name,
		req.ConnectionGroup.Metadata.Description,
		userID,
		connections,
		*spec,
	)

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		log.Errorf("Error creating connection group. Request: %v. Error: %v", req, err)
		return nil, err
	}

	group, err := serializeConnectionGroup(*connectionGroup, connections)
	if err != nil {
		return nil, err
	}

	response := &pb.CreateConnectionGroupResponse{
		ConnectionGroup: group,
	}

	err = messages.NewConnectionGroupCreatedMessage(connectionGroup).Publish()
	if err != nil {
		log.Errorf("failed to publish connection group created message: %v", err)
	}

	logger.Infof("Created connection group. Request: %v", req)

	return response, nil
}

func validateSpec(spec *pb.ConnectionGroup_Spec) (*models.ConnectionGroupSpec, error) {
	if spec == nil {
		return nil, fmt.Errorf("spec is required")
	}

	if spec.GroupBy == nil {
		return nil, fmt.Errorf("spec.GroupBy is required")
	}

	fields, err := validateGroupByFields(spec.GroupBy.Fields)
	if err != nil {
		return nil, err
	}

	err = validateTimeout(spec.Timeout)
	if err != nil {
		return nil, err
	}

	return &models.ConnectionGroupSpec{
		Timeout:         spec.Timeout,
		TimeoutBehavior: protoToTimeoutBehavior(spec.TimeoutBehavior),
		GroupBy: &models.ConnectionGroupBySpec{
			Fields: fields,
		},
	}, nil
}

func validateTimeout(timeout uint32) error {
	if timeout == 0 {
		return nil
	}

	if timeout < models.MinConnectionGroupTimeout || timeout > models.MaxConnectionGroupTimeout {
		return fmt.Errorf("timeout duration must be between %ds and %ds", models.MinConnectionGroupTimeout, models.MaxConnectionGroupTimeout)
	}

	return nil
}

func validateGroupByFields(in []*pb.ConnectionGroup_Spec_GroupBy_Field) ([]models.ConnectionGroupByField, error) {
	if len(in) < 1 {
		return nil, fmt.Errorf("connection group must have at least one field to group by")
	}

	out := make([]models.ConnectionGroupByField, len(in))
	for i, field := range in {
		if field.Name == "" || field.Expression == "" {
			return nil, fmt.Errorf("connection group field must have a name and an expression")
		}

		out[i] = models.ConnectionGroupByField{
			Name:       field.Name,
			Expression: field.Expression,
		}
	}

	return out, nil
}

func serializeConnectionGroup(connectionGroup models.ConnectionGroup, connections []models.Connection) (*pb.ConnectionGroup, error) {
	spec := connectionGroup.Spec.Data()
	conns, err := actions.SerializeConnections(connections)
	if err != nil {
		return nil, err
	}

	fields := make([]*pb.ConnectionGroup_Spec_GroupBy_Field, len(spec.GroupBy.Fields))
	for i, k := range spec.GroupBy.Fields {
		fields[i] = &pb.ConnectionGroup_Spec_GroupBy_Field{
			Name:       k.Name,
			Expression: k.Expression,
		}
	}

	metadata := &pb.ConnectionGroup_Metadata{
		Id:          connectionGroup.ID.String(),
		Name:        connectionGroup.Name,
		Description: connectionGroup.Description,
		CanvasId:    connectionGroup.CanvasID.String(),
		CreatedAt:   timestamppb.New(*connectionGroup.CreatedAt),
		CreatedBy:   connectionGroup.CreatedBy.String(),
	}

	if connectionGroup.UpdatedAt != nil {
		metadata.UpdatedAt = timestamppb.New(*connectionGroup.UpdatedAt)
		metadata.UpdatedBy = connectionGroup.UpdatedBy.String()
	}

	return &pb.ConnectionGroup{
		Metadata: metadata,
		Spec: &pb.ConnectionGroup_Spec{
			Connections:     conns,
			Timeout:         spec.Timeout,
			TimeoutBehavior: timeoutBehaviorToProto(spec.TimeoutBehavior),
			GroupBy: &pb.ConnectionGroup_Spec_GroupBy{
				Fields: fields,
			},
		},
	}, nil
}

func protoToTimeoutBehavior(behavior pb.ConnectionGroup_Spec_TimeoutBehavior) string {
	switch behavior {
	case pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_EMIT:
		return models.ConnectionGroupTimeoutBehaviorEmit
	case pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_DROP:
		return models.ConnectionGroupTimeoutBehaviorDrop
	default:
		return models.ConnectionGroupTimeoutBehaviorNone
	}
}

func timeoutBehaviorToProto(timeoutBehavior string) pb.ConnectionGroup_Spec_TimeoutBehavior {
	switch timeoutBehavior {
	case models.ConnectionGroupTimeoutBehaviorEmit:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_EMIT
	case models.ConnectionGroupTimeoutBehaviorDrop:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_DROP
	default:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_NONE
	}
}
