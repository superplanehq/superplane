package public

import (
	"net/http"

	"github.com/superplanehq/superplane/pkg/public/middleware"
)

func hasActiveSupportSession(r *http.Request) bool {
	return middleware.IsSupportSessionActive(r)
}

// rejectInSupportSession guards handlers that change state on GET, which
// the method-based block in the auth middleware does not catch.
func rejectInSupportSession(w http.ResponseWriter, r *http.Request) bool {
	if !hasActiveSupportSession(r) {
		return false
	}

	http.Error(w, middleware.SupportSessionReadOnlyMessage, http.StatusForbidden)
	return true
}
