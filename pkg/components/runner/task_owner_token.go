package runner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const TaskOwnerTokenPurpose = "runner_task_owner"
const EnvSuperplaneOwnerToken = "SUPERPLANE_OWNER_TOKEN"

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
	execution, err := models.FindNodeExecutionInTransaction(db, scope.CanvasID, scope.NodeExecutionID)
	if err != nil || execution.RunID != scope.CanvasRunID || execution.NodeID != scope.NodeID {
		return fmt.Errorf("owner execution does not match")
	}
	run, err := models.FindUnscopedCanvasRun(db, scope.CanvasRunID)
	if err != nil || run.WorkflowID != scope.CanvasID {
		return fmt.Errorf("owner run does not match")
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, scope.CanvasID)
	if err != nil || canvas.OrganizationID != scope.OrganizationID || canvas.FactoryID == nil || *canvas.FactoryID != scope.FactoryID {
		return fmt.Errorf("owner canvas does not match")
	}
	if scope.MergeConfidence {
		if execution.State != models.CanvasNodeExecutionStateStarted {
			return fmt.Errorf("merge confidence execution is inactive")
		}
		return nil
	}
	linked, err := ResolveArtifactRunContext(db, scope.CanvasRunID)
	if err != nil || linked.OrganizationID != scope.OrganizationID || linked.FactoryID != scope.FactoryID || linked.WorkOrderID != scope.WorkOrderID || linked.CanvasID != scope.CanvasID {
		return fmt.Errorf("owner task association does not match")
	}
	return nil
}

func HasTaskOwnerToken(environment []TaskEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplaneOwnerToken)
	return ok && value != ""
}

func AttachTaskOwnerEnv(ctx core.ExecutionContext, environment []TaskEnvironmentVariable, timeoutSeconds int) []TaskEnvironmentVariable {
	environment = withoutEnvironmentName(environment, EnvSuperplaneOwnerToken)
	if HasPlanningSessionToken(environment) || ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return environment
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return environment
	}
	orgID, err := uuid.Parse(ctx.OrganizationID)
	if err != nil {
		return environment
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if secret == "" || baseURL == "" {
		return environment
	}
	db := database.DB(context.Background())
	scope := TaskOwnerScope{
		OrganizationID: orgID, CanvasID: canvasID, CanvasRunID: ctx.RunID, NodeExecutionID: ctx.ID, NodeID: ctx.NodeID,
	}
	if token, ok := environmentValue(environment, EnvSuperplaneMergeConfidenceToken); ok {
		merge, err := ParseMergeConfidenceToken(jwt.NewSigner(secret), token)
		if err != nil || merge.OrganizationID != orgID || merge.CanvasRunID != ctx.RunID || merge.NodeExecutionID != ctx.ID {
			return environment
		}
		scope.FactoryID, scope.WorkOrderID, scope.MergeConfidence = merge.FactoryID, merge.WorkOrderID, true
	} else {
		linked, err := ResolveArtifactRunContext(db, ctx.RunID)
		if err != nil {
			return environment
		}
		scope.FactoryID, scope.WorkOrderID = linked.FactoryID, linked.WorkOrderID
	}
	if err := ValidateTaskOwnerScope(db, &scope); err != nil {
		return environment
	}
	factory, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		return environment
	}
	order, err := factory.FindWorkOrder(db, scope.WorkOrderID)
	if err != nil || order.State == models.FactoryWorkOrderStateClosed || order.IsArchived() {
		return environment
	}
	token, err := MintTaskOwnerToken(jwt.NewSigner(secret), scope, time.Duration(timeoutSeconds)*time.Second+artifactTokenGracePeriod)
	if err != nil {
		return environment
	}
	environment = withoutEnvironmentName(environment, EnvSuperplaneBaseURL)
	return append(environment, TaskEnvironmentVariable{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(baseURL, "/")}, TaskEnvironmentVariable{Name: EnvSuperplaneOwnerToken, Value: token})
}
