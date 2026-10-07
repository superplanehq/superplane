package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
	// EnvSuperplaneMergeConfidenceMaxScore tells the report tool which range
	// the check prompts use. 5 means at least one prompt still asks for 1 to 5.
	EnvSuperplaneMergeConfidenceMaxScore = "SUPERPLANE_MERGE_CONFIDENCE_MAX_SCORE"

	mergeConfidenceCurrentMaxScore = 3
	mergeConfidenceLegacyMaxScore  = 5
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
	// CheckLabels maps a check id to the step name. Built-in checks use their
	// own names when this map has no entry.
	CheckLabels map[string]string
	// CheckMaxScores maps a check id to 3 or 5. A nil map is a token minted
	// before score scales existed. Those runs still use 1 through 5.
	CheckMaxScores map[string]int
}

var (
	mergeConfidenceChecksLine   = regexp.MustCompile(`(?m)^Enabled checks: (none|(?:risk|performance|security|drift|reversibility)(?:, (?:risk|performance|security|drift|reversibility))*)\.$`)
	mergeCheckIDLine            = regexp.MustCompile(`(?m)^Merge check: ([a-z][a-z0-9-]{0,40})\.$`)
	mergeConfidenceLegacyScore  = regexp.MustCompile(`(?m)(?:\b1 to 5\b|\b1 through 5\b|\breport(?: the check)? with 5\b|^\s*[45] means\b| = [45] \()`)
	mergeConfidenceCurrentScore = regexp.MustCompile(`\b1 to 3\b|\b1 through 3\b`)
)

// MergeCheckID reads the check id from a step prompt. An empty string means the step is not a check.
func MergeCheckID(prompt string) string {
	match := mergeCheckIDLine.FindStringSubmatch(prompt)
	if match == nil {
		return ""
	}
	return match[1]
}

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

// MergeConfidenceChecksFromSteps reads one check from each step that starts
// with a Merge check line. When no step has that line, it reads the older
// Enabled checks line and returns no labels.
func MergeConfidenceChecksFromSteps(steps []AgentStep) ([]string, map[string]string) {
	ids := make([]string, 0)
	labels := map[string]string{}
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		id := MergeCheckID(*step.Prompt)
		if id == "" {
			continue
		}
		if _, seen := labels[id]; seen {
			continue
		}
		ids = append(ids, id)
		label := strings.TrimSpace(step.Name)
		if label == "" {
			label = id
		}
		labels[id] = label
	}
	if len(ids) > 0 {
		return ids, labels
	}
	return ParseMergeConfidenceChecks(agentStepPrompts(steps)), nil
}

// MergeConfidenceCheckScales reads the score range from each check prompt.
// A prompt that still asks for 1 to 5 stays on that scale. A prompt that asks
// for 1 to 3 uses the new scale. A prompt that names neither stays on 1 to 5.
func MergeConfidenceCheckScales(steps []AgentStep) map[string]int {
	scales := map[string]int{}
	sawCheck := false
	for _, step := range steps {
		if step.Prompt == nil {
			continue
		}
		id := MergeCheckID(*step.Prompt)
		if id == "" {
			continue
		}
		sawCheck = true
		scales[id] = mergeConfidencePromptScale(*step.Prompt)
	}
	if sawCheck {
		return scales
	}
	scale := mergeConfidenceLegacyMaxScore
	prompts := agentStepPrompts(steps)
	if promptScale(prompts) == mergeConfidenceCurrentMaxScore {
		scale = mergeConfidenceCurrentMaxScore
	}
	for _, id := range ParseMergeConfidenceChecks(prompts) {
		scales[id] = scale
	}
	return scales
}

func promptScale(prompts []string) int {
	if len(prompts) == 0 {
		return mergeConfidenceLegacyMaxScore
	}
	scale := mergeConfidenceLegacyMaxScore
	for _, prompt := range prompts {
		next := mergeConfidencePromptScale(prompt)
		if next == mergeConfidenceLegacyMaxScore && mergeConfidenceLegacyScore.MatchString(prompt) {
			return mergeConfidenceLegacyMaxScore
		}
		if next == mergeConfidenceCurrentMaxScore {
			scale = mergeConfidenceCurrentMaxScore
		}
	}
	return scale
}

func mergeConfidencePromptScale(prompt string) int {
	if mergeConfidenceLegacyScore.MatchString(prompt) {
		return mergeConfidenceLegacyMaxScore
	}
	if mergeConfidenceCurrentScore.MatchString(prompt) {
		return mergeConfidenceCurrentMaxScore
	}
	return mergeConfidenceLegacyMaxScore
}

// CheckScoreScale is the range for one reported check. A token with no scale
// claim still uses 1 through 5.
func (scope MergeConfidenceScope) CheckScoreScale(check string) float64 {
	if scope.CheckMaxScores == nil {
		return mergeConfidenceLegacyMaxScore
	}
	if scope.CheckMaxScores[strings.ToLower(strings.TrimSpace(check))] == mergeConfidenceLegacyMaxScore {
		return mergeConfidenceLegacyMaxScore
	}
	return mergeConfidenceCurrentMaxScore
}

func mergeConfidenceRunMaxScore(scales map[string]int) int {
	for _, scale := range scales {
		if scale == mergeConfidenceLegacyMaxScore {
			return mergeConfidenceLegacyMaxScore
		}
	}
	return mergeConfidenceCurrentMaxScore
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
	claims := map[string]string{
		"purpose":           MergeConfidenceTokenPurpose,
		"org_id":            scope.OrganizationID.String(),
		"factory_id":        scope.FactoryID.String(),
		"work_order_id":     scope.WorkOrderID.String(),
		"canvas_run_id":     scope.CanvasRunID.String(),
		"node_execution_id": scope.NodeExecutionID.String(),
		"enabled_checks":    strings.Join(scope.EnabledChecks, ","),
	}
	if len(scope.CheckLabels) > 0 {
		encoded, err := json.Marshal(scope.CheckLabels)
		if err != nil {
			return "", fmt.Errorf("merge confidence check names: %w", err)
		}
		claims["check_labels"] = string(encoded)
	}
	if scope.CheckMaxScores != nil {
		encoded, err := json.Marshal(scope.CheckMaxScores)
		if err != nil {
			return "", fmt.Errorf("merge confidence score scale: %w", err)
		}
		claims["check_max_scores"] = string(encoded)
	}
	return signer.GenerateWithClaims(ttl, claims)
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
	labels, _ := claims["check_labels"].(string)
	scope.CheckLabels = parseMergeConfidenceCheckLabels(labels)
	if rawScale, ok := claims["check_max_scores"].(string); ok {
		scales, err := parseMergeConfidenceCheckScales(rawScale)
		if err != nil {
			return nil, err
		}
		scope.CheckMaxScores = scales
	}
	return &scope, nil
}

func parseMergeConfidenceCheckLabels(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var labels map[string]string
	if err := json.Unmarshal([]byte(raw), &labels); err != nil {
		return nil
	}
	return labels
}

func parseMergeConfidenceCheckScales(raw string) (map[string]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]int{}, nil
	}
	var scales map[string]int
	if err := json.Unmarshal([]byte(raw), &scales); err != nil {
		return nil, fmt.Errorf("merge confidence score scale is invalid")
	}
	if scales == nil {
		return map[string]int{}, nil
	}
	return scales, nil
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

	enabled, labels := MergeConfidenceChecksFromSteps(steps)
	scales := MergeConfidenceCheckScales(steps)
	token, err := MintMergeConfidenceToken(jwt.NewSigner(secret), MergeConfidenceScope{
		OrganizationID:  orgID,
		FactoryID:       *canvas.FactoryID,
		WorkOrderID:     workOrderID,
		CanvasRunID:     ctx.RunID,
		NodeExecutionID: ctx.ID,
		EnabledChecks:   enabled,
		CheckLabels:     labels,
		CheckMaxScores:  scales,
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
	environment = withoutEnvironmentName(environment, EnvSuperplaneMergeConfidenceMaxScore)
	return append(environment,
		BrokerEnvironmentVariable{
			Name:  EnvSuperplaneMergeConfidenceToken,
			Value: token,
		},
		BrokerEnvironmentVariable{
			Name:  EnvSuperplaneMergeConfidenceMaxScore,
			Value: strconv.Itoa(mergeConfidenceRunMaxScore(scales)),
		},
	), nil
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
