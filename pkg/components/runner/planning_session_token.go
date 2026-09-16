package runner

import (
	"context"
	"fmt"
	"net"
	"net/url"
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
	PlanningSessionTokenPurpose      = "planning_session"
	EnvSuperplanePlanningID          = "SUPERPLANE_PLANNING_SESSION_ID"
	EnvSuperplanePlanningSessionKind = "SUPERPLANE_PLANNING_SESSION_KIND"
	EnvSuperplaneAnalysisSpecFile    = "SUPERPLANE_ANALYSIS_SPEC_FILE"
	EnvSuperplaneAnalysisScoreFile   = "SUPERPLANE_ANALYSIS_SCORE_FILE"
	EnvSuperplaneBaseURL             = "SUPERPLANE_BASE_URL"
	EnvSuperplaneRunToken            = "SUPERPLANE_RUN_TOKEN"
)

type PlanningSessionScope struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	SessionID      uuid.UUID
	CanvasRunID    uuid.UUID
}

func MintPlanningSessionToken(signer *jwt.Signer, scope PlanningSessionScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds) * time.Second
	}
	if scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.SessionID == uuid.Nil || scope.CanvasRunID == uuid.Nil {
		return "", fmt.Errorf("planning session scope is incomplete")
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":       PlanningSessionTokenPurpose,
		"org_id":        scope.OrganizationID.String(),
		"factory_id":    scope.FactoryID.String(),
		"session_id":    scope.SessionID.String(),
		"canvas_run_id": scope.CanvasRunID.String(),
	})
}

func ParsePlanningSessionToken(signer *jwt.Signer, token string) (*PlanningSessionScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != PlanningSessionTokenPurpose {
		return nil, fmt.Errorf("invalid planning session token purpose")
	}
	scope := PlanningSessionScope{}
	scope.OrganizationID, err = parsePlanningClaimUUID(claims, "org_id")
	if err != nil {
		return nil, err
	}
	scope.FactoryID, err = parsePlanningClaimUUID(claims, "factory_id")
	if err != nil {
		return nil, err
	}
	scope.SessionID, err = parsePlanningClaimUUID(claims, "session_id")
	if err != nil {
		return nil, err
	}
	scope.CanvasRunID, err = parsePlanningClaimUUID(claims, "canvas_run_id")
	if err != nil {
		return nil, err
	}
	return &scope, nil
}

func HasPlanningSessionToken(environment []BrokerEnvironmentVariable) bool {
	for _, item := range environment {
		if item.Name == EnvSuperplanePlanningID && strings.TrimSpace(item.Value) != "" {
			return true
		}
	}
	return false
}

func AttachPlanningSessionEnv(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, timeoutSeconds int) []BrokerEnvironmentVariable {
	if ctx.RunID == uuid.Nil {
		return environment
	}
	session, err := models.FindPlanningSessionByRun(database.DB(context.Background()), ctx.RunID)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip planning session token: session not found for run")
		}
		return environment
	}
	if !session.IsAnalysisSession() {
		return environment
	}

	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip planning session token: public SuperPlane URL is missing")
		}
		return environment
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip planning session token: JWT_SECRET is missing")
		}
		return environment
	}
	if session.CanvasRunID == nil {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip planning session token: canvas run is missing")
		}
		return environment
	}

	ttl := time.Duration(timeoutSeconds) * time.Second
	token, err := MintPlanningSessionToken(jwt.NewSigner(secret), PlanningSessionScope{
		OrganizationID: session.OrganizationID,
		FactoryID:      session.FactoryID,
		SessionID:      session.ID,
		CanvasRunID:    *session.CanvasRunID,
	}, ttl)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("skip planning session token: failed to mint token")
		}
		return environment
	}
	if ctx.Logger != nil {
		ctx.Logger.WithField("planning_session_id", session.ID).Info("attached planning session token")
	}
	environment = append(append(environment, planningSessionEnvVars(baseURL, token)...), BrokerEnvironmentVariable{
		Name:  EnvSuperplanePlanningID,
		Value: session.ID.String(),
	}, BrokerEnvironmentVariable{
		Name:  EnvSuperplanePlanningSessionKind,
		Value: session.Kind,
	}, BrokerEnvironmentVariable{
		Name:  EnvSuperplaneAnalysisSpecFile,
		Value: "/tmp/intent.md",
	}, BrokerEnvironmentVariable{
		Name:  EnvSuperplaneAnalysisScoreFile,
		Value: "/tmp/intake-analysis.json",
	})
	return environment
}

func planningSessionEnvVars(baseURL, token string) []BrokerEnvironmentVariable {
	return []BrokerEnvironmentVariable{
		{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(strings.TrimSpace(baseURL), "/")},
		{Name: EnvSuperplaneRunToken, Value: token},
	}
}

func PublicSuperplaneBaseURL(fallback string) string {
	candidates := []string{
		os.Getenv("WEBHOOKS_BASE_URL"),
		os.Getenv("BASE_URL"),
		fallback,
	}
	for _, candidate := range candidates {
		normalized := strings.TrimRight(strings.TrimSpace(candidate), "/")
		if normalized == "" || isLoopbackBaseURL(normalized) {
			continue
		}
		return normalized
	}
	return ""
}

// RunnerSuperplaneBaseURL is the SuperPlane origin the runner process can
// reach. A local Docker broker talks to the compose app on the host, even
// when BASE_URL is a public tunnel for GitHub.
func RunnerSuperplaneBaseURL(fallback string) string {
	if isLocalTaskBrokerURL(os.Getenv("TASK_BROKER_BASE_URL")) {
		return localComposeSuperplaneBaseURL(fallback)
	}
	return PublicSuperplaneBaseURL(fallback)
}

func localComposeSuperplaneBaseURL(fallback string) string {
	for _, candidate := range []string{os.Getenv("BASE_URL"), fallback} {
		if rewritten := rewriteLoopbackHostForDocker(candidate); rewritten != "" {
			return rewritten
		}
	}
	port := strings.TrimSpace(os.Getenv("PUBLIC_API_PORT"))
	if port == "" {
		port = "8000"
	}
	return "http://host.docker.internal:" + port
}

func rewriteLoopbackHostForDocker(raw string) string {
	normalized := strings.TrimRight(strings.TrimSpace(raw), "/")
	if normalized == "" || !isLoopbackBaseURL(normalized) {
		return ""
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return ""
	}
	port := parsed.Port()
	if port == "" {
		parsed.Host = "host.docker.internal"
	} else {
		parsed.Host = net.JoinHostPort("host.docker.internal", port)
	}
	return strings.TrimRight(parsed.String(), "/")
}

func parsePlanningClaimUUID(claims map[string]interface{}, key string) (uuid.UUID, error) {
	raw, _ := claims[key].(string)
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func isLoopbackBaseURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return true
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
