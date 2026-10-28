package billinghttp

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestAdminFeatureOrganizations(t *testing.T) {
	server, _, token := setupAdminTestServer(t)
	db := database.Conn()
	feature := billingmodels.PlanFeatureDefinition{Key: "access_listing", Label: "Access listing", Gated: false}
	require.NoError(t, billingmodels.CreatePlanFeatureDefinition(db, &feature))
	plan := billingmodels.BillingPlan{Key: "access_listing_custom", Name: "Custom access", Category: "paid", Visibility: "private", CreditFrequency: "none", CreditExpiration: "never", Features: []string{feature.Key}}
	require.NoError(t, billingmodels.SaveBillingPlan(db, &plan))
	var ids []uuid.UUID
	t.Cleanup(func() {
		require.NoError(t, db.Where("organization_id IN ?", ids).Delete(&billingmodels.OrganizationFeatureGrant{}).Error)
		require.NoError(t, db.Where("organization_id IN ?", ids).Delete(&billingmodels.OrganizationBillingPlan{}).Error)
		require.NoError(t, db.Unscoped().Where("id IN ?", ids).Delete(&models.Organization{}).Error)
		require.NoError(t, billingmodels.DeleteBillingPlan(db, plan.ID))
		require.NoError(t, db.Delete(&feature).Error)
	})
	for _, fixture := range []struct {
		name                          string
		grant, enabled, plan, deleted bool
	}{
		{"Alpha", true, true, false, false},
		{"Beta", false, false, true, false},
		{"Beta", true, true, true, false},
		{"Disabled with plan", true, false, true, false},
		{"Neither", false, false, false, false},
		{"Disabled", true, false, false, false},
		{"Deleted", true, true, true, true},
	} {
		org := models.Organization{ID: uuid.New(), Name: fixture.name, Slug: uuid.NewString()}
		require.NoError(t, db.Create(&org).Error)
		ids = append(ids, org.ID)
		if fixture.grant {
			require.NoError(t, billingmodels.SetOrganizationFeatureGrant(db, billingmodels.OrganizationFeatureGrant{OrganizationID: org.ID, FeatureKey: feature.Key, Enabled: fixture.enabled}))
		}
		if fixture.plan {
			expired := time.Now().Add(-time.Hour)
			require.NoError(t, db.Create(&billingmodels.OrganizationBillingPlan{OrganizationID: org.ID, PlanID: plan.ID, Plan: plan.Key, PlanSource: billingmodels.BillingPlanSourcePolar, CurrentPeriodEnd: &expired, CancelAtPeriodEnd: true}).Error)
		}
		if fixture.deleted {
			require.NoError(t, db.Delete(&org).Error)
		}
	}
	type item struct {
		ID           uuid.UUID `json:"id"`
		Name         string    `json:"name"`
		Slug         string    `json:"slug"`
		EnabledGrant bool      `json:"enabled_grant"`
		Plan         *struct {
			ID   uuid.UUID `json:"id"`
			Key  string    `json:"key"`
			Name string    `json:"name"`
		} `json:"plan"`
	}
	type page struct {
		Items  []item `json:"items"`
		Total  int64  `json:"total"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	request := func(key, query string) page {
		response := execRequest(server, requestParams{method: "GET", path: "/admin/api/feature-definitions/" + key + "/organizations" + query, authCookie: token})
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		var result page
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		return result
	}
	result := request(feature.Key, "")
	require.EqualValues(t, 4, result.Total)
	require.Equal(t, 50, result.Limit)
	require.Len(t, result.Items, 4)
	require.Equal(t, "Alpha", result.Items[0].Name)
	require.Equal(t, ids[0], result.Items[0].ID)
	require.NotEmpty(t, result.Items[0].Slug)
	require.True(t, result.Items[0].EnabledGrant)
	require.Nil(t, result.Items[0].Plan)
	require.Equal(t, "Beta", result.Items[1].Name)
	require.Equal(t, "Beta", result.Items[2].Name)
	require.Less(t, result.Items[1].ID.String(), result.Items[2].ID.String())
	for _, row := range result.Items[1:] {
		require.NotNil(t, row.Plan)
		require.Equal(t, plan.ID, row.Plan.ID)
		require.Equal(t, plan.Key, row.Plan.Key)
		require.Equal(t, plan.Name, row.Plan.Name)
		require.Equal(t, row.ID == ids[2], row.EnabledGrant)
	}
	filtered := request(feature.Key, "?search=bEtA&limit=1&offset=1")
	require.EqualValues(t, 2, filtered.Total)
	require.Equal(t, 1, filtered.Limit)
	require.Equal(t, 1, filtered.Offset)
	require.Equal(t, result.Items[2], filtered.Items[0])
	require.EqualValues(t, 1, request(feature.Key, "?search="+ids[0].String()).Total)
	require.Empty(t, request(feature.Key, "?search=missing").Items)
	require.Equal(t, 200, request(feature.Key, "?limit=999").Limit)
	require.Equal(t, 50, request(feature.Key, "?limit=-1&offset=-1").Limit)
	require.Empty(t, request(feature.Key, "?offset=999").Items)
	require.EqualValues(t, 4, request(feature.Key, "?offset=999").Total)
	response := execRequest(server, requestParams{method: "GET", path: "/admin/api/feature-definitions/unknown/organizations", authCookie: token})
	require.Equal(t, http.StatusNotFound, response.Code)
	// Reading expired saved plans must never lapse the assignment.
	saved, err := billingmodels.FindOrganizationBillingPlan(db, ids[1])
	require.NoError(t, err)
	require.Equal(t, plan.ID, saved.PlanID)
}
