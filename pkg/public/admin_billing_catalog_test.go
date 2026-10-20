package public

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"net/http"
	"testing"
	"time"
)

func TestAdminCatalogAuthorizationAndAssignment(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	regular, err := models.CreateAccount("Member", "catalog-member@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	for _, route := range []struct{ method, path, body string }{
		{"GET", "/admin/api/billing-plans", ""}, {"POST", "/admin/api/billing-plans", "{}"},
		{"PUT", "/admin/api/billing-plans/00000000-0000-0000-0000-000000000000", "{}"},
		{"DELETE", "/admin/api/billing-plans/00000000-0000-0000-0000-000000000000", ""},
		{"GET", "/admin/api/feature-definitions", ""},
		{"PUT", "/admin/api/organizations/" + r.Organization.ID.String() + "/features", "{}"},
	} {
		response := execRequest(server, requestParams{method: route.method, path: route.path, authCookie: regularToken, body: []byte(route.body), contentType: "application/json"})
		require.Equal(t, http.StatusNotFound, response.Code)
	}
	response := execRequest(server, requestParams{method: "POST", path: "/admin/api/billing-plans", authCookie: token, body: []byte(`{"key":"admin_catalog_test","name":"Custom","category":"paid","visibility":"private","credit_frequency":"once","credit_expiration":"never","credit_amount_cents":1000}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var plan models.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	path := "/admin/api/organizations/" + r.Organization.ID.String()
	response = execRequest(server, requestParams{method: "PUT", path: path + "/billing-plan", authCookie: token, body: []byte(`{"plan":"admin_catalog_test"}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: "PUT", path: path + "/features", authCookie: token, body: []byte(`{"feature_key":"byo_key","enabled":true}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: "DELETE", path: "/admin/api/billing-plans/" + plan.ID.String(), authCookie: token})
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBillingPlan{}).Error)
	require.NoError(t, models.DeleteBillingPlan(database.Conn(), plan.ID))
}
