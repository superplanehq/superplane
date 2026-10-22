package public

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/entitlements"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
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
		{"GET", "/admin/api/feature-definitions/byo_key/organizations", ""},
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

type adminCatalogRequest func(method, path, body string) *httptest.ResponseRecorder

func newAdminCatalogRequest(server *Server, token string) adminCatalogRequest {
	return func(method, path, body string) *httptest.ResponseRecorder {
		return execRequest(server, requestParams{method: method, path: "/admin/api/" + path, authCookie: token, body: []byte(body), contentType: "application/json"})
	}
}

func createAdminTestFeature(t *testing.T, request adminCatalogRequest, body string) {
	response := request("POST", "feature-definitions", body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
}

func createAdminTestPlan(t *testing.T, request adminCatalogRequest, body string) models.BillingPlan {
	response := request("POST", "billing-plans", body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var plan models.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	return plan
}

func catalogFeatureKeys(t *testing.T, request adminCatalogRequest) []string {
	response := request("GET", "feature-definitions", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var features []models.PlanFeatureDefinition
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &features))
	keys := []string{}
	for _, feature := range features {
		keys = append(keys, feature.Key)
	}
	return keys
}

// cleanupFeatureReferences removes rows that the tests create. The truncate
// helper does not clear billing plans, feature definitions, or feature grants.
func cleanupFeatureReferences(t *testing.T, organizationID uuid.UUID, planIDs []uuid.UUID, featureKeys []string) {
	t.Cleanup(func() {
		db := database.Conn()
		require.NoError(t, db.Where("organization_id = ?", organizationID).Delete(&models.OrganizationBillingPlan{}).Error)
		require.NoError(t, db.Where("feature_key IN ?", featureKeys).Delete(&models.OrganizationFeatureGrant{}).Error)
		require.NoError(t, db.Where("id IN ?", planIDs).Delete(&models.BillingPlan{}).Error)
		require.NoError(t, db.Where("key IN ?", featureKeys).Delete(&models.PlanFeatureDefinition{}).Error)
	})
}

func TestAdminDeleteFeatureDefinitionRemovesAllReferences(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	createAdminTestFeature(t, request, `{"key":"doomed_reports","label":"Doomed reports"}`)
	createAdminTestFeature(t, request, `{"key":"kept_exports","label":"Kept exports"}`)
	createAdminTestFeature(t, request, `{"key":"open_banner","label":"Open banner","gated":false}`)
	assigned := createAdminTestPlan(t, request, `{"key":"doomed_assigned","name":"Assigned","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","features":["doomed_reports","kept_exports","open_banner"]}`)
	polar := createAdminTestPlan(t, request, `{"key":"doomed_polar","name":"Polar","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","polar_product_id":"polar_doomed_reports","features":["doomed_reports"]}`)
	unassigned := createAdminTestPlan(t, request, `{"key":"doomed_unassigned","name":"Unassigned","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","features":["doomed_reports"]}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{assigned.ID, polar.ID, unassigned.ID}, []string{"doomed_reports", "kept_exports", "open_banner"})

	var systemPlan models.BillingPlan
	require.NoError(t, db.Where("key = ?", "none").First(&systemPlan).Error)
	originalSystemFeatures, err := json.Marshal(systemPlan.Features)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE billing_plans SET features = ?::jsonb WHERE id = ?`, `["doomed_reports","kept_exports"]`, systemPlan.ID).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(`UPDATE billing_plans SET features = ?::jsonb WHERE id = ?`, string(originalSystemFeatures), systemPlan.ID).Error)
	})

	response := request("PUT", orgPath+"/billing-plan", `{"plan":"doomed_assigned"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = request("PUT", orgPath+"/features", `{"feature_key":"doomed_reports","enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = request("PUT", orgPath+"/features", `{"feature_key":"kept_exports","enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	other := support.CreateOrganization(t, r, r.User)
	require.NoError(t, models.SetOrganizationFeatureGrant(db, models.OrganizationFeatureGrant{OrganizationID: other.ID, FeatureKey: "doomed_reports", Enabled: false}))
	t.Cleanup(func() {
		require.NoError(t, db.Where("organization_id = ?", other.ID).Delete(&models.OrganizationFeatureGrant{}).Error)
	})

	resolved, err := entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.True(t, resolved.Features["doomed_reports"])

	response = request("DELETE", "feature-definitions/doomed_reports", "")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.NotContains(t, catalogFeatureKeys(t, request), "doomed_reports")
	require.Contains(t, catalogFeatureKeys(t, request), "kept_exports")
	require.Contains(t, catalogFeatureKeys(t, request), "open_banner")

	assertPlanFeatures := func(planID uuid.UUID, expected []string) {
		var plan models.BillingPlan
		require.NoError(t, db.First(&plan, "id = ?", planID).Error)
		require.ElementsMatch(t, expected, plan.Features)
	}
	assertPlanFeatures(assigned.ID, []string{"kept_exports", "open_banner"})
	assertPlanFeatures(polar.ID, []string{})
	assertPlanFeatures(unassigned.ID, []string{})
	assertPlanFeatures(systemPlan.ID, []string{"kept_exports"})
	var polarPlan models.BillingPlan
	require.NoError(t, db.First(&polarPlan, "id = ?", polar.ID).Error)
	require.NotNil(t, polarPlan.PolarProductID)
	require.Equal(t, "polar_doomed_reports", *polarPlan.PolarProductID)

	var remainingGrants []models.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "doomed_reports").Find(&remainingGrants).Error)
	require.Empty(t, remainingGrants)
	var keptGrants []models.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "kept_exports").Find(&keptGrants).Error)
	require.Len(t, keptGrants, 1)
	require.True(t, keptGrants[0].Enabled)

	assignment, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, assigned.ID, assignment.PlanID)

	resolved, err = entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.NotContains(t, resolved.Features, "doomed_reports")
	require.True(t, resolved.Features["kept_exports"])
	require.True(t, resolved.Features["open_banner"])

	response = request("DELETE", "feature-definitions/doomed_reports", "")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
	response = request("DELETE", "feature-definitions/never_existed", "")
	require.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
}

func TestAdminRecreatedFeatureDoesNotInheritDeletedReferences(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	createAdminTestFeature(t, request, `{"key":"reborn_reports","label":"Reborn reports"}`)
	plan := createAdminTestPlan(t, request, `{"key":"reborn_plan","name":"Reborn","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","features":["reborn_reports"]}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{plan.ID}, []string{"reborn_reports"})
	response := request("PUT", orgPath+"/billing-plan", `{"plan":"reborn_plan"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = request("PUT", orgPath+"/features", `{"feature_key":"reborn_reports","enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	response = request("DELETE", "feature-definitions/reborn_reports", "")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	createAdminTestFeature(t, request, `{"key":"reborn_reports","label":"Reborn reports","gated":true}`)

	resolved, err := entitlements.Resolve(db, r.Organization.ID)
	require.NoError(t, err)
	require.False(t, resolved.Features["reborn_reports"])
	var grants []models.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "reborn_reports").Find(&grants).Error)
	require.Empty(t, grants)
	var stored models.BillingPlan
	require.NoError(t, db.First(&stored, "id = ?", plan.ID).Error)
	require.NotContains(t, stored.Features, "reborn_reports")
}

func TestAdminDeleteFeatureDefinitionRequiresInstallationAdmin(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()
	request := newAdminCatalogRequest(server, token)
	createAdminTestFeature(t, request, `{"key":"guarded_reports","label":"Guarded reports"}`)
	cleanupFeatureReferences(t, r.Organization.ID, nil, []string{"guarded_reports"})
	response := request("PUT", "organizations/"+r.Organization.ID.String()+"/features", `{"feature_key":"guarded_reports","enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	regular, err := models.CreateAccount("Feature Member", "feature-member@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	response = execRequest(server, requestParams{method: "DELETE", path: "/admin/api/feature-definitions/guarded_reports", authCookie: regularToken})
	require.Equal(t, http.StatusNotFound, response.Code)
	response = execRequest(server, requestParams{method: "DELETE", path: "/admin/api/feature-definitions/guarded_reports"})
	require.NotEqual(t, http.StatusNoContent, response.Code)

	require.Contains(t, catalogFeatureKeys(t, request), "guarded_reports")
	var grants []models.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "guarded_reports").Find(&grants).Error)
	require.Len(t, grants, 1)
}

func TestAdminDeleteFeatureDefinitionRollsBackWhenDefinitionDeleteFails(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	createAdminTestFeature(t, request, `{"key":"rollback_reports","label":"Rollback reports"}`)
	plan := createAdminTestPlan(t, request, `{"key":"rollback_plan","name":"Rollback","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","features":["rollback_reports"]}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{plan.ID}, []string{"rollback_reports"})
	response := request("PUT", orgPath+"/billing-plan", `{"plan":"rollback_plan"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = request("PUT", orgPath+"/features", `{"feature_key":"rollback_reports","enabled":true}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	// The database rejects the final definition delete, after the plan and grant cleanup has run.
	require.NoError(t, db.Exec(`CREATE OR REPLACE FUNCTION fail_feature_definition_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'feature definition delete blocked'; END $$`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_feature_definition_delete BEFORE DELETE ON plan_feature_definitions FOR EACH ROW WHEN (OLD.key = 'rollback_reports') EXECUTE FUNCTION fail_feature_definition_delete()`).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(`DROP TRIGGER IF EXISTS fail_feature_definition_delete ON plan_feature_definitions`).Error)
		require.NoError(t, db.Exec(`DROP FUNCTION IF EXISTS fail_feature_definition_delete()`).Error)
	})

	response = request("DELETE", "feature-definitions/rollback_reports", "")
	require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())

	require.Contains(t, catalogFeatureKeys(t, request), "rollback_reports")
	var stored models.BillingPlan
	require.NoError(t, db.First(&stored, "id = ?", plan.ID).Error)
	require.Equal(t, []string{"rollback_reports"}, stored.Features)
	var grants []models.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "rollback_reports").Find(&grants).Error)
	require.Len(t, grants, 1)
	require.True(t, grants[0].Enabled)
	assignment, err := models.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, assignment.PlanID)
}
