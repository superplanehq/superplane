package billinghttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	sentrytest "github.com/superplanehq/superplane/test/support/sentry"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/ee/billing/entitlements"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestAdminCatalogAuthorizationAndAssignment(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	regular, err := models.CreateAccount("Member", "catalog-member@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	for _, route := range []struct{ method, path, body string }{
		{"GET", "/admin/api/polar/products", ""},
		{"GET", "/admin/api/credit-packs", ""}, {"POST", "/admin/api/credit-packs", "{}"},
		{"PUT", "/admin/api/credit-packs/00000000-0000-0000-0000-000000000000", "{}"},
		{"DELETE", "/admin/api/credit-packs/00000000-0000-0000-0000-000000000000", ""},
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
	var plan billingmodels.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	path := "/admin/api/organizations/" + r.Organization.ID.String()
	response = execRequest(server, requestParams{method: "PUT", path: path + "/billing-plan", authCookie: token, body: []byte(`{"plan":"admin_catalog_test"}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: "PUT", path: path + "/features", authCookie: token, body: []byte(`{"feature_key":"byo_key","enabled":true}`), contentType: "application/json"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	response = execRequest(server, requestParams{method: "DELETE", path: "/admin/api/billing-plans/" + plan.ID.String(), authCookie: token})
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
	require.NoError(t, billingmodels.DeleteBillingPlan(database.Conn(), plan.ID))
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
		require.NoError(t, database.Conn().Where("key = ?", "custom_reports").Delete(&billingmodels.PlanFeatureDefinition{}).Error)
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
	var plan billingmodels.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("organization_id = ?", r.Organization.ID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, billingmodels.DeleteBillingPlan(database.Conn(), plan.ID))
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

func newAdminCatalogRequest(server *testServer, token string) adminCatalogRequest {
	return func(method, path, body string) *httptest.ResponseRecorder {
		return execRequest(server, requestParams{method: method, path: "/admin/api/" + path, authCookie: token, body: []byte(body), contentType: "application/json"})
	}
}

func createAdminTestFeature(t *testing.T, request adminCatalogRequest, body string) {
	response := request("POST", "feature-definitions", body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
}

func createAdminTestPlan(t *testing.T, request adminCatalogRequest, body string) billingmodels.BillingPlan {
	response := request("POST", "billing-plans", body)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var plan billingmodels.BillingPlan
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &plan))
	return plan
}

func catalogFeatureKeys(t *testing.T, request adminCatalogRequest) []string {
	response := request("GET", "feature-definitions", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var features []billingmodels.PlanFeatureDefinition
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
		require.NoError(t, db.Where("organization_id = ?", organizationID).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, db.Where("feature_key IN ?", featureKeys).Delete(&billingmodels.OrganizationFeatureGrant{}).Error)
		require.NoError(t, db.Where("id IN ?", planIDs).Delete(&billingmodels.BillingPlan{}).Error)
		require.NoError(t, db.Where("key IN ?", featureKeys).Delete(&billingmodels.PlanFeatureDefinition{}).Error)
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

	var systemPlan billingmodels.BillingPlan
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
	require.NoError(t, billingmodels.SetOrganizationFeatureGrant(db, billingmodels.OrganizationFeatureGrant{OrganizationID: other.ID, FeatureKey: "doomed_reports", Enabled: false}))
	t.Cleanup(func() {
		require.NoError(t, db.Where("organization_id = ?", other.ID).Delete(&billingmodels.OrganizationFeatureGrant{}).Error)
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
		var plan billingmodels.BillingPlan
		require.NoError(t, db.First(&plan, "id = ?", planID).Error)
		require.ElementsMatch(t, expected, plan.Features)
	}
	assertPlanFeatures(assigned.ID, []string{"kept_exports", "open_banner"})
	assertPlanFeatures(polar.ID, []string{})
	assertPlanFeatures(unassigned.ID, []string{})
	assertPlanFeatures(systemPlan.ID, []string{"kept_exports"})
	var polarPlan billingmodels.BillingPlan
	require.NoError(t, db.First(&polarPlan, "id = ?", polar.ID).Error)
	require.NotNil(t, polarPlan.PolarProductID)
	require.Equal(t, "polar_doomed_reports", *polarPlan.PolarProductID)

	var remainingGrants []billingmodels.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "doomed_reports").Find(&remainingGrants).Error)
	require.Empty(t, remainingGrants)
	var keptGrants []billingmodels.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "kept_exports").Find(&keptGrants).Error)
	require.Len(t, keptGrants, 1)
	require.True(t, keptGrants[0].Enabled)

	assignment, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
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
	var grants []billingmodels.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "reborn_reports").Find(&grants).Error)
	require.Empty(t, grants)
	var stored billingmodels.BillingPlan
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
	var grants []billingmodels.OrganizationFeatureGrant
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
	var stored billingmodels.BillingPlan
	require.NoError(t, db.First(&stored, "id = ?", plan.ID).Error)
	require.Equal(t, []string{"rollback_reports"}, stored.Features)
	var grants []billingmodels.OrganizationFeatureGrant
	require.NoError(t, db.Where("feature_key = ?", "rollback_reports").Find(&grants).Error)
	require.Len(t, grants, 1)
	require.True(t, grants[0].Enabled)
	assignment, err := billingmodels.FindOrganizationBillingPlan(db, r.Organization.ID)
	require.NoError(t, err)
	require.Equal(t, plan.ID, assignment.PlanID)
}

type billingPlanDeleteReasonRow struct {
	Key                  string
	DeleteDisabledReason *string `json:"delete_disabled_reason"`
}

// listBillingPlanDeleteReasons returns the list response keyed by plan key. It
// also checks that every item keeps the editable fields and that the reason
// is present as either a string or an explicit JSON null.
func listBillingPlanDeleteReasons(t *testing.T, request adminCatalogRequest) map[string]*string {
	response := request("GET", "billing-plans", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var raw []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &raw))
	reasons := map[string]*string{}
	for _, item := range raw {
		for _, field := range []string{"id", "key", "name", "features", "polar_product_id", "credit_expiration"} {
			require.Contains(t, item, field)
		}
		require.Contains(t, item, "delete_disabled_reason")
	}
	var items []billingPlanDeleteReasonRow
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
	for _, item := range items {
		reasons[item.Key] = item.DeleteDisabledReason
	}
	return reasons
}

func requireDeleteReason(t *testing.T, reasons map[string]*string, key string, expected string) {
	t.Helper()
	require.NotNil(t, reasons[key], "plan %s should be blocked", key)
	require.Equal(t, expected, *reasons[key], "plan %s", key)
}

func TestAdminBillingPlanListDeleteDisabledReasonPriority(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	polar := createAdminTestPlan(t, request, `{"key":"reason_polar","name":"Polar","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never","polar_product_id":"polar_reason"}`)
	assigned := createAdminTestPlan(t, request, `{"key":"reason_assigned","name":"Assigned","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	free := createAdminTestPlan(t, request, `{"key":"reason_free","name":"Free","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{polar.ID, assigned.ID, free.ID}, nil)

	assignPlan := func(key string) {
		response := request("PUT", orgPath+"/billing-plan", `{"plan":"`+key+`"}`)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	}
	systemReason := billingmodels.BillingPlanDeleteReasonSystem
	polarReason := billingmodels.BillingPlanDeleteReasonPolar
	assignedReason := billingmodels.BillingPlanDeleteReasonAssigned

	reasons := listBillingPlanDeleteReasons(t, request)
	for _, key := range []string{"none", "trial", "business_legacy", "business_manual"} {
		requireDeleteReason(t, reasons, key, systemReason)
	}
	requireDeleteReason(t, reasons, polar.Key, polarReason)
	require.Nil(t, reasons[assigned.Key])
	require.Nil(t, reasons[free.Key])

	assignPlan(assigned.Key)
	reasons = listBillingPlanDeleteReasons(t, request)
	requireDeleteReason(t, reasons, assigned.Key, assignedReason)
	require.Nil(t, reasons[free.Key])

	assignPlan(polar.Key)
	reasons = listBillingPlanDeleteReasons(t, request)
	requireDeleteReason(t, reasons, polar.Key, polarReason)
	require.Nil(t, reasons[assigned.Key])

	assignPlan("business_manual")
	reasons = listBillingPlanDeleteReasons(t, request)
	requireDeleteReason(t, reasons, "business_manual", systemReason)
	requireDeleteReason(t, reasons, polar.Key, polarReason)
	require.Nil(t, reasons[free.Key])
}

func TestAdminBillingPlanListReportsAssignment(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	assigned := createAdminTestPlan(t, request, `{"key":"flag_assigned","name":"Assigned","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	unassigned := createAdminTestPlan(t, request, `{"key":"flag_unassigned","name":"Unassigned","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{assigned.ID, unassigned.ID}, nil)

	response := request("PUT", orgPath+"/billing-plan", `{"plan":"`+assigned.Key+`"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	response = request("GET", "billing-plans", "")
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var items []struct {
		Key      string `json:"key"`
		Assigned *bool  `json:"assigned"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
	flags := map[string]*bool{}
	for _, item := range items {
		flags[item.Key] = item.Assigned
	}
	require.NotNil(t, flags[assigned.Key])
	require.True(t, *flags[assigned.Key])
	require.NotNil(t, flags[unassigned.Key])
	require.False(t, *flags[unassigned.Key])
}

func TestAdminBillingPlanListLooksUpAssignmentsOnce(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	first := createAdminTestPlan(t, request, `{"key":"lookup_first","name":"First","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	second := createAdminTestPlan(t, request, `{"key":"lookup_second","name":"Second","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{first.ID, second.ID}, nil)

	db := database.Conn()
	lookups := 0
	const callbackName = "test:count_organization_billing_plan_lookups"
	require.NoError(t, db.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "organization_billing_plans" {
			lookups++
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callbackName)) })

	listBillingPlanDeleteReasons(t, request)
	require.Equal(t, 1, lookups)
}

func TestAdminBillingPlanDeletionFollowsAssignmentLifecycle(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	plan := createAdminTestPlan(t, request, `{"key":"lifecycle_plan","name":"Lifecycle","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	replacement := createAdminTestPlan(t, request, `{"key":"lifecycle_replacement","name":"Replacement","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{plan.ID, replacement.ID}, nil)

	require.Nil(t, listBillingPlanDeleteReasons(t, request)[plan.Key])

	response := request("PUT", orgPath+"/billing-plan", `{"plan":"lifecycle_plan"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	// The client may still hold the null reason from before the assignment.
	response = request("DELETE", "billing-plans/"+plan.ID.String(), "")
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), billingmodels.BillingPlanDeleteReasonAssigned)
	requireDeleteReason(t, listBillingPlanDeleteReasons(t, request), plan.Key, billingmodels.BillingPlanDeleteReasonAssigned)

	response = request("PUT", orgPath+"/billing-plan", `{"plan":"lifecycle_replacement"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Nil(t, listBillingPlanDeleteReasons(t, request)[plan.Key])

	response = request("DELETE", "billing-plans/"+plan.ID.String(), "")
	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
	require.NotContains(t, listBillingPlanDeleteReasons(t, request), plan.Key)
}

func TestAdminBillingPlanInactiveSubscriptionAssignmentBlocksDeletion(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	db := database.Conn()
	request := newAdminCatalogRequest(server, token)
	orgPath := "organizations/" + r.Organization.ID.String()
	plan := createAdminTestPlan(t, request, `{"key":"inactive_plan","name":"Inactive","category":"paid","visibility":"private","credit_frequency":"none","credit_expiration":"never"}`)
	cleanupFeatureReferences(t, r.Organization.ID, []uuid.UUID{plan.ID}, nil)
	response := request("PUT", orgPath+"/billing-plan", `{"plan":"inactive_plan"}`)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	ended := time.Now().Add(-24 * time.Hour)
	require.NoError(t, db.Model(&billingmodels.OrganizationBillingPlan{}).
		Where("organization_id = ?", r.Organization.ID).
		Updates(map[string]any{"polar_subscription_status": "canceled", "current_period_end": ended}).Error)

	requireDeleteReason(t, listBillingPlanDeleteReasons(t, request), plan.Key, billingmodels.BillingPlanDeleteReasonAssigned)
	response = request("DELETE", "billing-plans/"+plan.ID.String(), "")
	require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), billingmodels.BillingPlanDeleteReasonAssigned)
}

// requireBillingPlanJSONError checks a billing-plan error response and returns its message.
func requireBillingPlanJSONError(t *testing.T, response *httptest.ResponseRecorder, status int) string {
	t.Helper()
	require.Equal(t, status, response.Code, response.Body.String())
	require.Equal(t, "application/json", response.Header().Get("Content-Type"))
	var body struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body), response.Body.String())
	require.NotEmpty(t, body.Message)
	return body.Message
}

// billingPlanBody returns a valid plan payload. Fields in overrides replace the defaults.
func billingPlanBody(t *testing.T, overrides map[string]any) string {
	t.Helper()
	fields := map[string]any{"name": "Plan", "category": "paid", "visibility": "private", "credit_frequency": "none", "credit_expiration": "never"}
	maps.Copy(fields, overrides)
	body, err := json.Marshal(fields)
	require.NoError(t, err)
	return string(body)
}

// captureAdminErrorLogs returns the buffer that receives the request logger output.
func captureAdminErrorLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	buffer := &bytes.Buffer{}
	logger := log.StandardLogger()
	previous := logger.Out
	logger.SetOutput(buffer)
	t.Cleanup(func() { logger.SetOutput(previous) })
	return buffer
}

// failBillingPlanWrite makes a trigger raise an error for one plan key. The
// database returns the detail, so the test can check it stays out of responses.
func failBillingPlanWrite(t *testing.T, name, event, rowCondition string) {
	t.Helper()
	db := database.Conn()
	require.NoError(t, db.Exec(`CREATE OR REPLACE FUNCTION fail_billing_plan_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'billing plan storage detail'; END $$`).Error)
	require.NoError(t, db.Exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE %s ON billing_plans FOR EACH ROW WHEN (%s) EXECUTE FUNCTION fail_billing_plan_write()`, name, event, rowCondition)).Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON billing_plans`, name)).Error)
		require.NoError(t, db.Exec(`DROP FUNCTION IF EXISTS fail_billing_plan_write()`).Error)
	})
}

// failQueriesOnTable makes GORM queries on one table return an error.
func failQueriesOnTable(t *testing.T, table string) {
	t.Helper()
	db := database.Conn()
	callbackName := "test:fail_queries_on_" + table
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			tx.AddError(errors.New("billing list storage detail"))
		}
	}))
	t.Cleanup(func() { require.NoError(t, db.Callback().Query().Remove(callbackName)) })
}

func TestAdminBillingPlanErrorsAreJSON(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	var trial billingmodels.BillingPlan
	require.NoError(t, database.Conn().Where("key = ?", "trial").First(&trial).Error)
	missing := uuid.NewString()
	validBody := billingPlanBody(t, map[string]any{"key": "json_errors_missing", "name": "Missing"})

	for _, tc := range []struct {
		name    string
		method  string
		path    string
		body    string
		status  int
		message string
	}{
		{"malformed body", "POST", "billing-plans", `{`, http.StatusBadRequest, "Invalid billing plan"},
		{"invalid update ID", "PUT", "billing-plans/not-a-uuid", validBody, http.StatusBadRequest, "Invalid billing plan ID"},
		{"invalid delete ID", "DELETE", "billing-plans/not-a-uuid", "", http.StatusBadRequest, "Invalid billing plan ID"},
		{"model validation", "POST", "billing-plans", billingPlanBody(t, map[string]any{"key": "Invalid Key"}), http.StatusBadRequest, "a valid key and name are required"},
		{"missing update target", "PUT", "billing-plans/" + missing, validBody, http.StatusNotFound, "Billing plan not found"},
		{"missing delete target", "DELETE", "billing-plans/" + missing, "", http.StatusNotFound, "Billing plan not found"},
		{"blocked deletion", "DELETE", "billing-plans/" + trial.ID.String(), "", http.StatusBadRequest, billingmodels.BillingPlanDeleteReasonSystem},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := request(tc.method, tc.path, tc.body)
			message := requireBillingPlanJSONError(t, response, tc.status)
			require.Contains(t, message, tc.message)
			require.NotContains(t, message, billingmodels.ErrCatalogPlan.Error())
		})
	}
}

func TestAdminBillingPlanConflictsReturnBadRequestAndKeepRows(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	db := database.Conn()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	keyA, keyB, keyC := "conflict_a_"+suffix, "conflict_b_"+suffix, "conflict_c_"+suffix
	productA := "prod_a_" + suffix
	const keyMessage = "This plan key already exists. Enter a different key."
	const productMessage = "This Polar product is already linked to another plan. Select a different product."
	t.Cleanup(func() {
		require.NoError(t, db.Where("key IN ?", []string{keyA, keyB, keyC}).Delete(&billingmodels.BillingPlan{}).Error)
	})

	createAdminTestPlan(t, request, billingPlanBody(t, map[string]any{"key": keyA, "polar_product_id": productA}))

	response := request("POST", "billing-plans", billingPlanBody(t, map[string]any{"key": keyA, "name": "Duplicate key"}))
	require.Equal(t, keyMessage, requireBillingPlanJSONError(t, response, http.StatusBadRequest))
	var count int64
	require.NoError(t, db.Model(&billingmodels.BillingPlan{}).Where("key = ?", keyA).Count(&count).Error)
	require.Equal(t, int64(1), count)

	response = request("POST", "billing-plans", billingPlanBody(t, map[string]any{"key": keyB, "polar_product_id": productA}))
	require.Equal(t, productMessage, requireBillingPlanJSONError(t, response, http.StatusBadRequest))
	_, err := billingmodels.FindBillingPlan(db, keyB)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)

	unlinked := createAdminTestPlan(t, request, billingPlanBody(t, map[string]any{"key": keyC}))
	before, err := billingmodels.FindBillingPlan(db, keyC)
	require.NoError(t, err)
	unlinked.PolarProductID = &productA
	body, err := json.Marshal(unlinked)
	require.NoError(t, err)
	response = request("PUT", "billing-plans/"+unlinked.ID.String(), string(body))
	require.Equal(t, productMessage, requireBillingPlanJSONError(t, response, http.StatusBadRequest))
	after, err := billingmodels.FindBillingPlan(db, keyC)
	require.NoError(t, err)
	require.Equal(t, before, after, "rejected update must not change the stored plan")
	linked, err := billingmodels.FindBillingPlanByProduct(db, productA)
	require.NoError(t, err)
	require.Equal(t, keyA, linked.Key)
}

func TestAdminBillingPlanUnexpectedSaveErrorIsReported(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	request := newAdminCatalogRequest(server, token)
	const key = "storage_failure_plan"
	const planName = "Customer pricing draft"
	failBillingPlanWrite(t, "fail_billing_plan_insert", "INSERT", "NEW.key = '"+key+"'")
	t.Cleanup(func() {
		require.NoError(t, database.Conn().Where("key = ?", key).Delete(&billingmodels.BillingPlan{}).Error)
	})
	logs := captureAdminErrorLogs(t)
	reports := sentrytest.BindHub(t)

	response := request("POST", "billing-plans", billingPlanBody(t, map[string]any{"key": key, "name": planName}))
	message := requireBillingPlanJSONError(t, response, http.StatusInternalServerError)
	require.NotContains(t, message, "storage")
	require.NotContains(t, response.Body.String(), planName)

	require.Contains(t, logs.String(), "billing plan storage detail")
	events := reports.Events()
	require.Len(t, events, 1)
	require.Contains(t, sentrytest.ExceptionValues(events[0]), "ERROR: billing plan storage detail (SQLSTATE P0001)")
	payload, err := json.Marshal(events[0])
	require.NoError(t, err)
	require.NotContains(t, string(payload), planName, "error reports must omit the request body")
}

func TestAdminBillingPlanUnexpectedErrorsReturnSafeJSON(t *testing.T) {
	const detail = "billing plan storage detail"

	t.Run("delete", func(t *testing.T) {
		server, _, token := setupAdminTestServer(t)
		request := newAdminCatalogRequest(server, token)
		const key = "delete_failure_plan"
		plan := createAdminTestPlan(t, request, billingPlanBody(t, map[string]any{"key": key}))
		t.Cleanup(func() {
			require.NoError(t, database.Conn().Where("key = ?", key).Delete(&billingmodels.BillingPlan{}).Error)
		})
		failBillingPlanWrite(t, "fail_billing_plan_delete", "DELETE", "OLD.key = '"+key+"'")

		response := request("DELETE", "billing-plans/"+plan.ID.String(), "")
		message := requireBillingPlanJSONError(t, response, http.StatusInternalServerError)
		require.NotContains(t, message, "storage")
		require.NotContains(t, response.Body.String(), detail)
		_, err := billingmodels.FindBillingPlan(database.Conn(), key)
		require.NoError(t, err)
	})

	for _, table := range []string{"billing_plans", "organization_billing_plans"} {
		t.Run("list "+table, func(t *testing.T) {
			server, _, token := setupAdminTestServer(t)
			request := newAdminCatalogRequest(server, token)
			failQueriesOnTable(t, table)

			response := request("GET", "billing-plans", "")
			message := requireBillingPlanJSONError(t, response, http.StatusInternalServerError)
			require.NotContains(t, message, "storage")
			require.NotContains(t, response.Body.String(), detail)
		})
	}
}
