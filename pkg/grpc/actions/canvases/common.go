package canvases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const canvasNameAlreadyExistsMessage = "Canvas with the same name already exists"
const canvasNameUniqueConstraint = "workflows_organization_id_name_key"

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

func parseAndValidateCanvasChangeRequestApprovers(
	authService authorization.Authorization,
	organizationID string,
	changeManagement *pb.Canvas_ChangeManagement,
) ([]models.CanvasChangeRequestApprover, error) {
	approvers, err := parseCanvasChangeRequestApprovalConfig(changeManagement)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", err)
	}
	if approvers == nil {
		return nil, nil
	}
	if err := validateCanvasChangeRequestApprovers(authService, organizationID, approvers); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid change request approval config: %v", err)
	}

	return approvers, nil
}

func ensureCanvasNameAvailableInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	name string,
) error {
	existingCanvas, err := models.FindCanvasByNameInTransaction(tx, name, organizationID)
	if err == nil && existingCanvas.ID != canvasID {
		return models.ErrCanvasNameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func mapCanvasNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	if isCanvasNameUniqueConstraintError(err) {
		return status.Error(codes.AlreadyExists, canvasNameAlreadyExistsMessage)
	}

	return err
}

func isCanvasNameUniqueConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == canvasNameUniqueConstraint
}

func publishCanvasVersionInTransaction(
	ctx context.Context,
	tx *gorm.DB,
	liveVersion *models.CanvasVersion,
	nextVersion *models.CanvasVersion,
	options changesets.CanvasPublisherOptions,
) error {
	changeset, err := changesets.NewChangesetBuilder(
		liveVersion.Nodes,
		liveVersion.Edges,
		nextVersion.Nodes,
		nextVersion.Edges,
	).Build()
	if err != nil {
		return err
	}

	if len(changeset.GetChanges()) == 0 {
		return models.PromoteToLiveInTransaction(tx, nextVersion, nextVersion.Nodes, nextVersion.Edges)
	}

	publisher, err := changesets.NewCanvasPublisher(tx, nextVersion, liveVersion, options)
	if err != nil {
		return err
	}

	return publisher.Publish(ctx)
}
