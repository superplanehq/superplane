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

// AttachRunnerEnv mints the token of the global runner MCP server. Call it
// after the planning, artifact, and merge confidence attach calls: the flow
// comes from the tokens that those calls added. A run without an open task
// gets no token, so the global server does not start.
func AttachRunnerEnv(ctx core.ExecutionContext, environment []TaskEnvironmentVariable, timeoutSeconds int) []TaskEnvironmentVariable {
	environment = withoutEnvironmentName(environment, EnvSuperplaneRunnerToken)
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if secret == "" || baseURL == "" {
		return environment
	}

	signer := jwt.NewSigner(secret)
	scope, err := resolveRunnerScope(database.DB(context.Background()), signer, ctx, environment)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Debug("skip runner token")
		}
		return environment
	}
	token, err := MintRunnerToken(signer, *scope, time.Duration(timeoutSeconds)*time.Second+artifactTokenGracePeriod)
	if err != nil {
		return environment
	}
	environment = withoutEnvironmentName(environment, EnvSuperplaneBaseURL)
	return append(environment,
		TaskEnvironmentVariable{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(baseURL, "/")},
		TaskEnvironmentVariable{Name: EnvSuperplaneRunnerToken, Value: token},
	)
}

func resolveRunnerScope(db *gorm.DB, signer *jwt.Signer, ctx core.ExecutionContext, environment []TaskEnvironmentVariable) (*RunnerScope, error) {
	if ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return nil, fmt.Errorf("run is missing")
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas: %w", err)
	}
	orgID, err := uuid.Parse(ctx.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization: %w", err)
	}
	scope := &RunnerScope{
		OrganizationID:  orgID,
		CanvasID:        canvasID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		NodeID:          ctx.NodeID,
	}

	switch {
	case HasPlanningSessionToken(environment):
		err = resolvePlanningRunnerScope(db, scope)
	case HasMergeConfidenceToken(environment):
		err = resolveMergeConfidenceRunnerScope(signer, scope, environment)
	default:
		err = resolveTaskRunnerScope(db, scope)
	}
	if err != nil {
		return nil, err
	}
	scope.Scopes = ScopesForFlow(scope.Flow)
	if err := ValidateRunnerScope(db, scope); err != nil {
		return nil, err
	}
	if scope.Flow == RunnerFlowPlanning {
		return scope, nil
	}
	if err := requireOpenTask(db, scope); err != nil {
		return nil, err
	}
	return scope, nil
}

func resolvePlanningRunnerScope(db *gorm.DB, scope *RunnerScope) error {
	session, err := models.FindPlanningSessionByRun(db, scope.CanvasRunID)
	if err != nil {
		return err
	}
	if session.OrganizationID != scope.OrganizationID || session.State == models.PlanningSessionStateEnded {
		return fmt.Errorf("planning session is not active")
	}
	scope.Flow = RunnerFlowPlanning
	scope.FactoryID = session.FactoryID
	scope.PlanningSessionID = session.ID
	return nil
}

func resolveMergeConfidenceRunnerScope(signer *jwt.Signer, scope *RunnerScope, environment []TaskEnvironmentVariable) error {
	token, _ := environmentValue(environment, EnvSuperplaneMergeConfidenceToken)
	merge, err := ParseMergeConfidenceToken(signer, token)
	if err != nil {
		return err
	}
	if merge.OrganizationID != scope.OrganizationID || merge.CanvasRunID != scope.CanvasRunID || merge.NodeExecutionID != scope.NodeExecutionID {
		return fmt.Errorf("merge confidence token does not match the run")
	}
	scope.Flow = RunnerFlowMergeConfidence
	scope.FactoryID = merge.FactoryID
	scope.WorkOrderID = merge.WorkOrderID
	return nil
}

func resolveTaskRunnerScope(db *gorm.DB, scope *RunnerScope) error {
	linked, err := ResolveArtifactRunContext(db, scope.CanvasRunID)
	if err != nil {
		return err
	}
	scope.Flow = RunnerFlowTask
	scope.FactoryID = linked.FactoryID
	scope.WorkOrderID = linked.WorkOrderID
	return nil
}

func requireOpenTask(db *gorm.DB, scope *RunnerScope) error {
	factory, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		return err
	}
	order, err := factory.FindWorkOrder(db, scope.WorkOrderID)
	if err != nil {
		return err
	}
	if order.State == models.FactoryWorkOrderStateClosed || order.IsArchived() {
		return fmt.Errorf("task is closed or archived")
	}
	return nil
}
