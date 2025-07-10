package workers

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/models"
)

type ApprovalChecker struct {
	CanvasID uuid.UUID
	Logger   *log.Entry
}

func (ac *ApprovalChecker) CheckRequirements(approvals []models.StageEventApproval, requirements []models.ApprovalRequirement) (bool, error) {
	// Check each requirement individually
	for _, req := range requirements {
		satisfied, err := ac.checkRequirement(approvals, req)
		if err != nil {
			return false, fmt.Errorf("error checking requirement %v: %v", req, err)
		}

		if !satisfied {
			ac.Logger.Infof("Requirement not satisfied: %s %s (count: %d)", req.Type, req.Name, req.Count)
			return false, nil
		}
	}

	return true, nil
}

func (ac *ApprovalChecker) checkRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	switch requirement.Type {
	case models.ApprovalRequirementTypeUser:
		return ac.checkUserRequirement(approvals, requirement)
	case models.ApprovalRequirementTypeRole:
		return ac.checkRoleRequirement(approvals, requirement)
	case models.ApprovalRequirementTypeGroup:
		return ac.checkGroupRequirement(approvals, requirement)
	default:
		return false, fmt.Errorf("unknown requirement type: %s", requirement.Type)
	}
}

func (ac *ApprovalChecker) checkUserRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	matchingApprovals := 0

	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		if requirement.ID != "" {
			if approval.ApprovedBy.String() == requirement.ID {
				matchingApprovals++
			}
		} else if requirement.Name != "" {
			user, err := models.FindUserByID(approval.ApprovedBy.String())
			if err != nil {
				ac.Logger.Warnf("Error finding user %s: %v", approval.ApprovedBy.String(), err)
				continue
			}

			if user.Username == requirement.Name {
				matchingApprovals++
			}
		}
	}

	return matchingApprovals >= requirement.Count, nil
}

func (ac *ApprovalChecker) checkRoleRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	authService, err := authorization.NewAuthService()
	if err != nil {
		return false, fmt.Errorf("error creating auth service: %v", err)
	}

	matchingApprovals := 0
	roleName := requirement.Name
	if requirement.ID != "" {
		roleName = requirement.ID
	}

	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		// Get user roles for this canvas
		userRoles, err := authService.GetUserRolesForCanvas(approval.ApprovedBy.String(), ac.CanvasID.String())
		if err != nil {
			ac.Logger.Warnf("Error getting roles for user %s: %v", approval.ApprovedBy.String(), err)
			continue
		}

		// Check if user has the required role
		hasRole := false
		for _, role := range userRoles {
			if role.Name == roleName {
				hasRole = true
				break
			}
		}

		if hasRole {
			matchingApprovals++
		}
	}

	return matchingApprovals >= requirement.Count, nil
}

func (ac *ApprovalChecker) checkGroupRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	authService, err := authorization.NewAuthService()
	if err != nil {
		return false, fmt.Errorf("error creating auth service: %v", err)
	}

	// Get canvas to find organization ID
	canvas, err := models.FindCanvasByID(ac.CanvasID.String())
	if err != nil {
		return false, fmt.Errorf("error finding canvas: %v", err)
	}

	matchingApprovals := 0
	groupName := requirement.Name
	if requirement.ID != "" {
		groupName = requirement.ID
	}

	// Get all users in the group
	groupUsers, err := authService.GetGroupUsers(canvas.OrganizationID.String(), groupName)
	if err != nil {
		return false, fmt.Errorf("error getting group users: %v", err)
	}

	// Convert group users to a map for faster lookup
	groupUserMap := make(map[string]bool)
	for _, userID := range groupUsers {
		// Remove "user:" prefix if present
		cleanUserID := strings.TrimPrefix(userID, "user:")
		groupUserMap[cleanUserID] = true
	}

	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		if groupUserMap[approval.ApprovedBy.String()] {
			matchingApprovals++
		}
	}

	return matchingApprovals >= requirement.Count, nil
}
