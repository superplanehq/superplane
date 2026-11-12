package runner

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// AttachRunnerEnv applies trusted owner credentials or resolves ordinary task access.
func AttachRunnerEnv(ctx core.ExecutionContext, environment []TaskEnvironmentVariable, timeoutSeconds int) []TaskEnvironmentVariable {
	for _, name := range ctx.RunnerEnvironmentRemove {
		environment = withoutEnvironmentName(environment, name)
	}
	if ctx.RunnerEnvironment != nil && !HasPlanningSessionToken(environment) {
		names := make([]string, 0, len(ctx.RunnerEnvironment))
		for name := range ctx.RunnerEnvironment {
			names = append(names, name)
		}
		slices.Sort(names)
		environment = withoutEnvironmentName(environment, EnvSuperplaneRunnerToken)
		environment = withoutEnvironmentName(environment, EnvSuperplaneTaskID)
		for _, name := range names {
			environment = withoutEnvironmentName(environment, name)
			environment = append(environment, TaskEnvironmentVariable{Name: name, Value: ctx.RunnerEnvironment[name]})
		}
		return environment
	}
	return AttachScopedRunnerEnv(database.DB(context.Background()), ctx, environment, timeoutSeconds, nil)
}

// AttachScopedRunnerEnv validates a trusted scope before minting a runner credential.
func AttachScopedRunnerEnv(db *gorm.DB, ctx core.ExecutionContext, environment []TaskEnvironmentVariable, timeoutSeconds int, prepared *RunnerScope) []TaskEnvironmentVariable {
	environment = withoutEnvironmentName(environment, EnvSuperplaneRunnerToken)
	environment = withoutEnvironmentName(environment, EnvSuperplaneTaskID)
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if secret == "" || baseURL == "" {
		return environment
	}

	signer := jwt.NewSigner(secret)
	scope, order, err := resolveRunnerScope(db, ctx, environment, prepared)
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
	environment = append(environment,
		TaskEnvironmentVariable{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(baseURL, "/")},
		TaskEnvironmentVariable{Name: EnvSuperplaneRunnerToken, Value: token},
	)
	if order != nil {
		environment = append(environment, TaskEnvironmentVariable{Name: EnvSuperplaneTaskID, Value: strconv.FormatInt(order.Number, 10)})
	}
	return environment
}

// resolveRunnerScope returns the open task of the run. A planning run has no task.
func resolveRunnerScope(
	db *gorm.DB,
	ctx core.ExecutionContext,
	environment []TaskEnvironmentVariable,
	prepared *RunnerScope,
) (*RunnerScope, *models.FactoryWorkOrder, error) {
	if ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return nil, nil, fmt.Errorf("run is missing")
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid canvas: %w", err)
	}
	orgID, err := uuid.Parse(ctx.OrganizationID)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid organization: %w", err)
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
	case prepared != nil:
		if prepared.OrganizationID != scope.OrganizationID || prepared.CanvasID != scope.CanvasID || prepared.CanvasRunID != scope.CanvasRunID || prepared.NodeExecutionID != scope.NodeExecutionID || prepared.NodeID != scope.NodeID {
			return nil, nil, fmt.Errorf("runner scope does not match the execution")
		}
		cloned := *prepared
		scope = &cloned
	default:
		err = resolveTaskRunnerScope(db, scope)
		if err != nil {
			if scheduleErr := resolveScheduleRunnerScope(db, ctx, scope); scheduleErr == nil {
				err = nil
			}
		}
	}
	if err != nil {
		return nil, nil, err
	}
	scope.Scopes = ScopesForFlow(scope.Flow)
	if scope.Flow == RunnerFlowTask {
		scope.Scopes = append(scope.Scopes, TaskCapabilityScopes(environment)...)
		multiRepo, err := models.HasExperimentalFeature(orgID, features.FeatureFactoryMultiRepoTasks)
		if err != nil {
			return nil, nil, err
		}
		if multiRepo {
			scope.Scopes = append(scope.Scopes, MultiRepoTaskScopes...)
		}
	}
	if err := validateRunnerTokenScope(db, scope); err != nil {
		return nil, nil, err
	}
	if scope.Flow == RunnerFlowPlanning || scope.Flow == RunnerFlowSchedule {
		return scope, nil, nil
	}
	order, err := findOpenTask(db, scope)
	if err != nil {
		return nil, nil, err
	}
	return scope, order, nil
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

func resolveScheduleRunnerScope(db *gorm.DB, ctx core.ExecutionContext, scope *RunnerScope) error {
	node, err := models.FindCanvasNode(db, scope.CanvasID, ctx.NodeID)
	if err != nil {
		return err
	}
	template, _ := node.Metadata.Data()[models.FactoryAppTemplateMetadataKey].(map[string]any)
	mode, _ := template["startMode"].(string)
	id, _ := template["id"].(string)
	if id != models.FactoryAppTemplateCustomAutomationID || mode != "scheduled" {
		return fmt.Errorf("not a schedule automation")
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(db, scope.CanvasID)
	if err != nil || canvas.OrganizationID != scope.OrganizationID || canvas.FactoryID == nil {
		return fmt.Errorf("runner canvas does not match")
	}
	scope.Flow = RunnerFlowSchedule
	scope.FactoryID = *canvas.FactoryID
	scope.WorkOrderID = uuid.Nil
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

func findOpenTask(db *gorm.DB, scope *RunnerScope) (*models.FactoryWorkOrder, error) {
	factory, err := models.FindFactory(db, scope.OrganizationID, scope.FactoryID)
	if err != nil {
		return nil, err
	}
	order, err := factory.FindWorkOrder(db, scope.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if order.State == models.FactoryWorkOrderStateClosed || order.IsArchived() {
		return nil, fmt.Errorf("task is closed or archived")
	}
	return order, nil
}
