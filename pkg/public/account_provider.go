package public

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func (s *Server) connectGitHubAccount(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if impersonation, ok := middleware.GetImpersonationFromContext(r.Context()); ok && impersonation.Active {
		http.Error(w, "Provider linking is unavailable during impersonation", http.StatusForbidden)
		return
	}

	s.authHandler.BeginGitHubAccountLink(w, r, account)
}

func (s *Server) completeGitHubAccountLink(w http.ResponseWriter, r *http.Request) {
	account, ok := middleware.GetAccountFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if impersonation, ok := middleware.GetImpersonationFromContext(r.Context()); ok && impersonation.Active {
		http.Error(w, "Provider linking is unavailable during impersonation", http.StatusForbidden)
		return
	}

	s.authHandler.CompleteGitHubAccountLink(w, r, account)
}
