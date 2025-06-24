package connectiongroups

import (
	"context"
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateConnectionGroup(ctx context.Context, req *pb.CreateConnectionGroupRequest) (*pb.CreateConnectionGroupResponse, error) {
	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

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

	err = actions.ValidateUUIDs(req.RequesterId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvas, req.ConnectionGroup.Spec.Connections)
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
		req.RequesterId,
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

	if len(spec.GroupBy.Fields) == 0 {
		return nil, fmt.Errorf("spec.GroupBy fields cannot be empty")
	}

	fields, err := validateGroupByFields(spec.GroupBy.Fields)
	if err != nil {
		return nil, err
	}

	return &models.ConnectionGroupSpec{
		GroupBy: &models.ConnectionGroupBySpec{
			Fields: fields,
			EmitOn: protoToEmitOn(spec.GroupBy.EmitOn),
		},
	}, nil
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

	return &pb.ConnectionGroup{
		Metadata: &pb.ConnectionGroup_Metadata{
			Id:        connectionGroup.ID.String(),
			Name:      connectionGroup.Name,
			CanvasId:  connectionGroup.CanvasID.String(),
			CreatedAt: timestamppb.New(*connectionGroup.CreatedAt),
			CreatedBy: connectionGroup.CreatedBy.String(),
		},
		Spec: &pb.ConnectionGroup_Spec{
			Connections: conns,
			GroupBy: &pb.ConnectionGroup_Spec_GroupBy{
				Fields: fields,
				EmitOn: emitOnToProto(spec.GroupBy.EmitOn),
			},
		},
	}, nil
}

func protoToEmitOn(emitOn pb.ConnectionGroup_Spec_GroupBy_EmitOn) string {
	switch emitOn {
	case pb.ConnectionGroup_Spec_GroupBy_EMIT_ON_MAJORITY:
		return models.ConnectionGroupEmitOnMajority
	default:
		return models.ConnectionGroupEmitOnAll
	}
}

func emitOnToProto(emitOn string) pb.ConnectionGroup_Spec_GroupBy_EmitOn {
	switch emitOn {
	case models.ConnectionGroupEmitOnMajority:
		return pb.ConnectionGroup_Spec_GroupBy_EMIT_ON_MAJORITY
	default:
		return pb.ConnectionGroup_Spec_GroupBy_EMIT_ON_ALL
	}
}
