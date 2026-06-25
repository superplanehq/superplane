package public

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

const verifyOIDCTokenFailedMessage = "token verification failed"

type verifyOIDCTokenRequest struct {
	Token string `json:"token"`
}

type verifyOIDCTokenResponse struct {
	Valid  bool           `json:"valid"`
	Claims map[string]any `json:"claims,omitempty"`
	Error  string         `json:"error,omitempty"`
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

	claims, err := s.oidcProvider.Validate(request.Token)
	if err != nil {
		logOIDCVerificationFailure("token validation failed", err)
		respondVerifyOIDCTokenError(w, http.StatusUnauthorized, verifyOIDCTokenFailedMessage)
		return
	}

	respondJSON(w, verifyOIDCTokenResponse{
		Valid:  true,
		Claims: claims,
	})
}

func logOIDCVerificationFailure(reason string, err error) {
	log.WithError(err).Warn("OIDC token verification failed: " + reason)
}

func respondVerifyOIDCTokenError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	respondJSON(w, verifyOIDCTokenResponse{
		Valid: false,
		Error: message,
	})
}
