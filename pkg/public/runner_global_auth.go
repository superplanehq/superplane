package public

import (
	"net/http"

	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/database"
)

// authenticateRunner checks the runner token of the global runner MCP server.
// Every global runner endpoint calls it with the scope that its tool needs.
func (s *Server) authenticateRunner(w http.ResponseWriter, r *http.Request, required runneraction.RunnerToolScope) (*runneraction.RunnerScope, bool) {
	scope, err := runneraction.ParseRunnerToken(s.jwt, bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	if err := runneraction.ValidateRunnerScope(database.DB(r.Context()), scope); err != nil {
		http.Error(w, "Runner scope is invalid", http.StatusForbidden)
		return nil, false
	}
	if required != "" && !scope.HasScope(required) {
		http.Error(w, "This run cannot use this tool", http.StatusForbidden)
		return nil, false
	}
	return scope, true
}
