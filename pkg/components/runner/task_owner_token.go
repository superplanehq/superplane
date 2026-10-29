package runner

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
	"gorm.io/gorm"
)

// TaskOwnerTokenPurpose is the legacy owner token. New runs use the runner
// token. The backend still accepts this token so that runs started before the
// global runner MCP server can finish.
const TaskOwnerTokenPurpose = "runner_task_owner"

// TaskOwnerScope binds ownership to one validated task and agent execution.
type TaskOwnerScope struct {
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	WorkOrderID     uuid.UUID
	CanvasID        uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	NodeID          string
	MergeConfidence bool
}

func MintTaskOwnerToken(signer *jwt.Signer, scope TaskOwnerScope, ttl time.Duration) (string, error) {
	if signer == nil || scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.WorkOrderID == uuid.Nil ||
		scope.CanvasID == uuid.Nil || scope.CanvasRunID == uuid.Nil || scope.NodeExecutionID == uuid.Nil || strings.TrimSpace(scope.NodeID) == "" {
		return "", fmt.Errorf("task owner scope is incomplete")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds)*time.Second + artifactTokenGracePeriod
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose": TaskOwnerTokenPurpose, "org_id": scope.OrganizationID.String(),
		"factory_id": scope.FactoryID.String(), "work_order_id": scope.WorkOrderID.String(),
		"canvas_id": scope.CanvasID.String(), "canvas_run_id": scope.CanvasRunID.String(),
		"node_execution_id": scope.NodeExecutionID.String(), "node_id": scope.NodeID,
		"merge_confidence": fmt.Sprint(scope.MergeConfidence),
	})
}

func ParseTaskOwnerToken(signer *jwt.Signer, token string) (*TaskOwnerScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	if claims["purpose"] != TaskOwnerTokenPurpose {
		return nil, fmt.Errorf("invalid task owner token purpose")
	}
	scope := &TaskOwnerScope{}
	for name, destination := range map[string]*uuid.UUID{
		"org_id": &scope.OrganizationID, "factory_id": &scope.FactoryID, "work_order_id": &scope.WorkOrderID,
		"canvas_id": &scope.CanvasID, "canvas_run_id": &scope.CanvasRunID, "node_execution_id": &scope.NodeExecutionID,
	} {
		*destination, err = parsePlanningClaimUUID(claims, name)
		if err != nil {
			return nil, err
		}
	}
	scope.NodeID, _ = claims["node_id"].(string)
	if strings.TrimSpace(scope.NodeID) == "" {
		return nil, fmt.Errorf("invalid node_id")
	}
	scope.MergeConfidence = claims["merge_confidence"] == "true"
	return scope, nil
}

// ValidateTaskOwnerScope rechecks the signed run associations on every request.
func ValidateTaskOwnerScope(db *gorm.DB, scope *TaskOwnerScope) error {
	flow := RunnerFlowTask
	if scope.MergeConfidence {
		flow = RunnerFlowMergeConfidence
	}
	return ValidateRunnerScope(db, &RunnerScope{
		OrganizationID:  scope.OrganizationID,
		FactoryID:       scope.FactoryID,
		WorkOrderID:     scope.WorkOrderID,
		CanvasID:        scope.CanvasID,
		CanvasRunID:     scope.CanvasRunID,
		NodeExecutionID: scope.NodeExecutionID,
		NodeID:          scope.NodeID,
		Flow:            flow,
	})
}
