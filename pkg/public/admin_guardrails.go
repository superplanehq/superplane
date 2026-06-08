package public

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/guardrails"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// ── request / response types ─────────────────────────────────────────────────

type guardrailPolicyResponse struct {
	OrgID                   string   `json:"org_id"`
	WorkflowID              *string  `json:"workflow_id,omitempty"`
	NodeID                  *string  `json:"node_id,omitempty"`
	ComponentType           *string  `json:"component_type,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode"`
	SoftBlockScoreThreshold int      `json:"soft_block_score_threshold"`
	HardBlockScoreThreshold int      `json:"hard_block_score_threshold"`
	ClassifierEnabled       bool     `json:"classifier_enabled"`
	ClassifierSamplingRate  float64  `json:"classifier_sampling_rate"`
	ClassifierSensitivity   string   `json:"classifier_sensitivity"`
	SoftBlockTimeoutSeconds int      `json:"soft_block_timeout_seconds"`
	UpdatedAt               *string  `json:"updated_at,omitempty"`
}

type upsertGuardrailPolicyRequest struct {
	EnforcementMode         string  `json:"enforcement_mode"`
	SoftBlockScoreThreshold int     `json:"soft_block_score_threshold"`
	HardBlockScoreThreshold int     `json:"hard_block_score_threshold"`
	ClassifierEnabled       bool    `json:"classifier_enabled"`
	ClassifierSamplingRate  float64 `json:"classifier_sampling_rate"`
	ClassifierSensitivity   string  `json:"classifier_sensitivity"`
	SoftBlockTimeoutSeconds int     `json:"soft_block_timeout_seconds"`
}

type pendingOverrideResponse struct {
	ID              string    `json:"id"`
	ExecutionID     string    `json:"execution_id"`
	OrgID           string    `json:"org_id"`
	WorkflowID      string    `json:"workflow_id"`
	NodeID          string    `json:"node_id"`
	RiskScore       int       `json:"risk_score"`
	FindingsCount   int       `json:"findings_count"`
	ComponentType   *string   `json:"component_type,omitempty"`
	CreatedAt       string    `json:"created_at"`
}

type approveOverrideRequest struct {
	Justification string `json:"justification"`
}

// ── org-level policy handlers ─────────────────────────────────────────────────

func (s *Server) adminGetOrgGuardrailPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	if _, err := models.FindOrganizationByID(orgID.String()); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	policy, err := guardrails.GetOrgPolicy(orgID)
	if err != nil {
		log.Errorf("admin guardrails: get org policy %s: %v", orgID, err)
		http.Error(w, "Failed to load policy", http.StatusInternalServerError)
		return
	}

	respondJSON(w, serializePolicy(policy))
}

func (s *Server) adminUpsertOrgGuardrailPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	if _, err := models.FindOrganizationByID(orgID.String()); err != nil {
		http.Error(w, "Organization not found", http.StatusNotFound)
		return
	}

	caller, callerOK := middleware.GetAccountFromContext(r.Context())
	callerID := uuid.Nil
	if callerOK {
		callerID = caller.ID
	}

	var req upsertGuardrailPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SoftBlockTimeoutSeconds == 0 {
		req.SoftBlockTimeoutSeconds = 86400
	}
	if req.ClassifierSensitivity == "" {
		req.ClassifierSensitivity = models.GuardrailClassifierSensitivityBalanced
	}

	if err := guardrails.UpsertOrgPolicy(guardrails.UpsertOrgPolicyRequest{
		OrgID:                   orgID,
		CallerID:                callerID,
		EnforcementMode:         req.EnforcementMode,
		SoftBlockScoreThreshold: req.SoftBlockScoreThreshold,
		HardBlockScoreThreshold: req.HardBlockScoreThreshold,
		ClassifierEnabled:       req.ClassifierEnabled,
		ClassifierSamplingRate:  req.ClassifierSamplingRate,
		ClassifierSensitivity:   req.ClassifierSensitivity,
		SoftBlockTimeoutSeconds: req.SoftBlockTimeoutSeconds,
	}); err != nil {
		log.Errorf("admin guardrails: upsert org policy %s: %v", orgID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policy, err := guardrails.GetOrgPolicy(orgID)
	if err != nil {
		http.Error(w, "Policy saved but failed to reload", http.StatusInternalServerError)
		return
	}

	respondJSON(w, serializePolicy(policy))
}

// ── workflow-level policy handlers ───────────────────────────────────────────

func (s *Server) adminGetWorkflowGuardrailPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	canvasID, ok := parseCanvasID(w, r)
	if !ok {
		return
	}

	policy, err := models.FindPromptGuardrailPolicyForWorkflow(orgID, canvasID)
	if err != nil {
		// No workflow-specific policy — fall back to org policy.
		policy, err = guardrails.GetOrgPolicy(orgID)
		if err != nil {
			http.Error(w, "Failed to load policy", http.StatusInternalServerError)
			return
		}
	}

	respondJSON(w, serializePolicy(policy))
}

func (s *Server) adminUpsertWorkflowGuardrailPolicy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	canvasID, ok := parseCanvasID(w, r)
	if !ok {
		return
	}

	caller, callerOK := middleware.GetAccountFromContext(r.Context())
	callerID := uuid.Nil
	if callerOK {
		callerID = caller.ID
	}

	var req upsertGuardrailPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SoftBlockTimeoutSeconds == 0 {
		req.SoftBlockTimeoutSeconds = 86400
	}
	if req.ClassifierSensitivity == "" {
		req.ClassifierSensitivity = models.GuardrailClassifierSensitivityBalanced
	}

	if err := upsertWorkflowPolicy(orgID, canvasID, callerID, req); err != nil {
		log.Errorf("admin guardrails: upsert workflow policy org=%s canvas=%s: %v", orgID, canvasID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	policy, err := models.FindPromptGuardrailPolicyForWorkflow(orgID, canvasID)
	if err != nil {
		http.Error(w, "Policy saved but failed to reload", http.StatusInternalServerError)
		return
	}

	respondJSON(w, serializePolicy(policy))
}

// ── override management ───────────────────────────────────────────────────────

func (s *Server) adminListGuardrailOverrides(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	results, err := guardrails.ListPendingOverrides(orgID)
	if err != nil {
		log.Errorf("admin guardrails: list overrides org=%s: %v", orgID, err)
		http.Error(w, "Failed to load pending overrides", http.StatusInternalServerError)
		return
	}

	items := make([]pendingOverrideResponse, 0, len(results))
	for _, r := range results {
		item := pendingOverrideResponse{
			ID:            r.ID.String(),
			ExecutionID:   r.ExecutionID.String(),
			OrgID:         r.OrgID.String(),
			WorkflowID:    r.WorkflowID.String(),
			NodeID:        r.NodeID,
			RiskScore:     r.RiskScore,
			FindingsCount: len(r.Findings.Data()),
			CreatedAt:     r.CreatedAt.Format(time.RFC3339),
		}
		if r.ComponentType != nil {
			item.ComponentType = r.ComponentType
		}
		items = append(items, item)
	}

	respondJSON(w, map[string]any{"items": items, "total": len(items)})
}

func (s *Server) adminApproveGuardrailOverride(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}
	_ = orgID

	scanResultIDStr := mux.Vars(r)["scanResultId"]
	scanResultID, err := uuid.Parse(scanResultIDStr)
	if err != nil {
		http.Error(w, "Invalid scan result ID", http.StatusBadRequest)
		return
	}

	caller, callerOK := middleware.GetAccountFromContext(r.Context())
	if !callerOK {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req approveOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := guardrails.ApproveOverride(scanResultID, caller.ID, req.Justification); err != nil {
		log.Errorf("admin guardrails: approve override %s: %v", scanResultID, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	respondJSON(w, map[string]string{"status": "approved"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func parseOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(mux.Vars(r)["orgId"])
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func parseCanvasID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(mux.Vars(r)["canvasId"])
	if err != nil {
		http.Error(w, "Invalid canvas ID", http.StatusBadRequest)
		return uuid.Nil, false
	}
	return id, true
}

func serializePolicy(p *models.PromptGuardrailPolicy) guardrailPolicyResponse {
	resp := guardrailPolicyResponse{
		OrgID:                   p.OrgID.String(),
		EnforcementMode:         p.EnforcementMode,
		SoftBlockScoreThreshold: p.SoftBlockScoreThreshold,
		HardBlockScoreThreshold: p.HardBlockScoreThreshold,
		ClassifierEnabled:       p.ClassifierEnabled,
		ClassifierSamplingRate:  p.ClassifierSamplingRate,
		ClassifierSensitivity:   p.ClassifierSensitivity,
		SoftBlockTimeoutSeconds: p.SoftBlockTimeoutSeconds,
	}
	if p.WorkflowID != nil {
		s := p.WorkflowID.String()
		resp.WorkflowID = &s
	}
	if p.NodeID != nil {
		resp.NodeID = p.NodeID
	}
	if p.ComponentType != nil {
		resp.ComponentType = p.ComponentType
	}
	if !p.UpdatedAt.IsZero() {
		s := p.UpdatedAt.Format(time.RFC3339)
		resp.UpdatedAt = &s
	}
	return resp
}

func upsertWorkflowPolicy(orgID, workflowID, callerID uuid.UUID, req upsertGuardrailPolicyRequest) error {
	existing, err := models.FindPromptGuardrailPolicyForWorkflow(orgID, workflowID)
	if err == nil && existing != nil {
		return models.UpdateWorkflowGuardrailPolicy(existing.ID, callerID, map[string]any{
			"enforcement_mode":           req.EnforcementMode,
			"soft_block_score_threshold": req.SoftBlockScoreThreshold,
			"hard_block_score_threshold": req.HardBlockScoreThreshold,
			"classifier_enabled":         req.ClassifierEnabled,
			"classifier_sampling_rate":   req.ClassifierSamplingRate,
			"classifier_sensitivity":     req.ClassifierSensitivity,
			"soft_block_timeout_seconds": req.SoftBlockTimeoutSeconds,
		})
	}
	return models.CreateWorkflowGuardrailPolicy(orgID, workflowID, callerID, models.PromptGuardrailPolicy{
		OrgID:                   orgID,
		WorkflowID:              &workflowID,
		EnforcementMode:         req.EnforcementMode,
		SoftBlockScoreThreshold: req.SoftBlockScoreThreshold,
		HardBlockScoreThreshold: req.HardBlockScoreThreshold,
		ClassifierEnabled:       req.ClassifierEnabled,
		ClassifierSamplingRate:  req.ClassifierSamplingRate,
		ClassifierSensitivity:   req.ClassifierSensitivity,
		SoftBlockTimeoutSeconds: req.SoftBlockTimeoutSeconds,
		CreatedBy:               callerID,
	})
}
