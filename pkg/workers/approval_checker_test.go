package workers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
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
			Count: 2, // Requires 2 approvals but only has 1
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
