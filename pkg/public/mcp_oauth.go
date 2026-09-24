package public

import (
	"net/http"

	factoryactions "github.com/superplanehq/superplane/pkg/grpc/actions/factories"
	"github.com/superplanehq/superplane/pkg/mcp"
)

func (s *Server) HandleMCPOAuthCallback(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	baseURL := s.mcpOAuthBaseURL()
	httpClient := mcp.DoerFromCore(nil)
	if s.registry != nil {
		httpClient = mcp.DoerFromCore(s.registry.HTTPContext())
	}

	path, status, message := factoryactions.CompleteFactoryAgentResourceOAuth(
		r.Context(),
		s.encryptor,
		httpClient,
		baseURL,
		r.URL.Query().Get("code"),
		state,
		r.URL.Query().Get("error"),
	)
	if status == http.StatusFound && path != "" {
		http.Redirect(w, r, path, http.StatusFound)
		return
	}
	if message == "" {
		message = "MCP OAuth failed"
	}
	if status == 0 {
		status = http.StatusInternalServerError
	}
	http.Error(w, message, status)
}

func (s *Server) HandleMCPOAuthClientMetadata(w http.ResponseWriter, r *http.Request) {
	baseURL := s.mcpOAuthBaseURL()
	respondJSON(w, mcp.ClientMetadataDocument(mcp.ClientMetadataURL(baseURL), mcp.CallbackURL(baseURL)))
}

func (s *Server) mcpOAuthBaseURL() string {
	if s.WebhooksBaseURL != "" {
		return s.WebhooksBaseURL
	}
	return s.BaseURL
}
