package authentication

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/utils"
)

type ssoOrgResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LoginURL string `json:"login_url"`
}

type ssoLookupResponse struct {
	Orgs []ssoOrgResult `json:"orgs"`
}

// handleSSOLookup is a public (no auth required) endpoint.
// GET /auth/sso/lookup?email=user@company.com
// Returns the list of orgs that have Okta SAML enabled for the given email.
// Always returns HTTP 200 with an empty list on any failure — never reveals
// whether an email exists.
func (a *Handler) handleSSOLookup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	email := utils.NormalizeEmail(strings.TrimSpace(r.URL.Query().Get("email")))
	if email == "" {
		json.NewEncoder(w).Encode(ssoLookupResponse{Orgs: []ssoOrgResult{}})
		return
	}

	rows, err := models.FindOktaOrgsForEmail(database.Conn(), email)
	if err != nil {
		json.NewEncoder(w).Encode(ssoLookupResponse{Orgs: []ssoOrgResult{}})
		return
	}

	orgs := make([]ssoOrgResult, 0, len(rows))
	for _, row := range rows {
		loginURL := ""
		if a.publicAppBaseURL != "" {
			loginURL = a.publicAppBaseURL + "/auth/okta/" + row.OrgID + "/saml/login"
		}
		orgs = append(orgs, ssoOrgResult{
			ID:       row.OrgID,
			Name:     row.OrgName,
			LoginURL: loginURL,
		})
	}

	json.NewEncoder(w).Encode(ssoLookupResponse{Orgs: orgs})
}
