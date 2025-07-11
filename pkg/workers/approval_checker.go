package workers

import (
	"fmt"

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
	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		if requirement.ID != "" {
			if approval.ApprovedBy.String() == requirement.ID {
				return true, nil
			}
		} else if requirement.Name != "" {
			user, err := models.FindUserByID(approval.ApprovedBy.String())
			if err != nil {
				ac.Logger.Warnf("Error finding user %s: %v", approval.ApprovedBy.String(), err)
				continue
			}

			if user.Username == requirement.Name {
				return true, nil
			}
		}
	}

	return false, nil
}

func (ac *ApprovalChecker) checkRoleRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	authService, err := authorization.NewAuthService()
	if err != nil {
		return false, fmt.Errorf("error creating auth service: %v", err)
	}

	roleName := requirement.Name
	if requirement.ID != "" {
		roleName = requirement.ID
	}

	countedUsers := make(map[string]bool)

	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		userID := approval.ApprovedBy.String()

		if countedUsers[userID] {
			continue
		}

		userRoles, err := authService.GetUserRolesForCanvas(userID, ac.CanvasID.String())
		if err != nil {
			ac.Logger.Warnf("Error getting roles for user %s: %v", userID, err)
			continue
		}

		hasRole := false
		for _, role := range userRoles {
			if role.Name == roleName {
				hasRole = true
				break
			}
		}

		if hasRole {
			countedUsers[userID] = true
		}
	}

	return len(countedUsers) >= requirement.Count, nil
}

func (ac *ApprovalChecker) checkGroupRequirement(approvals []models.StageEventApproval, requirement models.ApprovalRequirement) (bool, error) {
	authService, err := authorization.NewAuthService()
	if err != nil {
		return false, fmt.Errorf("error creating auth service: %v", err)
	}

	groupName := requirement.Name
	if requirement.ID != "" {
		groupName = requirement.ID
	}

	groupUsers, err := authService.GetGroupUsers(ac.CanvasID.String(), authorization.DomainCanvas, groupName)
	if err != nil {
		return false, fmt.Errorf("error getting group users: %v", err)
	}

	groupUserMap := make(map[string]bool)
	for _, userID := range groupUsers {
		groupUserMap[userID] = true
	}

	countedUsers := make(map[string]bool)

	for _, approval := range approvals {
		if approval.ApprovedBy == nil {
			continue
		}

		userID := approval.ApprovedBy.String()

		if countedUsers[userID] {
			continue
		}

		if groupUserMap[userID] {
			countedUsers[userID] = true
		}
	}

	return len(countedUsers) >= requirement.Count, nil
}
