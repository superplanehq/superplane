package connectiongroups

import (
	"context"
	"errors"
	"fmt"
	"time"

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

	policy, err := validatePolicy(req.ConnectionGroup.Spec.Policy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	groupByKeys, err := validateGroupByKeys(req.ConnectionGroup.Spec.Keys)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvas, req.ConnectionGroup.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	//
	// Create connection group
	//
	connectionGroup, err := canvas.CreateConnectionGroup(
		req.ConnectionGroup.Metadata.Name,
		req.ConnectionGroup.Metadata.CreatedBy,
		connections,
		models.ConnectionGroupSpec{
			Policy: *policy,
			Keys:   groupByKeys,
		},
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

func validateGroupByKeys(in []*pb.ConnectionGroup_Spec_Key) ([]models.ConnectionGroupKeyDefinition, error) {
	if len(in) < 1 {
		return nil, fmt.Errorf("connection group must have at least one key to group by")
	}

	out := make([]models.ConnectionGroupKeyDefinition, len(in))
	for i, key := range in {
		if key.Name == "" || key.Expression == "" {
			return nil, fmt.Errorf("connection group key must have a name and an expression")
		}

		out[i] = models.ConnectionGroupKeyDefinition{
			Name:       key.Name,
			Expression: key.Expression,
		}
	}

	return out, nil
}

func validatePolicy(policy *pb.ConnectionGroup_Spec_Policy) (*models.ConnectionGroupPolicy, error) {
	if policy == nil {
		return nil, fmt.Errorf("policy is required")
	}

	_, err := time.ParseDuration(policy.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout: %v", err)
	}

	return &models.ConnectionGroupPolicy{
		Type:            protoToPolicyType(policy.Type),
		Timeout:         policy.Timeout,
		TimeoutBehavior: protoToTimeoutBehavior(policy.TimeoutBehavior),
	}, nil
}

func serializeConnectionGroup(connectionGroup models.ConnectionGroup, connections []models.Connection) (*pb.ConnectionGroup, error) {
	spec := connectionGroup.Spec.Data()
	conns, err := actions.SerializeConnections(connections)
	if err != nil {
		return nil, err
	}

	keys := make([]*pb.ConnectionGroup_Spec_Key, len(spec.Keys))
	for i, k := range spec.Keys {
		keys[i] = &pb.ConnectionGroup_Spec_Key{
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
		},
		Spec: &pb.ConnectionGroup_Spec{
			Keys:        keys,
			Connections: conns,
			Policy: &pb.ConnectionGroup_Spec_Policy{
				Type:            policyTypeToProto(spec.Policy.Type),
				Timeout:         spec.Policy.Timeout,
				TimeoutBehavior: timeoutBehaviorToProto(spec.Policy.TimeoutBehavior),
			},
		},
	}, nil
}

func protoToPolicyType(policyType pb.ConnectionGroup_Spec_PolicyType) string {
	switch policyType {
	case pb.ConnectionGroup_Spec_POLICY_TYPE_MAJORITY:
		return models.ConnectionGroupPolicyTypeMajority
	default:
		return models.ConnectionGroupPolicyTypeAll
	}
}

func policyTypeToProto(policyType string) pb.ConnectionGroup_Spec_PolicyType {
	switch policyType {
	case models.ConnectionGroupPolicyTypeMajority:
		return pb.ConnectionGroup_Spec_POLICY_TYPE_MAJORITY
	default:
		return pb.ConnectionGroup_Spec_POLICY_TYPE_ALL
	}
}

func timeoutBehaviorToProto(timeoutBehavior string) pb.ConnectionGroup_Spec_TimeoutBehavior {
	switch timeoutBehavior {
	case models.ConnectionGroupTimeoutBehaviorFail:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_FAIL
	case models.ConnectionGroupTimeoutBehaviorEmitPartial:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_EMIT_PARTIAL
	default:
		return pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_DROP
	}
}

func protoToTimeoutBehavior(timeoutBehavior pb.ConnectionGroup_Spec_TimeoutBehavior) string {
	switch timeoutBehavior {
	case pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_FAIL:
		return models.ConnectionGroupTimeoutBehaviorFail
	case pb.ConnectionGroup_Spec_TIMEOUT_BEHAVIOR_EMIT_PARTIAL:
		return models.ConnectionGroupTimeoutBehaviorEmitPartial
	default:
		return models.ConnectionGroupTimeoutBehaviorDrop
	}
}
