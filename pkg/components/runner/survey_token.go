package runner

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	WorkOrderSurveyTokenPurpose = "work_order_survey"
	EnvSuperplaneBaseURL        = "SUPERPLANE_BASE_URL"
	EnvSuperplaneRunToken       = "SUPERPLANE_RUN_TOKEN"
)

type WorkOrderSurveyScope struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	WorkOrderID    uuid.UUID
	CanvasRunID    uuid.UUID
	ExecutionID    uuid.UUID
}

func MintWorkOrderSurveyToken(signer *jwt.Signer, scope WorkOrderSurveyScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds) * time.Second
	}
	if scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.WorkOrderID == uuid.Nil || scope.CanvasRunID == uuid.Nil {
		return "", fmt.Errorf("work order survey scope is incomplete")
	}

	claims := map[string]string{
		"purpose":       WorkOrderSurveyTokenPurpose,
		"org_id":        scope.OrganizationID.String(),
		"factory_id":    scope.FactoryID.String(),
		"work_order_id": scope.WorkOrderID.String(),
		"canvas_run_id": scope.CanvasRunID.String(),
	}
	if scope.ExecutionID != uuid.Nil {
		claims["execution_id"] = scope.ExecutionID.String()
	}
	return signer.GenerateWithClaims(ttl, claims)
}

func ParseWorkOrderSurveyToken(signer *jwt.Signer, token string) (*WorkOrderSurveyScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != WorkOrderSurveyTokenPurpose {
		return nil, fmt.Errorf("invalid work order survey token purpose")
	}

	scope := WorkOrderSurveyScope{}
	scope.OrganizationID, err = parseSurveyClaimUUID(claims, "org_id")
	if err != nil {
		return nil, err
	}
	scope.FactoryID, err = parseSurveyClaimUUID(claims, "factory_id")
	if err != nil {
		return nil, err
	}
	scope.WorkOrderID, err = parseSurveyClaimUUID(claims, "work_order_id")
	if err != nil {
		return nil, err
	}
	scope.CanvasRunID, err = parseSurveyClaimUUID(claims, "canvas_run_id")
	if err != nil {
		return nil, err
	}
	if raw, ok := claims["execution_id"].(string); ok && strings.TrimSpace(raw) != "" {
		scope.ExecutionID, err = uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid execution_id")
		}
	}
	return &scope, nil
}

func WorkOrderSurveyEnvVars(baseURL, token string) []BrokerEnvironmentVariable {
	return []BrokerEnvironmentVariable{
		{Name: EnvSuperplaneBaseURL, Value: strings.TrimRight(strings.TrimSpace(baseURL), "/")},
		{Name: EnvSuperplaneRunToken, Value: token},
	}
}

func HasWorkOrderSurveyToken(environment []BrokerEnvironmentVariable) bool {
	for _, item := range environment {
		if item.Name == EnvSuperplaneRunToken && strings.TrimSpace(item.Value) != "" {
			return true
		}
	}
	return false
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

func AttachWorkOrderSurveyEnv(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, timeoutSeconds int) []BrokerEnvironmentVariable {
	scope, ok := resolveWorkOrderSurveyScope(ctx)
	if !ok {
		return environment
	}

	baseURL := PublicSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip work order survey token: public SuperPlane URL is missing")
		}
		return environment
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		if ctx.Logger != nil {
			ctx.Logger.Warn("skip work order survey token: JWT_SECRET is missing")
		}
		return environment
	}

	ttl := time.Duration(timeoutSeconds) * time.Second
	token, err := MintWorkOrderSurveyToken(jwt.NewSigner(secret), *scope, ttl)
	if err != nil {
		if ctx.Logger != nil {
			ctx.Logger.WithError(err).Warn("failed to mint work order survey token")
		}
		return environment
	}
	if ctx.Logger != nil {
		ctx.Logger.WithField("work_order_id", scope.WorkOrderID).Info("attached work order survey token")
	}
	return append(environment, WorkOrderSurveyEnvVars(baseURL, token)...)
}

func resolveWorkOrderSurveyScope(ctx core.ExecutionContext) (*WorkOrderSurveyScope, bool) {
	if ctx.ID == uuid.Nil || ctx.RunID == uuid.Nil || ctx.FactoryID == uuid.Nil || ctx.WorkOrderID == uuid.Nil {
		return nil, false
	}

	organizationID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return nil, false
	}

	return &WorkOrderSurveyScope{
		OrganizationID: organizationID,
		FactoryID:      ctx.FactoryID,
		WorkOrderID:    ctx.WorkOrderID,
		CanvasRunID:    ctx.RunID,
		ExecutionID:    ctx.ID,
	}, true
}

func parseSurveyClaimUUID(claims map[string]interface{}, key string) (uuid.UUID, error) {
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
