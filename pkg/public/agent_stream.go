package public

import (
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

// RegisterAgentStreamHandler registers the SSE stream route for agents.
func (s *Server) RegisterAgentStreamHandler(agentService *agents.Service) {
	streamHandler := agents.NewStreamHandler(agentService.Client, agentService.Store)

	s.Router.HandleFunc("/api/v1/agents/chats/{canvas_id}/stream", func(w http.ResponseWriter, r *http.Request) {
		// Authenticate via cookie (same as other routes)
		accountID, err := getAccountIDFromCookie(r, s.jwt)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		orgID := r.Header.Get("X-Organization-Id")
		if orgID == "" {
			// Try query param
			orgID = r.URL.Query().Get("organization_id")
		}
		if orgID == "" {
			http.Error(w, "organization ID required", http.StatusBadRequest)
			return
		}

		// Resolve user from account + org
		account, err := models.FindAccountByID(accountID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := models.FindActiveUserByEmail(orgID, account.Email)
		if err != nil {
			http.Error(w, "user not found in organization", http.StatusForbidden)
			return
		}

		// Extract canvas_id from path
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		var canvasID string
		for i, part := range parts {
			if part == "chats" && i+1 < len(parts) {
				canvasID = parts[i+1]
				break
			}
		}
		if canvasID == "" || canvasID == "stream" {
			http.Error(w, "invalid canvas ID", http.StatusBadRequest)
			return
		}

		streamHandler.HandleStream(w, r, orgID, user.ID.String(), canvasID)
	}).Methods("POST")
}

func getAccountIDFromCookie(r *http.Request, signer *jwt.Signer) (string, error) {
	cookie, err := r.Cookie("account_token")
	if err != nil {
		return "", err
	}

	claims, err := signer.ValidateAndGetClaims(cookie.Value)
	if err != nil {
		return "", err
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", err
	}

	return sub, nil
}
