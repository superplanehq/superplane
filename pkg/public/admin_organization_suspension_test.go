package public

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminOrganizationSuspension(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	path := "/admin/api/organizations/" + r.Organization.ID.String()
	// A regular organization member cannot use installation-admin controls.
	require.NoError(t, models.DemoteFromInstallationAdmin(r.Account.ID.String()))
	response := execRequest(server, requestParams{method: http.MethodPost, path: path + "/suspend", authCookie: token, body: []byte(`{"reason":"Abuse"}`)})
	require.Equal(t, http.StatusNotFound, response.Code)
	require.NoError(t, models.PromoteToInstallationAdmin(r.Account.ID.String()))
	response = execRequest(server, requestParams{method: http.MethodPost, path: path + "/suspend", authCookie: token, body: []byte(`{"reason":" "}`)})
	require.Equal(t, http.StatusBadRequest, response.Code)
	response = execRequest(server, requestParams{method: http.MethodPost, path: path + "/suspend", authCookie: token, body: []byte(`{"reason":"Unattended automation"}`)})
	require.Equal(t, http.StatusOK, response.Code)
	detail := getAdminOrganization(t, server, token, r.Organization.ID)
	require.NotNil(t, detail.SuspendedAt)
	require.Equal(t, "Unattended automation", detail.SuspensionReason)
	response = execRequest(server, requestParams{method: http.MethodGet, path: "/admin/api/organizations?status=suspended", authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	var filtered adminOrganizationListResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &filtered))
	require.Len(t, filtered.Items, 1)
	require.Equal(t, r.Organization.ID.String(), filtered.Items[0].ID)

	response = execRequest(server, requestParams{method: http.MethodPost, path: path + "/reactivate", authCookie: token})
	require.Equal(t, http.StatusConflict, response.Code)
	events, err := models.ListOrganizationSuspensionEvents(database.DB(t.Context()), r.Organization.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.NoError(t, r.Organization.CompleteSuspensionCancellation(database.DB(t.Context()), events[0].ID))
	response = execRequest(server, requestParams{method: http.MethodPost, path: path + "/reactivate", authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	require.Nil(t, getAdminOrganization(t, server, token, r.Organization.ID).SuspendedAt)
	response = execRequest(server, requestParams{method: http.MethodGet, path: path + "/suspension-history", authCookie: token})
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"action":"reactivate"`)
	require.Contains(t, response.Body.String(), r.Account.ID.String())
}
