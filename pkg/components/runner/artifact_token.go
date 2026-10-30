package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

const (
	ArtifactUploadTokenPurpose  = "runner_artifact_upload"
	EnvSuperplaneArtifactToken  = "SUPERPLANE_ARTIFACT_TOKEN"
	EnvSuperplaneVisualEvidence = "SUPERPLANE_VISUAL_EVIDENCE"
	EnvSuperplanePublishLinks   = "SUPERPLANE_PUBLISH_LINKS"
	EnvPlaywrightMCPBrowser     = "PLAYWRIGHT_MCP_BROWSER"
	artifactTokenGracePeriod    = 30 * time.Minute
)

type ArtifactUploadScope struct {
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	WorkOrderID     uuid.UUID
	CanvasID        uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	NodeID          string
}

var ErrArtifactRunScopeNotFound = errors.New("artifact run scope not found")

type ArtifactRunContext struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	CanvasID       uuid.UUID
	LineExecution  *models.FactoryWorkOrderExecution
	PublishLinks   bool
}

// ResolveArtifactRunContext accepts factory line runs, registered PR
// discussion runs, Verify visual evidence runs, and custom automation runs.
// Other factory canvases do not receive artifact access. Custom automation
// runs set PublishLinks so a Bash or prompt step can attach a link.
func ResolveArtifactRunContext(tx *gorm.DB, runID uuid.UUID) (*ArtifactRunContext, error) {
	execution, err := models.FindWorkOrderExecutionForRun(tx, runID)
	if err == nil {
		run, runErr := models.FindUnscopedCanvasRun(tx, runID)
		if runErr != nil {
			return nil, runErr
		}
		return &ArtifactRunContext{
			OrganizationID: execution.OrganizationID,
			FactoryID:      execution.FactoryID,
			WorkOrderID:    execution.WorkOrderID,
			CanvasID:       run.WorkflowID,
			LineExecution:  execution,
		}, nil
	}
	if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) {
		return nil, err
	}

	run, err := models.FindUnscopedCanvasRun(tx, runID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArtifactRunScopeNotFound
		}
		return nil, err
	}
	handler, err := models.FindPRFeedbackHandlerByCanvasID(tx, run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if handler == nil {
		return resolveFactoryColumnArtifactRun(tx, run)
	}
	if handler.Source != models.FactoryPRFeedbackHandlerSourcePullRequestDiscussion {
		return nil, ErrArtifactRunScopeNotFound
	}
	activity, err := models.FindPullRequestActivityByRunID(tx, runID)
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestActivityNotFound) {
			return nil, ErrArtifactRunScopeNotFound
		}
		return nil, err
	}
	if activity.FeedbackHandlerID == nil || *activity.FeedbackHandlerID != handler.ID {
		return nil, ErrArtifactRunScopeNotFound
	}
	factoryModel, err := models.FindFactory(tx, handler.OrganizationID, handler.FactoryID)
	if err != nil {
		return nil, err
	}
	pullRequest, err := factoryModel.FindPullRequest(tx, models.FactoryPullRequestLookup{ID: activity.PullRequestID})
	if err != nil {
		return nil, err
	}

	return &ArtifactRunContext{
		OrganizationID: handler.OrganizationID,
		FactoryID:      handler.FactoryID,
		WorkOrderID:    pullRequest.WorkOrderID,
		CanvasID:       run.WorkflowID,
	}, nil
}

func resolveFactoryColumnArtifactRun(tx *gorm.DB, run *models.CanvasRun) (*ArtifactRunContext, error) {
	runContext, err := resolveVisualEvidenceArtifactRun(tx, run)
	if err == nil || !errors.Is(err, ErrArtifactRunScopeNotFound) {
		return runContext, err
	}
	return resolveCustomAutomationArtifactRun(tx, run)
}

func resolveCustomAutomationArtifactRun(tx *gorm.DB, run *models.CanvasRun) (*ArtifactRunContext, error) {
	matches, err := models.CanvasHasFactoryAppTemplate(tx, run.WorkflowID, models.FactoryAppTemplateCustomAutomationID)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, ErrArtifactRunScopeNotFound
	}

	canvas, err := models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil {
		return nil, ErrArtifactRunScopeNotFound
	}

	rootEvent, err := models.FindRootEventForRun(tx, run.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArtifactRunScopeNotFound
		}
		return nil, err
	}

	factoryModel, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	workOrderID, err := customAutomationWorkOrderID(tx, factoryModel, rootEvent)
	if err != nil {
		return nil, err
	}

	return &ArtifactRunContext{
		OrganizationID: canvas.OrganizationID,
		FactoryID:      *canvas.FactoryID,
		WorkOrderID:    workOrderID,
		CanvasID:       run.WorkflowID,
		PublishLinks:   true,
	}, nil
}

func customAutomationWorkOrderID(tx *gorm.DB, factoryModel *models.Factory, event *models.CanvasEvent) (uuid.UUID, error) {
	if event == nil {
		return uuid.Nil, ErrArtifactRunScopeNotFound
	}
	payload, ok := objectMap(models.RootEventSourcePayload(event.Data.Data()))
	if !ok {
		return uuid.Nil, ErrArtifactRunScopeNotFound
	}
	if workOrder, ok := objectMap(payload["workOrder"]); ok {
		if id, err := uuid.Parse(strings.TrimSpace(fmt.Sprint(workOrder["id"]))); err == nil && id != uuid.Nil {
			order, findErr := factoryModel.FindWorkOrder(tx, id)
			if findErr != nil {
				if errors.Is(findErr, models.ErrFactoryWorkOrderNotFound) {
					return uuid.Nil, ErrArtifactRunScopeNotFound
				}
				return uuid.Nil, findErr
			}
			return order.ID, nil
		}
	}

	lookup, ok := customAutomationPullRequestLookup(payload)
	if !ok {
		return uuid.Nil, ErrArtifactRunScopeNotFound
	}
	pullRequest, err := factoryModel.FindPullRequest(tx, lookup)
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestNotFound) || errors.Is(err, models.ErrFactoryPullRequestLookupIncomplete) {
			return uuid.Nil, ErrArtifactRunScopeNotFound
		}
		return uuid.Nil, err
	}
	return pullRequest.WorkOrderID, nil
}

func customAutomationPullRequestLookup(payload map[string]any) (models.FactoryPullRequestLookup, bool) {
	repository, _ := objectMap(payload["repository"])
	fullName, _ := repository["full_name"].(string)
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return models.FactoryPullRequestLookup{}, false
	}
	if pullRequest, ok := objectMap(payload["pull_request"]); ok {
		number, ok := positiveInt64(pullRequest["number"])
		if ok {
			return models.FactoryPullRequestLookup{
				Provider:   models.FactoryPullRequestProviderGitHub,
				Repository: fullName,
				Number:     number,
			}, true
		}
	}
	if pullRequest, ok := objectMap(payload["pullrequest"]); ok {
		number, ok := positiveInt64(pullRequest["id"])
		if ok {
			return models.FactoryPullRequestLookup{
				Provider:   models.FactoryPullRequestProviderBitbucket,
				Repository: fullName,
				Number:     number,
			}, true
		}
	}
	return models.FactoryPullRequestLookup{}, false
}

func resolveVisualEvidenceArtifactRun(tx *gorm.DB, run *models.CanvasRun) (*ArtifactRunContext, error) {
	matches, err := models.CanvasHasFactoryAppTemplate(tx, run.WorkflowID, models.FactoryAppTemplateVisualEvidenceID)
	if err != nil {
		return nil, err
	}
	if !matches {
		return nil, ErrArtifactRunScopeNotFound
	}

	canvas, err := models.FindUnscopedCanvasInTransaction(tx, run.WorkflowID)
	if err != nil {
		return nil, err
	}
	if canvas.FactoryID == nil {
		return nil, ErrArtifactRunScopeNotFound
	}

	rootEvent, err := models.FindRootEventForRun(tx, run.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArtifactRunScopeNotFound
		}
		return nil, err
	}
	repository, number, ok := visualEvidencePullRequestIdentity(rootEvent)
	if !ok {
		return nil, ErrArtifactRunScopeNotFound
	}

	factoryModel, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}
	pullRequest, err := factoryModel.FindPullRequest(tx, models.FactoryPullRequestLookup{
		Provider:   models.FactoryPullRequestProviderGitHub,
		Repository: repository,
		Number:     number,
	})
	if err != nil {
		if errors.Is(err, models.ErrFactoryPullRequestNotFound) || errors.Is(err, models.ErrFactoryPullRequestLookupIncomplete) {
			return nil, ErrArtifactRunScopeNotFound
		}
		return nil, err
	}

	return &ArtifactRunContext{
		OrganizationID: canvas.OrganizationID,
		FactoryID:      *canvas.FactoryID,
		WorkOrderID:    pullRequest.WorkOrderID,
		CanvasID:       run.WorkflowID,
	}, nil
}

func visualEvidencePullRequestIdentity(event *models.CanvasEvent) (string, int64, bool) {
	if event == nil {
		return "", 0, false
	}
	payload, ok := objectMap(models.RootEventSourcePayload(event.Data.Data()))
	if !ok {
		return "", 0, false
	}
	repository, _ := objectMap(payload["repository"])
	fullName, _ := repository["full_name"].(string)
	pullRequest, _ := objectMap(payload["pull_request"])
	number, ok := positiveInt64(pullRequest["number"])
	if strings.TrimSpace(fullName) == "" || !ok {
		return "", 0, false
	}
	return strings.TrimSpace(fullName), number, true
}

func objectMap(value any) (map[string]any, bool) {
	mapped, ok := value.(map[string]any)
	return mapped, ok
}

func positiveInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Int64()
		return number, err == nil && number > 0
	case int64:
		return typed, typed > 0
	case int:
		return int64(typed), typed > 0
	case float64:
		return int64(typed), typed > 0 && typed == float64(int64(typed))
	case string:
		number, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return number, err == nil && number > 0
	default:
		return 0, false
	}
}

func MintArtifactUploadToken(signer *jwt.Signer, scope ArtifactUploadScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.WorkOrderID == uuid.Nil ||
		scope.CanvasID == uuid.Nil || scope.CanvasRunID == uuid.Nil || scope.NodeExecutionID == uuid.Nil ||
		strings.TrimSpace(scope.NodeID) == "" {
		return "", fmt.Errorf("artifact upload scope is incomplete")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds)*time.Second + artifactTokenGracePeriod
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":           ArtifactUploadTokenPurpose,
		"org_id":            scope.OrganizationID.String(),
		"factory_id":        scope.FactoryID.String(),
		"work_order_id":     scope.WorkOrderID.String(),
		"canvas_id":         scope.CanvasID.String(),
		"canvas_run_id":     scope.CanvasRunID.String(),
		"node_execution_id": scope.NodeExecutionID.String(),
		"node_id":           scope.NodeID,
	})
}

func ParseArtifactUploadToken(signer *jwt.Signer, token string) (*ArtifactUploadScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != ArtifactUploadTokenPurpose {
		return nil, fmt.Errorf("invalid artifact upload token purpose")
	}
	scope := &ArtifactUploadScope{}
	if scope.OrganizationID, err = parsePlanningClaimUUID(claims, "org_id"); err != nil {
		return nil, err
	}
	if scope.FactoryID, err = parsePlanningClaimUUID(claims, "factory_id"); err != nil {
		return nil, err
	}
	if scope.WorkOrderID, err = parsePlanningClaimUUID(claims, "work_order_id"); err != nil {
		return nil, err
	}
	if scope.CanvasID, err = parsePlanningClaimUUID(claims, "canvas_id"); err != nil {
		return nil, err
	}
	if scope.CanvasRunID, err = parsePlanningClaimUUID(claims, "canvas_run_id"); err != nil {
		return nil, err
	}
	if scope.NodeExecutionID, err = parsePlanningClaimUUID(claims, "node_execution_id"); err != nil {
		return nil, err
	}
	scope.NodeID, _ = claims["node_id"].(string)
	if strings.TrimSpace(scope.NodeID) == "" {
		return nil, fmt.Errorf("invalid node_id")
	}
	return scope, nil
}

func HasArtifactUploadToken(environment []TaskEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplaneArtifactToken)
	return ok && value != ""
}

func HasVisualEvidence(environment []TaskEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplaneVisualEvidence)
	return ok && value != ""
}

func HasPublishLinks(environment []TaskEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplanePublishLinks)
	return ok && value != ""
}

func AttachArtifactUploadEnv(ctx core.ExecutionContext, environment []TaskEnvironmentVariable, timeoutSeconds int, enabled bool) []TaskEnvironmentVariable {
	environment = removeArtifactCapabilityEnv(environment)
	if HasPlanningSessionToken(environment) || ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return environment
	}
	// A run id without a canvas is not a custom automation. Skip the lookup so
	// callers that only disable visual evidence stay free of a database call.
	if !enabled && strings.TrimSpace(ctx.WorkflowID) == "" {
		return environment
	}
	db := database.DB(context.Background())
	runContext, err := ResolveArtifactRunContext(db, ctx.RunID)
	if err != nil {
		if !errors.Is(err, ErrArtifactRunScopeNotFound) && ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip artifact upload token: failed to resolve work order")
		}
		return environment
	}
	if !enabled && !runContext.PublishLinks {
		return environment
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return environment
	}
	if canvasID != runContext.CanvasID {
		return environment
	}
	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if baseURL == "" || secret == "" {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip artifact upload token: SuperPlane URL or JWT secret is missing")
		}
		return environment
	}
	ttl := time.Duration(0)
	if timeoutSeconds > 0 {
		ttl = time.Duration(timeoutSeconds)*time.Second + artifactTokenGracePeriod
	}
	token, err := MintArtifactUploadToken(jwt.NewSigner(secret), ArtifactUploadScope{
		OrganizationID:  runContext.OrganizationID,
		FactoryID:       runContext.FactoryID,
		WorkOrderID:     runContext.WorkOrderID,
		CanvasID:        canvasID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		NodeID:          ctx.NodeID,
	}, ttl)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip artifact upload token: failed to mint token")
		}
		return environment
	}
	environment = append(environment,
		TaskEnvironmentVariable{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(baseURL, "/")},
		TaskEnvironmentVariable{Name: EnvSuperplaneArtifactToken, Value: token},
	)
	if enabled {
		environment = append(environment,
			TaskEnvironmentVariable{Name: EnvSuperplaneVisualEvidence, Value: "yes"},
			TaskEnvironmentVariable{Name: EnvPlaywrightMCPBrowser, Value: "chromium"},
		)
	}
	if runContext.PublishLinks {
		environment = append(environment, TaskEnvironmentVariable{Name: EnvSuperplanePublishLinks, Value: "yes"})
	}
	return environment
}

func removeArtifactCapabilityEnv(environment []TaskEnvironmentVariable) []TaskEnvironmentVariable {
	environment = withoutEnvironmentName(environment, EnvSuperplaneArtifactToken)
	environment = withoutEnvironmentName(environment, EnvSuperplaneVisualEvidence)
	return withoutEnvironmentName(environment, EnvSuperplanePublishLinks)
}
