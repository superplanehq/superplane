package canvases

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

func parseCanvasChangeRequestApprovalConfig(
	config *pb.CanvasChangeRequestApprovalConfig,
) ([]models.CanvasChangeRequestApprover, error) {
	if config == nil {
		return nil, nil
	}

	approvers := make([]models.CanvasChangeRequestApprover, 0, len(config.Items))
	seenAnyone := false
	seenUsers := map[string]struct{}{}
	seenRoles := map[string]struct{}{}
	for index, item := range config.Items {
		if item == nil {
			return nil, fmt.Errorf("approver %d is required", index+1)
		}

		approverType, err := canvasChangeRequestApproverTypeFromProto(item.Type)
		if err != nil {
			return nil, fmt.Errorf("approver %d: %w", index+1, err)
		}

		approver := models.CanvasChangeRequestApprover{
			Type: approverType,
			User: strings.TrimSpace(item.UserId),
			Role: strings.TrimSpace(item.RoleName),
		}

		if err := validateCanvasChangeRequestApprover(approver); err != nil {
			return nil, fmt.Errorf("approver %d: %w", index+1, err)
		}
		if err := validateCanvasChangeRequestApproverUniqueness(approver, seenAnyone, seenUsers, seenRoles); err != nil {
			return nil, fmt.Errorf("approver %d: %w", index+1, err)
		}

		switch approver.Type {
		case models.CanvasChangeRequestApproverTypeAnyone:
			seenAnyone = true
		case models.CanvasChangeRequestApproverTypeUser:
			seenUsers[approver.User] = struct{}{}
		case models.CanvasChangeRequestApproverTypeRole:
			seenRoles[approver.Role] = struct{}{}
		}

		approvers = append(approvers, approver)
	}

	if len(approvers) == 0 {
		return nil, fmt.Errorf("at least one approver is required")
	}

	return approvers, nil
}

func canvasChangeRequestApproverTypeFromProto(value pb.CanvasChangeRequestApprover_Type) (string, error) {
	switch value {
	case pb.CanvasChangeRequestApprover_TYPE_ANYONE:
		return models.CanvasChangeRequestApproverTypeAnyone, nil
	case pb.CanvasChangeRequestApprover_TYPE_USER:
		return models.CanvasChangeRequestApproverTypeUser, nil
	case pb.CanvasChangeRequestApprover_TYPE_ROLE:
		return models.CanvasChangeRequestApproverTypeRole, nil
	default:
		return "", fmt.Errorf("unsupported approver type %q", value.String())
	}
}

func validateCanvasChangeRequestApprover(approver models.CanvasChangeRequestApprover) error {
	switch approver.Type {
	case models.CanvasChangeRequestApproverTypeAnyone:
		return nil
	case models.CanvasChangeRequestApproverTypeUser:
		if approver.User == "" {
			return fmt.Errorf("user approvers require user_id")
		}
		return nil
	case models.CanvasChangeRequestApproverTypeRole:
		if approver.Role == "" {
			return fmt.Errorf("role approvers require role_name")
		}
		return nil
	default:
		return fmt.Errorf("unsupported approver type %q", approver.Type)
	}
}

func validateCanvasChangeRequestApproverUniqueness(
	approver models.CanvasChangeRequestApprover,
	seenAnyone bool,
	seenUsers map[string]struct{},
	seenRoles map[string]struct{},
) error {
	switch approver.Type {
	case models.CanvasChangeRequestApproverTypeAnyone:
		if seenAnyone {
			return fmt.Errorf("duplicate any-user approver is not allowed")
		}
	case models.CanvasChangeRequestApproverTypeUser:
		if _, exists := seenUsers[approver.User]; exists {
			return fmt.Errorf("duplicate user approver %s is not allowed", approver.User)
		}
	case models.CanvasChangeRequestApproverTypeRole:
		if _, exists := seenRoles[approver.Role]; exists {
			return fmt.Errorf("duplicate role approver %s is not allowed", approver.Role)
		}
	}

	return nil
}
