package middleware

import (
	"net/http"
	"regexp"
	"slices"
)

const SupportSessionReadOnlyMessage = "This action is not available in a read-only support session"

var supportSessionReadMethods = []string{http.MethodGet, http.MethodHead, http.MethodOptions}

// Non-read requests that a read-only support session can still send.
// Ending the session must stay possible, and some read endpoints use POST.
var supportSessionAllowedWrites = []*regexp.Regexp{
	regexp.MustCompile(`^/admin/api/support-session/end$`),
	regexp.MustCompile(`^/api/v1/factories/[^/]+/scheduled-intake/preview$`),
	regexp.MustCompile(`^/admin/api/installation/llm-providers/[^/]+/models$`),
}

// RejectSupportSessionWrite writes a 403 response and returns true when
// the request is a write inside an active support session.
func RejectSupportSessionWrite(w http.ResponseWriter, r *http.Request, info *SupportSessionInfo) bool {
	if info == nil || !info.Active {
		return false
	}

	if slices.Contains(supportSessionReadMethods, r.Method) {
		return false
	}

	if slices.ContainsFunc(supportSessionAllowedWrites, func(pattern *regexp.Regexp) bool {
		return pattern.MatchString(r.URL.Path)
	}) {
		return false
	}

	http.Error(w, SupportSessionReadOnlyMessage, http.StatusForbidden)
	return true
}

// IsSupportSessionActive reports whether the request is being served
// inside a fully validated support session. It relies on the context
// rather than re-parsing the cookie: the auth middleware has already
// verified that the cookie is a real support session token issued to
// *this* admin, that the admin is still an installation admin, and that
// the session is fresh relative to password rotation. A stale or
// unrelated cookie that happens to be structurally valid will not
// produce an Active info here.
func IsSupportSessionActive(r *http.Request) bool {
	info, ok := GetSupportSessionFromContext(r.Context())
	return ok && info != nil && info.Active
}
