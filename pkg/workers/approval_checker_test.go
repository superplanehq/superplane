package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestApprovalChecker_CheckRequirements_EmptyRequirements(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	approvals := []models.StageEventApproval{}
	requirements := []models.ApprovalRequirement{}

	satisfied, err := checker.CheckRequirements(approvals, requirements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !satisfied {
		t.Fatalf("Expected empty requirements to be satisfied")
	}
}

func TestApprovalChecker_CheckRequirements_UserRequirementByID(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	userID := uuid.New()
	now := time.Now()

	approvals := []models.StageEventApproval{
		{
			StageEventID: uuid.New(),
			ApprovedAt:   &now,
			ApprovedBy:   &userID,
		},
	}

	requirements := []models.ApprovalRequirement{
		{
			Type:  models.ApprovalRequirementTypeUser,
			ID:    userID.String(),
			Count: 1,
		},
	}

	satisfied, err := checker.CheckRequirements(approvals, requirements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !satisfied {
		t.Fatalf("Expected user requirement to be satisfied")
	}
}

func TestApprovalChecker_CheckRequirements_UserRequirementNotMet(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	userID := uuid.New()
	differentUserID := uuid.New()
	now := time.Now()

	approvals := []models.StageEventApproval{
		{
			StageEventID: uuid.New(),
			ApprovedAt:   &now,
			ApprovedBy:   &differentUserID,
		},
	}

	requirements := []models.ApprovalRequirement{
		{
			Type:  models.ApprovalRequirementTypeUser,
			ID:    userID.String(),
			Count: 1,
		},
	}

	satisfied, err := checker.CheckRequirements(approvals, requirements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if satisfied {
		t.Fatalf("Expected user requirement to not be satisfied")
	}
}

func TestApprovalChecker_CheckRequirements_UserRequirementCountNotMet(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	userID := uuid.New()
	anotherUserID := uuid.New()
	now := time.Now()

	approvals := []models.StageEventApproval{
		{
			StageEventID: uuid.New(),
			ApprovedAt:   &now,
			ApprovedBy:   &userID,
		},
	}

	requirements := []models.ApprovalRequirement{
		{
			Type: models.ApprovalRequirementTypeUser,
			ID:   userID.String(),
		},
		{
			Type: models.ApprovalRequirementTypeUser,
			ID:   anotherUserID.String(),
		},
	}

	satisfied, err := checker.CheckRequirements(approvals, requirements)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if satisfied {
		t.Fatalf("Expected user requirement to not be satisfied when count not met")
	}
}

func TestApprovalChecker_CheckRequirements_UnknownRequirementType(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	approvals := []models.StageEventApproval{}
	requirements := []models.ApprovalRequirement{
		{
			Type:  "unknown_type",
			Count: 1,
		},
	}

	satisfied, err := checker.CheckRequirements(approvals, requirements)
	if err == nil {
		t.Fatalf("Expected error for unknown requirement type")
	}

	if satisfied {
		t.Fatalf("Expected unknown requirement type to not be satisfied")
	}
}

func TestApprovalChecker_CheckUserRequirement_NilApprovedBy(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	now := time.Now()
	approvals := []models.StageEventApproval{
		{
			StageEventID: uuid.New(),
			ApprovedAt:   &now,
			ApprovedBy:   nil, // Nil approver should be ignored
		},
	}

	requirement := models.ApprovalRequirement{
		Type:  models.ApprovalRequirementTypeUser,
		ID:    uuid.New().String(),
		Count: 1,
	}

	satisfied, err := checker.checkUserRequirement(approvals, requirement)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if satisfied {
		t.Fatalf("Expected requirement not to be satisfied with nil approver")
	}
}

func TestApprovalChecker_CheckUserRequirement_EmptyName(t *testing.T) {
	checker := &ApprovalChecker{
		CanvasID: uuid.New(),
		Logger:   log.NewEntry(log.New()),
	}

	userID := uuid.New()
	now := time.Now()
	approvals := []models.StageEventApproval{
		{
			StageEventID: uuid.New(),
			ApprovedAt:   &now,
			ApprovedBy:   &userID,
		},
	}

	requirement := models.ApprovalRequirement{
		Type:  models.ApprovalRequirementTypeUser,
		Name:  "", // Empty name should not match
		Count: 1,
	}

	satisfied, err := checker.checkUserRequirement(approvals, requirement)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if satisfied {
		t.Fatalf("Expected requirement not to be satisfied with empty name")
	}
}

func TestApprovalChecker_CheckRoleRequirement_DatabaseIntegration(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	authService, err := authorization.NewAuthService()
	if err != nil {
		t.Fatalf("Failed to create auth service: %v", err)
	}

	canvasID := r.Canvas.ID
	orgID := r.Organization.ID
	userID := r.User

	err = authService.SetupOrganizationRoles(orgID.String())
	if err != nil {
		t.Fatalf("Failed to setup organization roles: %v", err)
	}

	err = authService.SetupCanvasRoles(canvasID.String())
	if err != nil {
		t.Fatalf("Failed to setup canvas roles: %v", err)
	}

	checker := &ApprovalChecker{
		CanvasID: canvasID,
		Logger:   log.NewEntry(log.New()),
	}

	t.Run("user with canvas admin role satisfies admin requirement", func(t *testing.T) {
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 1,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected role requirement to be satisfied")
		}
	})

	t.Run("user with canvas viewer role does not satisfy admin requirement", func(t *testing.T) {
		err := authService.RemoveRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to remove role: %v", err)
		}

		err = authService.AssignRole(userID.String(), authorization.RoleCanvasViewer, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 1,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if satisfied {
			t.Fatalf("Expected role requirement to not be satisfied")
		}
	})

	t.Run("multiple users with role satisfy count requirement", func(t *testing.T) {
		user2ID := uuid.New()
		user2 := &models.User{
			ID:       user2ID,
			Name:     "test2",
			Username: "test2",
		}
		database.Conn().Create(user2)

		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		err = authService.AssignRole(user2ID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user2ID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 2,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected role requirement to be satisfied with count 2")
		}
	})

	t.Run("insufficient count does not satisfy requirement", func(t *testing.T) {
		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 3,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if satisfied {
			t.Fatalf("Expected role requirement to not be satisfied with insufficient count")
		}
	})
}

func TestApprovalChecker_CheckGroupRequirement_DatabaseIntegration(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	authService, err := authorization.NewAuthService()
	if err != nil {
		t.Fatalf("Failed to create auth service: %v", err)
	}

	canvasID := r.Canvas.ID
	orgID := r.Organization.ID
	userID := r.User

	err = authService.SetupOrganizationRoles(orgID.String())
	if err != nil {
		t.Fatalf("Failed to setup organization roles: %v", err)
	}

	checker := &ApprovalChecker{
		CanvasID: canvasID,
		Logger:   log.NewEntry(log.New()),
	}

	t.Run("user in group satisfies group requirement", func(t *testing.T) {
		groupName := "engineering-team"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  groupName,
			Count: 1,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected group requirement to be satisfied")
		}
	})

	t.Run("user not in group does not satisfy group requirement", func(t *testing.T) {
		groupName := "different-team"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  groupName,
			Count: 1,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if satisfied {
			t.Fatalf("Expected group requirement to not be satisfied")
		}
	})

	t.Run("multiple users in group satisfy count requirement", func(t *testing.T) {
		user2ID := uuid.New()
		user2 := &models.User{
			ID:       user2ID,
			Name:     "test2",
			Username: "test2",
		}
		database.Conn().Create(user2)

		groupName := "approval-team"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), user2ID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user2 to group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user2ID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  groupName,
			Count: 2,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected group requirement to be satisfied with count 2")
		}
	})

	t.Run("insufficient count does not satisfy group requirement", func(t *testing.T) {
		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  "approval-team",
			Count: 3,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if satisfied {
			t.Fatalf("Expected group requirement to not be satisfied with insufficient count")
		}
	})

	t.Run("group requirement with ID instead of name", func(t *testing.T) {
		groupName := "id-based-group"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			ID:    groupName,
			Count: 1,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected group requirement to be satisfied using ID")
		}
	})
}

func TestApprovalChecker_CheckRequirements_EdgeCases_DatabaseIntegration(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	authService, err := authorization.NewAuthService()
	if err != nil {
		t.Fatalf("Failed to create auth service: %v", err)
	}

	canvasID := r.Canvas.ID
	orgID := r.Organization.ID
	userID := r.User

	err = authService.SetupOrganizationRoles(orgID.String())
	if err != nil {
		t.Fatalf("Failed to setup organization roles: %v", err)
	}

	err = authService.SetupCanvasRoles(canvasID.String())
	if err != nil {
		t.Fatalf("Failed to setup canvas roles: %v", err)
	}

	checker := &ApprovalChecker{
		CanvasID: canvasID,
		Logger:   log.NewEntry(log.New()),
	}

	t.Run("mixed requirements - role and group", func(t *testing.T) {
		user2ID := uuid.New()
		user2 := &models.User{
			ID:       user2ID,
			Name:     "test2",
			Username: "test2",
		}
		database.Conn().Create(user2)

		groupName := "mixed-team"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		err = authService.AssignRole(user2ID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user2ID,
			},
		}

		requirements := []models.ApprovalRequirement{
			{
				Type:  models.ApprovalRequirementTypeGroup,
				Name:  groupName,
				Count: 1,
			},
			{
				Type:  models.ApprovalRequirementTypeRole,
				Name:  authorization.RoleCanvasAdmin,
				Count: 1,
			},
		}

		satisfied, err := checker.CheckRequirements(approvals, requirements)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected mixed requirements to be satisfied")
		}
	})

	t.Run("role requirement with ID instead of name", func(t *testing.T) {
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			ID:    authorization.RoleCanvasAdmin,
			Count: 1,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected role requirement to be satisfied using ID")
		}
	})

	t.Run("non-existent group is not satisfied", func(t *testing.T) {
		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  "non-existent-group",
			Count: 1,
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error for non-existent group, got %v", err)
		}

		if satisfied {
			t.Fatalf("Expected requirement to not be satisfied for non-existent group")
		}
	})

	t.Run("nil approved by users are ignored", func(t *testing.T) {
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   nil,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 1,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected role requirement to be satisfied ignoring nil approvers")
		}
	})

	t.Run("count requirement defaults to 1 when not specified", func(t *testing.T) {
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type: models.ApprovalRequirementTypeRole,
			Name: authorization.RoleCanvasAdmin,
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !satisfied {
			t.Fatalf("Expected role requirement to be satisfied with default count")
		}
	})

	t.Run("same user NOT counted multiple times for role requirement", func(t *testing.T) {
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID, // Same user approving twice
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 2, // Requiring 2 approvals
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should NOT be satisfied because same user should only count once
		if satisfied {
			t.Fatalf("Expected requirement to NOT be satisfied - same user should only count once")
		}
	})

	t.Run("same user NOT counted multiple times for group requirement", func(t *testing.T) {
		groupName := "duplicate-approval-test"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID, // Same user approving twice
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  groupName,
			Count: 2, // Requiring 2 approvals
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should NOT be satisfied because same user should only count once
		if satisfied {
			t.Fatalf("Expected requirement to NOT be satisfied - same user should only count once")
		}
	})

	t.Run("unique users are counted correctly for role requirement", func(t *testing.T) {
		user2ID := uuid.New()
		user2 := &models.User{
			ID:       user2ID,
			Name:     "test2",
			Username: "test2",
		}
		database.Conn().Create(user2)

		user3ID := uuid.New()
		user3 := &models.User{
			ID:       user3ID,
			Name:     "test3",
			Username: "test3",
		}
		database.Conn().Create(user3)

		// Assign role to all users
		err := authService.AssignRole(userID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		err = authService.AssignRole(user2ID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		err = authService.AssignRole(user3ID.String(), authorization.RoleCanvasAdmin, canvasID.String(), authorization.DomainCanvas)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID, // Same user approving twice - should only count once
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user2ID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user3ID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeRole,
			Name:  authorization.RoleCanvasAdmin,
			Count: 3, // Requiring 3 unique approvals
		}

		satisfied, err := checker.checkRoleRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should be satisfied because we have 3 unique users despite duplicate approval
		if !satisfied {
			t.Fatalf("Expected requirement to be satisfied with 3 unique users")
		}
	})

	t.Run("unique users are counted correctly for group requirement", func(t *testing.T) {
		user2ID := uuid.New()
		user2 := &models.User{
			ID:       user2ID,
			Name:     "test2",
			Username: "test2",
		}
		database.Conn().Create(user2)

		user3ID := uuid.New()
		user3 := &models.User{
			ID:       user3ID,
			Name:     "test3",
			Username: "test3",
		}
		database.Conn().Create(user3)

		groupName := "unique-counting-test"
		err := authService.CreateGroup(orgID.String(), groupName, authorization.RoleOrgViewer)
		if err != nil {
			t.Fatalf("Failed to create group: %v", err)
		}

		// Add all users to group
		err = authService.AddUserToGroup(orgID.String(), userID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user to group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), user2ID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user2 to group: %v", err)
		}

		err = authService.AddUserToGroup(orgID.String(), user3ID.String(), groupName)
		if err != nil {
			t.Fatalf("Failed to add user3 to group: %v", err)
		}

		now := time.Now()
		approvals := []models.StageEventApproval{
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &userID, // Same user approving twice - should only count once
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user2ID,
			},
			{
				StageEventID: uuid.New(),
				ApprovedAt:   &now,
				ApprovedBy:   &user3ID,
			},
		}

		requirement := models.ApprovalRequirement{
			Type:  models.ApprovalRequirementTypeGroup,
			Name:  groupName,
			Count: 3, // Requiring 3 unique approvals
		}

		satisfied, err := checker.checkGroupRequirement(approvals, requirement)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should be satisfied because we have 3 unique users despite duplicate approval
		if !satisfied {
			t.Fatalf("Expected requirement to be satisfied with 3 unique users")
		}
	})
}
