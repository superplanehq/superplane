package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers"
	"github.com/superplanehq/superplane/test/support"
)

// The gateway owns transport authentication and privacy. Model lifecycle tests
// cannot detect a route registered outside the admin authentication boundary.
func TestOrganizationSuspensionGateway(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	server.InitRouter()
	registerTestGRPCGateway(t, server, resource.AuthService, resource.Registry, resource.Encryptor, support.NewOIDCProvider())
	db := database.DB(t.Context())
	rawToken := uuid.NewString()
	require.NoError(t, models.CreateUserAPIToken(db, models.NewUserAPIToken(resource.User, "Suspension test", crypto.HashToken(rawToken))))
	base := "/admin/api/organizations/" + resource.Organization.ID.String()
	request := func(method, path, body, token string) int {
		response := execRequest(server, requestParams{method: method, path: path, body: []byte(body), contentType: "application/json", authCookie: cookie, authToken: token})
		return response.Code
	}
	require.Equal(t, http.StatusBadRequest, request("POST", base+"/suspend", `{"reason":"  "}`, ""))
	require.Equal(t, http.StatusBadRequest, request("POST", "/admin/api/organizations/invalid/suspend", `{"reason":"abuse"}`, ""))
	require.Equal(t, http.StatusNotFound, request("GET", "/admin/api/organizations/"+uuid.NewString()+"/suspension-history", "", ""))
	require.Equal(t, http.StatusOK, request("POST", base+"/suspend", `{"reason":"  Private abuse reason  "}`, ""))
	require.Equal(t, http.StatusOK, request("POST", base+"/suspend", `{"reason":"Replacement"}`, rawToken))
	events, err := models.ListOrganizationSuspensionEvents(db, resource.Organization.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "Private abuse reason", events[0].Reason)
	require.Equal(t, resource.Account.ID, events[0].AccountID)
	require.Equal(t, http.StatusConflict, request("POST", base+"/reactivate", `{}`, rawToken))
	for _, token := range []string{"", rawToken} {
		member := execRequest(server, requestParams{method: "GET", path: "/api/v1/organizations/" + resource.Organization.ID.String(), authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()}})
		require.Equal(t, http.StatusOK, member.Code, member.Body.String())
		require.NotContains(t, member.Body.String(), events[0].Reason)
		require.NotContains(t, member.Body.String(), "accountId")
		var response struct {
			Organization struct {
				Suspended bool `json:"suspended"`
			} `json:"organization"`
		}
		require.NoError(t, json.Unmarshal(member.Body.Bytes(), &response))
		require.True(t, response.Organization.Suspended)
		write := execRequest(server, requestParams{method: "PATCH", path: "/api/v1/organizations/" + resource.Organization.ID.String(), body: []byte(`{"organization":{"metadata":{"name":"Changed"}}}`), contentType: "application/json", authCookie: cookie, authToken: token, headers: map[string]string{"x-organization-id": resource.Organization.ID.String()}})
		require.Equal(t, http.StatusForbidden, write.Code, write.Body.String())
	}
	require.NoError(t, workers.NewOrganizationSuspensionWorker().Process(t.Context()))
	require.Equal(t, http.StatusOK, request("POST", base+"/reactivate", `{}`, rawToken))
	require.Equal(t, http.StatusOK, request("POST", base+"/reactivate", `{}`, ""))
	events, err = models.ListOrganizationSuspensionEvents(db, resource.Organization.ID)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, models.OrganizationSuspensionActionReactivate, events[0].Action)
	require.NotNil(t, events[1].CancellationCompletedAt)
	// The same authenticated member loses private admin access without losing reads.
	require.NoError(t, db.Model(resource.Account).Update("installation_admin", false).Error)
	for _, suffix := range []string{"suspend", "reactivate", "suspension-history"} {
		method := "POST"
		if suffix == "suspension-history" {
			method = "GET"
		}
		for _, token := range []string{"", rawToken} {
			require.Equal(t, http.StatusNotFound, request(method, base+"/"+suffix, `{"reason":"abuse"}`, token))
		}
	}
}

func TestSuspensionAdminStatusFilterIncludesPinnedMatches(t *testing.T) {
	server, resource, cookie := setupAdminTestServer(t)
	db := database.DB(t.Context())
	require.NoError(t, resource.Organization.Suspend(db, resource.Account.ID, "Filter target"))
	require.NoError(t, models.PinOrganizationForAccount(db, resource.Account.ID, resource.Organization.ID))
	for _, status := range []string{"active", "suspended", "all"} {
		response := execRequest(server, requestParams{method: "GET", path: "/admin/api/organizations?status=" + status + "&search=" + resource.Organization.ID.String() + "&limit=1&offset=0&sort_by=name&sort_direction=asc", authCookie: cookie})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var page adminOrganizationListResponse
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
		require.Empty(t, page.Items)
		require.Zero(t, page.Total)
		if status == "active" {
			require.Empty(t, page.Pinned)
			require.Zero(t, page.MatchTotal)
			continue
		}
		require.EqualValues(t, 1, page.MatchTotal)
		require.Len(t, page.Pinned, 1)
		require.True(t, page.Pinned[0].Suspended)
	}
}
