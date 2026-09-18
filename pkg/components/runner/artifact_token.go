package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	ArtifactUploadTokenPurpose = "runner_artifact_upload"
	EnvSuperplaneArtifactToken = "SUPERPLANE_ARTIFACT_TOKEN"
	EnvPlaywrightMCPBrowser    = "PLAYWRIGHT_MCP_BROWSER"
	artifactTokenGracePeriod   = 30 * time.Minute
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

func HasArtifactUploadToken(environment []BrokerEnvironmentVariable) bool {
	for _, item := range environment {
		if item.Name == EnvSuperplaneArtifactToken && strings.TrimSpace(item.Value) != "" {
			return true
		}
	}
	return false
}

func AttachArtifactUploadEnv(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, timeoutSeconds int, enabled bool) []BrokerEnvironmentVariable {
	if !enabled || HasPlanningSessionToken(environment) || ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return environment
	}
	db := database.DB(context.Background())
	execution, err := models.FindWorkOrderExecutionForRun(db, ctx.RunID)
	if err != nil {
		if !errors.Is(err, models.ErrFactoryWorkOrderExecutionNotFound) && ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip artifact upload token: failed to resolve work order")
		}
		return environment
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
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
		OrganizationID:  execution.OrganizationID,
		FactoryID:       execution.FactoryID,
		WorkOrderID:     execution.WorkOrderID,
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
	return append(environment,
		BrokerEnvironmentVariable{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(baseURL, "/")},
		BrokerEnvironmentVariable{Name: EnvSuperplaneArtifactToken, Value: token},
		BrokerEnvironmentVariable{Name: EnvPlaywrightMCPBrowser, Value: "chromium"},
	)
}
