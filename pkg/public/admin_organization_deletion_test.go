package public

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingmodels "github.com/superplanehq/superplane/ee/billing/models"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestAdminDeleteOrganization(t *testing.T) {
	server, r, token := setupAdminTestServer(t)
	registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
	db := database.DB(t.Context())
	organization, err := models.CreateOrganization("Delete target", "")
	require.NoError(t, err)
	owner, err := models.CreateUserInTransaction(db, organization.ID, r.Account.ID, r.Account.Email, r.Account.Name)
	require.NoError(t, err)
	require.NoError(t, server.authService.SetupOrganization(db, organization.ID.String(), owner.ID.String()))
	overview := execRequest(server, requestParams{method: "GET", path: "/admin/api/organizations/" + organization.ID.String(), authCookie: token})
	require.Equal(t, http.StatusOK, overview.Code, overview.Body.String())

	regular, err := models.CreateAccount("Regular", uuid.NewString()+"@example.com")
	require.NoError(t, err)
	regularToken, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), regular.ID.String(), time.Now(), time.Hour)
	require.NoError(t, err)
	unauthenticated := execRequest(server, requestParams{method: "DELETE", path: "/admin/api/organizations/" + organization.ID.String()})
	assert.NotEqual(t, http.StatusOK, unauthenticated.Code)

	for _, test := range []struct {
		name   string
		id     string
		token  string
		status int
	}{
		{"non-admin cannot delete", organization.ID.String(), regularToken, http.StatusNotFound},
		{"invalid ID", "invalid", token, http.StatusBadRequest},
		{"unknown organization", uuid.NewString(), token, http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := execRequest(server, requestParams{method: "DELETE", path: "/admin/api/organizations/" + test.id, authCookie: test.token})
			assert.Equal(t, test.status, response.Code)
			_, err := models.FindOrganizationByID(organization.ID.String())
			require.NoError(t, err)
		})
	}

	// Installation access does not require membership in the target organization.
	require.NoError(t, db.Delete(owner).Error)
	recorder := useAnalytics(t)
	response := execRequest(server, requestParams{method: "DELETE", path: "/admin/api/organizations/" + organization.ID.String(), authCookie: token})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	_, err = models.FindOrganizationByID(organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindRoleMetadata(db, "org_admin", models.DomainTypeOrganization, organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = models.FindAccountByID(r.Account.ID.String())
	require.NoError(t, err)
	events := recorder.snapshot()
	require.Len(t, events, 1)
	assert.Equal(t, "organization_deleted", string(events[0].Name))
	assert.Equal(t, organization.ID.String(), events[0].OrganizationID)
	assert.Equal(t, r.Account.ID.String(), events[0].DistinctID)
	assert.Equal(t, "admin", events[0].Properties["deletion_source"])
}

func TestAdminDeleteOrganizationPreservesBillableOrganization(t *testing.T) {
	for _, verified := range []bool{true, false} {
		name := "live subscription"
		if !verified {
			name = "subscription cannot be verified"
		}
		t.Run(name, func(t *testing.T) {
			server, r, token := setupAdminTestServer(t)
			registerTestGRPCGateway(t, server, r.AuthService, r.Registry, r.Encryptor, support.NewOIDCProvider())
			recorder := useAnalytics(t)
			db := database.DB(t.Context())
			require.NoError(t, billingmodels.BindLegacyBillingProduct(db, "prod_business"))
			_, err := billingmodels.ApplyPolarSubscription(db, r.Organization.ID, billingmodels.PolarSubscriptionApply{
				ID: "sub_delete", ProductID: "prod_business", Status: billingmodels.PolarSubscriptionStatusActive,
			}, nil)
			require.NoError(t, err)
			useAccountDeletionPolar(t, func(w http.ResponseWriter, orgID string) {
				if !verified {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				writePolarSubscriptions(t, w, map[string]any{"id": "sub_delete", "status": "active", "external_customer_id": orgID})
			})
			response := execRequest(server, requestParams{method: "DELETE", path: "/admin/api/organizations/" + r.Organization.ID.String(), authCookie: token})
			if verified {
				assert.Equal(t, http.StatusConflict, response.Code)
				assert.Contains(t, response.Body.String(), "Cancel the subscription in Billing")
			} else {
				assert.Equal(t, http.StatusServiceUnavailable, response.Code)
				assert.Contains(t, response.Body.String(), "Try again later")
			}
			_, err = models.FindOrganizationByID(r.Organization.ID.String())
			require.NoError(t, err)
			assert.Empty(t, recorder.snapshot())
		})
	}
}
