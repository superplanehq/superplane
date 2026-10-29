package public

import (
	"net/http"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
)

func (s *Server) handleRunnerTools(w http.ResponseWriter, r *http.Request) {
	scope, ok := s.authenticateRunner(w, r, "")
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tools": runneraction.AllowedRunnerTools(scope.Scopes),
	})
}
