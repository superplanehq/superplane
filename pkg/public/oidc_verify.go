package public

import (
	"encoding/json"
	"net/http"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/oidc"
)

type verifyOIDCTokenRequest struct {
	Token    string                         `json:"token"`
	Expected *verifyOIDCTokenExpectedClaims `json:"expected,omitempty"`
}

type verifyOIDCTokenExpectedClaims struct {
	OrgID        string `json:"org_id,omitempty"`
	CanvasID     string `json:"canvas_id,omitempty"`
	NodeID       string `json:"node_id,omitempty"`
	Component    string `json:"component,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	PipelineFile string `json:"pipeline_file,omitempty"`
	Ref          string `json:"ref,omitempty"`
	CommitSha    string `json:"commit_sha,omitempty"`
}

type verifyOIDCTokenResponse struct {
	Valid  bool                           `json:"valid"`
	Claims *verifyOIDCTokenResponseClaims `json:"claims,omitempty"`
	Error  string                         `json:"error,omitempty"`
}

type verifyOIDCTokenResponseClaims struct {
	OrgID        string `json:"org_id"`
	CanvasID     string `json:"canvas_id"`
	NodeID       string `json:"node_id"`
	ExecutionID  string `json:"execution_id"`
	Component    string `json:"component,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	PipelineFile string `json:"pipeline_file,omitempty"`
	Ref          string `json:"ref,omitempty"`
	CommitSha    string `json:"commit_sha,omitempty"`
}

func (s *Server) handleVerifyOIDCToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request verifyOIDCTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondVerifyOIDCTokenError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if request.Token == "" {
		respondVerifyOIDCTokenError(w, http.StatusBadRequest, "token is required")
		return
	}

	claims, err := oidc.ValidateExecutionToken(s.oidcProvider, request.Token)
	if err != nil {
		respondVerifyOIDCTokenError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if request.Expected != nil {
		expected := oidc.ExecutionTokenExpected{
			OrgID:        request.Expected.OrgID,
			CanvasID:     request.Expected.CanvasID,
			NodeID:       request.Expected.NodeID,
			Component:    request.Expected.Component,
			ProjectID:    request.Expected.ProjectID,
			PipelineFile: request.Expected.PipelineFile,
			Ref:          request.Expected.Ref,
			CommitSha:    request.Expected.CommitSha,
		}
		if err := expected.Matches(claims); err != nil {
			respondVerifyOIDCTokenError(w, http.StatusForbidden, err.Error())
			return
		}
	}

	if err := authorizeExecutionToken(database.Conn(), claims); err != nil {
		respondVerifyOIDCTokenError(w, http.StatusForbidden, err.Error())
		return
	}

	respondJSON(w, verifyOIDCTokenResponse{
		Valid: true,
		Claims: &verifyOIDCTokenResponseClaims{
			OrgID:        claims.OrgID,
			CanvasID:     claims.CanvasID,
			NodeID:       claims.NodeID,
			ExecutionID:  claims.ExecutionID,
			Component:    claims.Component,
			ProjectID:    claims.ProjectID,
			PipelineFile: claims.PipelineFile,
			Ref:          claims.Ref,
			CommitSha:    claims.CommitSha,
		},
	})
}

func respondVerifyOIDCTokenError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	respondJSON(w, verifyOIDCTokenResponse{
		Valid: false,
		Error: message,
	})
}
