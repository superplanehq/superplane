package public

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"net/http"
	"net/http/httptest"
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
		{"POST", "/admin/api/feature-definitions", "{}"},
		{"PUT", "/admin/api/feature-definitions/custom_feature", "{}"},
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

func TestAdminCustomFeaturePlanAccess(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		return execRequest(server, requestParams{method: method, path: "/admin/api/" + path, authCookie: token, body: []byte(body), contentType: "application/json"})
	}
	response := request("POST", "feature-definitions", `{"key":"custom_reports","label":"Custom reports"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var feature struct {
		Key   string
		Gated bool
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &feature))
	require.True(t, feature.Gated)
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("key = ?", "custom_reports").Delete(&models.PlanFeatureDefinition{}).Error)
	})
	response = request("POST", "feature-definitions", `{"key":"custom_reports","label":"Duplicate"}`)
	require.Equal(t, http.StatusBadRequest, response.Code)
	response = request("POST", "feature-definitions", `{"key":"Invalid key","label":"Invalid"}`)
	require.Equal(t, http.StatusBadRequest, response.Code)
	response = request("GET", "feature-definitions", "")
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), `"custom_reports"`)
	enabled, err := entitlements.HasFeature(database.Conn(), r.Organization.ID, "custom_reports")
	require.NoError(t, err)
	require.False(t, enabled)
	response = request("POST", "billing-plans", `{"key":"custom_reports_plan","name":"Reports","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","features":["custom_reports"]}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var plan models.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&models.OrganizationBillingPlan{}).Error)
		require.NoError(t, models.DeleteBillingPlan(database.Conn(), plan.ID))
	})
	response = request("PUT", "organizations/"+r.Organization.ID.String()+"/billing-plan", `{"plan":"custom_reports_plan"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	enabled, err = entitlements.HasFeature(database.Conn(), r.Organization.ID, "custom_reports")
	require.NoError(t, err)
	require.True(t, enabled)
	enabled, err = entitlements.HasFeature(database.Conn(), r.Organization.ID, "unknown_feature")
	require.NoError(t, err)
	require.False(t, enabled)
	plan.Features = []string{}
	body, err := json.Marshal(plan)
	require.NoError(t, err)
	response = request("PUT", "billing-plans/"+plan.ID.String(), string(body))
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	enabled, err = entitlements.HasFeature(database.Conn(), r.Organization.ID, "custom_reports")
	require.NoError(t, err)
	require.False(t, enabled)
	response = request("PUT", "feature-definitions/custom_reports", `{"key":"renamed","label":"Reports updated","gated":true}`)
	require.Equal(t, http.StatusBadRequest, response.Code)
	response = request("PUT", "feature-definitions/custom_reports", `{"key":"custom_reports","label":"Reports updated","gated":false}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	enabled, err = entitlements.HasFeature(database.Conn(), r.Organization.ID, "custom_reports")
	require.NoError(t, err)
	require.True(t, enabled)
}
