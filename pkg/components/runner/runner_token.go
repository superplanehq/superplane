package runner

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/jwt"
)

const (
	RunnerTokenPurpose       = "runner_run"
	EnvSuperplaneRunnerToken = "SUPERPLANE_RUNNER_TOKEN"
)

// RunnerScope is the signed identity of one agent run. The global runner MCP
// server and the runner CLI use it as their only credential.
type RunnerScope struct {
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	// WorkOrderID is uuid.Nil in planning. The draft task of a session can
	// change, so the backend reads it from the session on every request.
	WorkOrderID       uuid.UUID
	PlanningSessionID uuid.UUID
	CanvasID          uuid.UUID
	CanvasRunID       uuid.UUID
	NodeExecutionID   uuid.UUID
	NodeID            string
	Flow              RunnerFlow
	Scopes            []RunnerToolScope
	// MergeChecks and MergeCheckLabels come from the merge confidence token.
	// The backend reports a merge check only when its id is in MergeChecks.
	MergeChecks      []string
	MergeCheckLabels map[string]string
}

func (s *RunnerScope) HasScope(scope RunnerToolScope) bool {
	return slices.Contains(s.Scopes, scope)
}

func (s RunnerScope) complete() error {
	if s.OrganizationID == uuid.Nil || s.FactoryID == uuid.Nil || s.CanvasID == uuid.Nil ||
		s.CanvasRunID == uuid.Nil || s.NodeExecutionID == uuid.Nil || strings.TrimSpace(s.NodeID) == "" {
		return fmt.Errorf("runner scope is incomplete")
	}
	if !s.Flow.Valid() {
		return fmt.Errorf("invalid runner flow %q", s.Flow)
	}
	if s.Flow == RunnerFlowPlanning && s.PlanningSessionID == uuid.Nil {
		return fmt.Errorf("planning runner scope needs a planning session")
	}
	if s.Flow != RunnerFlowPlanning && s.WorkOrderID == uuid.Nil {
		return fmt.Errorf("runner scope needs a task")
	}
	return nil
}

func MintRunnerToken(signer *jwt.Signer, scope RunnerScope, ttl time.Duration) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("jwt signer is required")
	}
	if err := scope.complete(); err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultExecutionTimeoutSeconds)*time.Second + artifactTokenGracePeriod
	}
	claims := map[string]string{
		"purpose":             RunnerTokenPurpose,
		"org_id":              scope.OrganizationID.String(),
		"factory_id":          scope.FactoryID.String(),
		"work_order_id":       optionalUUIDClaim(scope.WorkOrderID),
		"planning_session_id": optionalUUIDClaim(scope.PlanningSessionID),
		"canvas_id":           scope.CanvasID.String(),
		"canvas_run_id":       scope.CanvasRunID.String(),
		"node_execution_id":   scope.NodeExecutionID.String(),
		"node_id":             scope.NodeID,
		"flow":                string(scope.Flow),
		"scopes":              joinRunnerScopes(scope.Scopes),
		"merge_checks":        strings.Join(scope.MergeChecks, ","),
	}
	if len(scope.MergeCheckLabels) > 0 {
		encoded, err := json.Marshal(scope.MergeCheckLabels)
		if err != nil {
			return "", fmt.Errorf("merge check labels: %w", err)
		}
		claims["merge_check_labels"] = string(encoded)
	}
	return signer.GenerateWithClaims(ttl, claims)
}

func ParseRunnerToken(signer *jwt.Signer, token string) (*RunnerScope, error) {
	if signer == nil {
		return nil, fmt.Errorf("jwt signer is required")
	}
	claims, err := signer.ValidateAndGetClaims(token)
	if err != nil {
		return nil, err
	}
	if claims["purpose"] != RunnerTokenPurpose {
		return nil, fmt.Errorf("invalid runner token purpose")
	}
	scope := &RunnerScope{}
	for name, destination := range map[string]*uuid.UUID{
		"org_id": &scope.OrganizationID, "factory_id": &scope.FactoryID, "canvas_id": &scope.CanvasID,
		"canvas_run_id": &scope.CanvasRunID, "node_execution_id": &scope.NodeExecutionID,
	} {
		*destination, err = parsePlanningClaimUUID(claims, name)
		if err != nil {
			return nil, err
		}
	}
	for name, destination := range map[string]*uuid.UUID{
		"work_order_id": &scope.WorkOrderID, "planning_session_id": &scope.PlanningSessionID,
	} {
		*destination, err = parseOptionalUUIDClaim(claims, name)
		if err != nil {
			return nil, err
		}
	}
	scope.NodeID, _ = claims["node_id"].(string)
	flow, _ := claims["flow"].(string)
	scope.Flow = RunnerFlow(flow)
	rawScopes, _ := claims["scopes"].(string)
	scope.Scopes = splitRunnerScopes(rawScopes)
	if rawChecks, _ := claims["merge_checks"].(string); strings.TrimSpace(rawChecks) != "" {
		scope.MergeChecks = parseMergeConfidenceChecksClaim(rawChecks)
	}
	labels, _ := claims["merge_check_labels"].(string)
	scope.MergeCheckLabels = parseMergeConfidenceCheckLabels(labels)
	if err := scope.complete(); err != nil {
		return nil, err
	}
	return scope, nil
}

// PlanningSessionScope returns the planning session of a planning run.
func (s *RunnerScope) PlanningSessionScope() (*PlanningSessionScope, bool) {
	if s.Flow != RunnerFlowPlanning {
		return nil, false
	}
	return &PlanningSessionScope{
		OrganizationID: s.OrganizationID,
		FactoryID:      s.FactoryID,
		SessionID:      s.PlanningSessionID,
		CanvasRunID:    s.CanvasRunID,
	}, true
}

// ArtifactUploadScope returns the task artifact scope of a task run.
func (s *RunnerScope) ArtifactUploadScope() (*ArtifactUploadScope, bool) {
	if s.Flow != RunnerFlowTask {
		return nil, false
	}
	return &ArtifactUploadScope{
		OrganizationID:  s.OrganizationID,
		FactoryID:       s.FactoryID,
		WorkOrderID:     s.WorkOrderID,
		CanvasID:        s.CanvasID,
		CanvasRunID:     s.CanvasRunID,
		NodeExecutionID: s.NodeExecutionID,
		NodeID:          s.NodeID,
	}, true
}

// MergeConfidenceScope returns the merge check scope of a merge confidence run.
func (s *RunnerScope) MergeConfidenceScope() (*MergeConfidenceScope, bool) {
	if s.Flow != RunnerFlowMergeConfidence {
		return nil, false
	}
	return &MergeConfidenceScope{
		OrganizationID:  s.OrganizationID,
		FactoryID:       s.FactoryID,
		WorkOrderID:     s.WorkOrderID,
		CanvasRunID:     s.CanvasRunID,
		NodeExecutionID: s.NodeExecutionID,
		EnabledChecks:   s.MergeChecks,
		CheckLabels:     s.MergeCheckLabels,
	}, true
}

func HasRunnerToken(environment []TaskEnvironmentVariable) bool {
	value, ok := environmentValue(environment, EnvSuperplaneRunnerToken)
	return ok && value != ""
}

func optionalUUIDClaim(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

func parseOptionalUUIDClaim(claims map[string]any, key string) (uuid.UUID, error) {
	raw, _ := claims[key].(string)
	if strings.TrimSpace(raw) == "" {
		return uuid.Nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func joinRunnerScopes(scopes []RunnerToolScope) string {
	names := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names = append(names, string(scope))
	}
	return strings.Join(names, " ")
}

func splitRunnerScopes(raw string) []RunnerToolScope {
	scopes := []RunnerToolScope{}
	for _, name := range strings.Fields(raw) {
		scopes = append(scopes, RunnerToolScope(name))
	}
	return scopes
}
