package canvases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateCanvas(
	_ context.Context,
	authService authorization.Authorization,
	organizationID string,
	id string,
	name *string,
	description *string,
	canvasVersioningEnabled *bool,
	changeRequestApprovalConfig *pb.CanvasChangeRequestApprovalConfig,
) (*pb.UpdateCanvasResponse, error) {
	canvasID, err := uuid.Parse(id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if _, templateErr := models.FindCanvasTemplate(canvasID); templateErr == nil {
				return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
			}
		}
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	if canvas.IsTemplate {
		return nil, status.Error(codes.FailedPrecondition, "templates are read-only")
	}

	changed := false
	if name != nil {
		nextName := strings.TrimSpace(*name)
		if nextName == "" {
			return nil, status.Error(codes.InvalidArgument, "canvas name is required")
		}

		if canvas.Name != nextName {
			canvas.Name = nextName
			changed = true
		}
	}

	if description != nil && canvas.Description != *description {
		canvas.Description = *description
		changed = true
	}

	if changeRequestApprovalConfig != nil {
		approvers, parseErr := parseCanvasChangeRequestApprovalConfig(changeRequestApprovalConfig)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", parseErr)
		}

		validateErr := validateCanvasChangeRequestApprovers(
			authService,
			organizationID,
			approvers,
		)
		if validateErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", validateErr)
		}

		if !slices.EqualFunc(canvas.ChangeRequestApprovers, approvers, func(left, right models.CanvasChangeRequestApprover) bool {
			return left.Type == right.Type && left.User == right.User && left.Role == right.Role
		}) {
			canvas.ChangeRequestApprovers = approvers
			changed = true
		}
	}

	if canvasVersioningEnabled != nil && canvas.CanvasVersioningEnabled != *canvasVersioningEnabled {
		canvas.CanvasVersioningEnabled = *canvasVersioningEnabled
		changed = true
	}

	if changed {
		now := time.Now()
		canvas.UpdatedAt = &now

		if saveErr := database.Conn().Save(canvas).Error; saveErr != nil {
			if strings.Contains(saveErr.Error(), ErrDuplicateCanvasName) {
				return nil, status.Errorf(codes.AlreadyExists, "Canvas with the same name already exists")
			}
			log.Errorf("failed to update canvas %s metadata: %v", canvas.ID.String(), saveErr)
			return nil, status.Error(codes.Internal, "failed to update canvas")
		}
	}

	if publishErr := messages.NewCanvasUpdatedMessage(canvas.ID.String()).Publish(true); publishErr != nil {
		log.Errorf("failed to publish canvas updated RabbitMQ message: %v", publishErr)
	}

	serializedCanvas, serializeErr := SerializeCanvas(canvas, false)
	if serializeErr != nil {
		log.Errorf("failed to serialize canvas %s after update: %v", canvas.ID.String(), serializeErr)
		return nil, status.Error(codes.Internal, "failed to serialize canvas")
	}

	return &pb.UpdateCanvasResponse{
		Canvas: serializedCanvas,
	}, nil
}

func validateCanvasChangeRequestApprovers(
	authService authorization.Authorization,
	organizationID string,
	approvers []models.CanvasChangeRequestApprover,
) error {
	if len(approvers) == 0 {
		return fmt.Errorf("at least one approver is required")
	}

	requestedUserIDs := make([]string, 0, len(approvers))
	requestedRoles := make([]string, 0, len(approvers))
	for _, approver := range approvers {
		if err := validateCanvasChangeRequestApprover(approver); err != nil {
			return err
		}
		if approver.Type == models.CanvasChangeRequestApproverTypeUser {
			if _, parseErr := uuid.Parse(approver.User); parseErr != nil {
				return fmt.Errorf("approver user %s is not a valid UUID", approver.User)
			}
			requestedUserIDs = append(requestedUserIDs, approver.User)
		}
		if approver.Type == models.CanvasChangeRequestApproverTypeRole {
			requestedRoles = append(requestedRoles, approver.Role)
		}
	}

	activeUsers, err := models.ListActiveUsersByID(organizationID, requestedUserIDs)
	if err != nil {
		return fmt.Errorf("failed to validate approver users: %w", err)
	}
	activeUserIDs := make(map[string]struct{}, len(activeUsers))
	for _, user := range activeUsers {
		activeUserIDs[user.ID.String()] = struct{}{}
	}
	for _, userID := range requestedUserIDs {
		if _, ok := activeUserIDs[userID]; !ok {
			return fmt.Errorf("approver user %s was not found in this organization", userID)
		}
	}

	for _, roleName := range requestedRoles {
		_, roleErr := authService.GetRoleDefinition(roleName, models.DomainTypeOrganization, organizationID)
		if roleErr != nil {
			return fmt.Errorf("approver role %s was not found in this organization", roleName)
		}
	}

	return nil
}
