package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

var suspensionAllowedWrites = []*regexp.Regexp{
	regexp.MustCompile(`^/api/v1/canvases/[^/]+/node-executions/[^/]+/runner-live-logs/session$`),
	regexp.MustCompile(`^/api/v1/canvases/[^/]+/(runs|executions)/[^/]+/cancel$`),
	regexp.MustCompile(`^/api/v1/agents/chats/[^/]+/interrupt$`),
	regexp.MustCompile(`^/api/v1/factories/[^/]+/scheduled-intake/preview$`),
}

func rejectSuspendedOrganizationWrite(w http.ResponseWriter, r *http.Request, user *models.User) bool {
	if slices.Contains([]string{http.MethodGet, http.MethodHead, http.MethodOptions}, r.Method) {
		return false
	}
	if (r.Method == http.MethodPost || r.Method == http.MethodPatch) && slices.ContainsFunc(suspensionAllowedWrites, func(pattern *regexp.Regexp) bool { return pattern.MatchString(r.URL.Path) }) {
		return false
	}
	err := models.RequireOrganizationActive(database.DB(r.Context()), user.OrganizationID)
	if err == nil {
		return false
	}
	if errors.Is(err, models.ErrOrganizationSuspended) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": "ORGANIZATION_SUSPENDED", "message": models.OrganizationSuspendedMessage})
		return true
	}
	http.Error(w, "Could not load organization status.", http.StatusInternalServerError)
	return true
}
