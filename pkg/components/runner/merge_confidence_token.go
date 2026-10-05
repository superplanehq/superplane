package runner

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	MergeConfidenceTokenPurpose = "merge_confidence"

	// EnvSuperplaneMergeConfidenceOrderID is set on the merge confidence
	// agent node. The runner reads it, then removes it from the task
	// environment. The work order id travels in the signed token instead.
	EnvSuperplaneMergeConfidenceOrderID = "SUPERPLANE_MERGE_CONFIDENCE_ORDER_ID"
	EnvSuperplaneMergeConfidenceToken   = "SUPERPLANE_MERGE_CONFIDENCE_TOKEN"
)

// MergeConfidenceScope is the runner token for one merge confidence agent run.
// EnabledChecks is the list from the agent prompt. The server reports a check
// only when its name is in this list.
type MergeConfidenceScope struct {
	OrganizationID  uuid.UUID
	FactoryID       uuid.UUID
	WorkOrderID     uuid.UUID
	CanvasRunID     uuid.UUID
	NodeExecutionID uuid.UUID
	EnabledChecks   []string
}

var mergeConfidenceChecksLine = regexp.MustCompile(`(?m)^Enabled checks: (none|(?:risk|performance|security|drift|reversibility)(?:, (?:risk|performance|security|drift|reversibility))*)\.$`)

// ParseMergeConfidenceChecks reads the enabled-checks line from agent prompts.
// A missing line or "none" enables nothing.
func ParseMergeConfidenceChecks(prompts []string) []string {
	for _, prompt := range prompts {
		match := mergeConfidenceChecksLine.FindStringSubmatch(prompt)
		if match == nil {
			continue
		}
		if match[1] == "none" {
			return []string{}
		}
		return uniqueMergeConfidenceChecks(strings.Split(match[1], ", "))
	}
	return []string{}
}

func uniqueMergeConfidenceChecks(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	enabled := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		enabled = append(enabled, name)
	}
	return enabled
}

func MintMergeConfidenceToken(signer *jwt.Signer, scope MergeConfidenceScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds) * time.Second
	}
	if scope.OrganizationID == uuid.Nil || scope.FactoryID == uuid.Nil || scope.WorkOrderID == uuid.Nil || scope.CanvasRunID == uuid.Nil || scope.NodeExecutionID == uuid.Nil {
		return "", fmt.Errorf("merge confidence scope is incomplete")
	}
	return signer.GenerateWithClaims(ttl, map[string]string{
		"purpose":           MergeConfidenceTokenPurpose,
		"org_id":            scope.OrganizationID.String(),
		"factory_id":        scope.FactoryID.String(),
		"work_order_id":     scope.WorkOrderID.String(),
		"canvas_run_id":     scope.CanvasRunID.String(),
		"node_execution_id": scope.NodeExecutionID.String(),
		"enabled_checks":    strings.Join(scope.EnabledChecks, ","),
	})
}

func ParseMergeConfidenceToken(signer *jwt.Signer, token string) (*MergeConfidenceScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != MergeConfidenceTokenPurpose {
		return nil, fmt.Errorf("invalid merge confidence token purpose")
	}
	scope := MergeConfidenceScope{}
	scope.OrganizationID, err = parsePlanningClaimUUID(claims, "org_id")
	if err != nil {
		return nil, err
	}
	scope.FactoryID, err = parsePlanningClaimUUID(claims, "factory_id")
	if err != nil {
		return nil, err
	}
	scope.WorkOrderID, err = parsePlanningClaimUUID(claims, "work_order_id")
	if err != nil {
		return nil, err
	}
	scope.CanvasRunID, err = parsePlanningClaimUUID(claims, "canvas_run_id")
	if err != nil {
		return nil, err
	}
	scope.NodeExecutionID, err = parsePlanningClaimUUID(claims, "node_execution_id")
	if err != nil {
		return nil, err
	}
	raw, _ := claims["enabled_checks"].(string)
	scope.EnabledChecks = parseMergeConfidenceChecksClaim(raw)
	return &scope, nil
}

func parseMergeConfidenceChecksClaim(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	enabled := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		enabled = append(enabled, name)
	}
	return enabled
}

func HasMergeConfidenceToken(environment []BrokerEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplaneMergeConfidenceToken)
	return ok && value != ""
}

// AttachMergeConfidenceEnv mints a runner token when the node carries a work
// order id. Other agent runs are unchanged. The work order id is removed from
// the task environment after the token is minted.
func AttachMergeConfidenceEnv(ctx core.ExecutionContext, environment []BrokerEnvironmentVariable, steps []AgentStep, timeoutSeconds int) ([]BrokerEnvironmentVariable, error) {
	orderID, ok := environmentValue(environment, EnvSuperplaneMergeConfidenceOrderID)
	if !ok {
		return environment, nil
	}
	environment = withoutEnvironmentName(environment, EnvSuperplaneMergeConfidenceOrderID)

	workOrderID, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("merge confidence work order id is invalid")
	}
	orgID, err := uuid.Parse(strings.TrimSpace(ctx.OrganizationID))
	if err != nil {
		return nil, fmt.Errorf("merge confidence organization is invalid")
	}
	canvasID, err := uuid.Parse(strings.TrimSpace(ctx.WorkflowID))
	if err != nil {
		return nil, fmt.Errorf("merge confidence canvas is invalid")
	}
	if ctx.RunID == uuid.Nil || ctx.ID == uuid.Nil {
		return nil, fmt.Errorf("merge confidence run is incomplete")
	}

	canvas, err := models.FindCanvasInTransaction(database.DB(context.Background()), orgID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("merge confidence canvas: %w", err)
	}
	if canvas.FactoryID == nil || *canvas.FactoryID == uuid.Nil {
		return nil, fmt.Errorf("merge confidence requires a factory canvas")
	}

	baseURL := RunnerSuperplaneBaseURL(ctx.BaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("merge confidence requires a SuperPlane URL")
	}
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		return nil, fmt.Errorf("merge confidence requires JWT_SECRET")
	}

	token, err := MintMergeConfidenceToken(jwt.NewSigner(secret), MergeConfidenceScope{
		OrganizationID:  orgID,
		FactoryID:       *canvas.FactoryID,
		WorkOrderID:     workOrderID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		EnabledChecks:   ParseMergeConfidenceChecks(agentStepPrompts(steps)),
	}, time.Duration(timeoutSeconds)*time.Second)
	if err != nil {
		return nil, fmt.Errorf("mint merge confidence token: %w", err)
	}

	if _, exists := environmentValue(environment, EnvSuperplaneBaseURL); !exists {
		environment = append(environment, BrokerEnvironmentVariable{
			Name:  EnvSuperplaneBaseURL,
			Value: baseURL,
		})
	}
	return append(environment, BrokerEnvironmentVariable{
		Name:  EnvSuperplaneMergeConfidenceToken,
		Value: token,
	}), nil
}

func agentStepPrompts(steps []AgentStep) []string {
	prompts := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		prompts = append(prompts, *step.Prompt)
	}
	return prompts
}

func environmentValue(environment []BrokerEnvironmentVariable, name string) (string, bool) {
	for _, item := range environment {
		if item.Name != name {
			continue
		}
		return strings.TrimSpace(item.Value), true
	}
	return "", false
}

func withoutEnvironmentName(environment []BrokerEnvironmentVariable, name string) []BrokerEnvironmentVariable {
	filtered := make([]BrokerEnvironmentVariable, 0, len(environment))
	for _, item := range environment {
		if item.Name == name {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}
