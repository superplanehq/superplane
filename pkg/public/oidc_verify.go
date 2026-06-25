package public

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/ciauth"
)

const verifyOIDCTokenFailedMessage = "token verification failed"

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

	claims, err := ciauth.ValidateToken(s.oidcProvider, request.Token)
	if err != nil {
		logOIDCVerificationFailure("token validation failed", err)
		respondVerifyOIDCTokenError(w, http.StatusUnauthorized, verifyOIDCTokenFailedMessage)
		return
	}

	if request.Expected != nil {
		if err := toExecutionTokenExpected(*request.Expected).Matches(claims); err != nil {
			logOIDCVerificationFailure("expected claim mismatch", err)
			respondVerifyOIDCTokenError(w, http.StatusForbidden, verifyOIDCTokenFailedMessage)
			return
		}
	}

	respondJSON(w, verifyOIDCTokenResponse{
		Valid:  true,
		Claims: toVerifyOIDCTokenResponseClaims(claims),
	})
}

func toExecutionTokenExpected(expected verifyOIDCTokenExpectedClaims) ciauth.ExecutionTokenExpected {
	additional := map[string]string{}
	if expected.ProjectID != "" {
		additional["project_id"] = expected.ProjectID
	}
	if expected.PipelineFile != "" {
		additional["pipeline_file"] = expected.PipelineFile
	}
	if expected.Ref != "" {
		additional["ref"] = expected.Ref
	}
	if expected.CommitSha != "" {
		additional["commit_sha"] = expected.CommitSha
	}

	return ciauth.ExecutionTokenExpected{
		OrgID:      expected.OrgID,
		CanvasID:   expected.CanvasID,
		NodeID:     expected.NodeID,
		Component:  expected.Component,
		Additional: additional,
	}
}

func toVerifyOIDCTokenResponseClaims(claims ciauth.ExecutionTokenClaims) *verifyOIDCTokenResponseClaims {
	return &verifyOIDCTokenResponseClaims{
		OrgID:        claims.OrgID,
		CanvasID:     claims.CanvasID,
		NodeID:       claims.NodeID,
		ExecutionID:  claims.ExecutionID,
		Component:    claims.Component,
		ProjectID:    claims.Additional["project_id"],
		PipelineFile: claims.Additional["pipeline_file"],
		Ref:          claims.Additional["ref"],
		CommitSha:    claims.Additional["commit_sha"],
	}
}

func logOIDCVerificationFailure(reason string, err error) {
	log.WithError(err).Warn("OIDC execution token verification failed: " + reason)
}

func respondVerifyOIDCTokenError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	respondJSON(w, verifyOIDCTokenResponse{
		Valid: false,
		Error: message,
	})
}
