package canvases

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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

func ensureCanvasNameAvailableInTransaction(
	tx *gorm.DB,
	organizationID uuid.UUID,
	canvasID uuid.UUID,
	name string,
) error {
	if err := tx.
		Model(&models.Organization{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", organizationID).
		First(&models.Organization{}).
		Error; err != nil {
		return err
	}

	existingCanvas, err := models.FindCanvasByNameInTransaction(tx, name, organizationID)
	if err == nil && existingCanvas.ID != canvasID {
		return models.ErrCanvasNameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}
