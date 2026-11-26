package public

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	billingorganizations "github.com/superplanehq/superplane/ee/billing/organizations"
	"github.com/superplanehq/superplane/ee/rbac"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/grpc"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

// The real licensed gateway is the primary proof that create permission cannot
// modify a resource already connected to the requesting member.
func TestRBACCreationCannotExpandExistingGrants(t *testing.T) {
	for _, kind := range []string{"roles", "groups"} {
		t.Run(kind, func(t *testing.T) {
			r := support.Setup(t)
			defer r.Close()
			server, _, _ := setupTestServer(r, t)
			// Register the gateway without the fixture's static-file catch-all.
			server.Router = mux.NewRouter()
			license := licensingtest.EnterpriseService(licensing.FeatureCustomRoles, licensing.FeatureGroups)
			server.SetLicenseService(license)
			db := database.DB(t.Context())
			orgID := r.Organization.ID.String()
			_, err := models.GrantEnterpriseFeatures(db, r.Organization.ID,
				[]string{string(licensing.FeatureCustomRoles), string(licensing.FeatureGroups)},
				models.EnterpriseFeatureSourceAdmin, &r.Account.ID)
			require.NoError(t, err)
			features := enterprise.NewRegistry()
			features.Register(enterprise.Billing, billingorganizations.New())
			features.Register(enterprise.RBAC, rbac.NewService(r.AuthService, license))
			services, err := grpc.NewServices(grpc.ServicesConfig{
				Enterprise: features, Entitlements: license, License: license,
				AuthService: r.AuthService, Registry: r.Registry, Encryptor: r.Encryptor,
				OIDCProvider: support.NewOIDCProvider(), JWTSigner: jwt.NewSigner("test"),
				BaseURL: "http://localhost", WebhooksBaseURL: "http://localhost", RunnerAPIBaseURL: "http://localhost",
			})
			require.NoError(t, err)
			require.NoError(t, server.RegisterGRPCGateway(services))
			account, err := models.CreateAccount("Dummy creator", uuid.NewString()+"@example.test")
			require.NoError(t, err)
			member, err := models.CreateUserInTransaction(db, r.Organization.ID, account.ID, account.Email, account.Name)
			require.NoError(t, err)
			token, err := authentication.GenerateAccountToken(jwt.NewSigner("test-client-secret"), account.ID.String(), time.Now(), time.Hour)
			require.NoError(t, err)
			creatorRole := "role_creator"
			if kind == "groups" {
				creatorRole = "group_creator"
			}
			require.NoError(t, r.AuthService.CreateCustomRole(orgID, &authorization.RoleDefinition{
				Name: creatorRole, DomainType: models.DomainTypeOrganization, DisplayName: "Original", Description: "Original description",
				Permissions: []*authorization.Permission{{Resource: kind, Action: "create"}},
			}))
			require.NoError(t, r.AuthService.AssignRole(member.ID.String(), creatorRole, orgID, models.DomainTypeOrganization))
			if kind == "groups" {
				require.NoError(t, r.AuthService.CreateGroup(orgID, models.DomainTypeOrganization, "team", models.RoleOrgOperator, "Original", "Original description"))
				require.NoError(t, r.AuthService.AddUserToGroup(orgID, models.DomainTypeOrganization, member.ID.String(), "team"))
			}
			request := func(method, path, body string) *httptest.ResponseRecorder {
				return execRequest(server, requestParams{method: method, path: path, body: []byte(body), authCookie: token,
					contentType: "application/json", headers: map[string]string{"X-Organization-ID": orgID}})
			}
			existingName := creatorRole
			updateBody := `{"role":{"spec":{"display_name":"Updated","permissions":[{"resource":"roles","action":"create"}]}}}`
			if kind == "groups" {
				existingName = "team"
				updateBody = `{"group":{"spec":{"role":"org_admin"}}}`
			}
			assertRestricted := func() {
				for _, resource := range []string{"roles", "groups", "members", "org"} {
					allowed, err := r.AuthService.CheckOrganizationPermission(t.Context(), member.ID.String(), orgID, resource, "update")
					require.NoError(t, err)
					assert.False(t, allowed, resource)
				}
				response := request("PATCH", "/api/v1/organizations/"+orgID, `{"organization":{"metadata":{"name":"Escalated"}}}`)
				// Gateway authorization hides denied resources with NotFound.
				assert.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
				response = request("PUT", "/api/v1/"+kind+"/"+existingName, updateBody)
				assert.Equal(t, http.StatusNotFound, response.Code, response.Body.String())
			}
			assertRestricted()
			payload := func(name string) string {
				if kind == "groups" {
					return fmt.Sprintf(`{"group":{"metadata":{"name":%q},"spec":{"role":"org_admin","display_name":"Changed","description":"Changed description"}}}`, name)
				}
				return fmt.Sprintf(`{"role":{"metadata":{"name":%q},"spec":{"display_name":"Changed","description":"Changed description","permissions":[{"resource":"roles","action":"create"}],"inherited_role":{"metadata":{"name":"org_admin"}}}}}`, name)
			}
			// Positive control reaches the licensed create path before the attack.
			response := request("POST", "/api/v1/"+kind, payload("new-name"))
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			if kind == "roles" {
				definition, err := r.AuthService.GetRoleDefinition(t.Context(), "new-name", models.DomainTypeOrganization, orgID)
				require.NoError(t, err)
				assert.Equal(t, []*authorization.Permission{{Resource: "roles", Action: "create", DomainType: models.DomainTypeOrganization}}, definition.Permissions)
				require.NotNil(t, definition.InheritsFrom)
				assert.Equal(t, models.RoleOrgAdmin, definition.InheritsFrom.Name)
				metadata, err := models.FindRoleMetadata(db, "new-name", models.DomainTypeOrganization, orgID)
				require.NoError(t, err)
				assert.Equal(t, "Changed", metadata.DisplayName)
				assert.Equal(t, "Changed description", metadata.Description)
			} else {
				role, err := r.AuthService.GetGroupRole(t.Context(), orgID, models.DomainTypeOrganization, "new-name")
				require.NoError(t, err)
				assert.Equal(t, models.RoleOrgAdmin, role)
				metadata, err := models.FindGroupMetadata("new-name", models.DomainTypeOrganization, orgID)
				require.NoError(t, err)
				assert.Equal(t, "Changed", metadata.DisplayName)
				assert.Equal(t, "Changed description", metadata.Description)
			}
			roles, err := r.AuthService.GetUserRolesForOrg(t.Context(), member.ID.String(), orgID)
			require.NoError(t, err)
			for _, role := range roles {
				assert.NotEqual(t, "new-name", role.Name)
				assert.NotEqual(t, models.RoleOrgAdmin, role.Name)
			}
			groups, err := r.AuthService.GetUserGroups(t.Context(), orgID, models.DomainTypeOrganization, member.ID.String())
			require.NoError(t, err)
			assert.NotContains(t, groups, "new-name")
			var beforeRoles, afterRoles []models.RoleMetadata
			var beforeGroups, afterGroups []models.GroupMetadata
			var beforePolicies, afterPolicies []gormadapter.CasbinRule
			require.NoError(t, db.Order("id").Find(&beforeRoles).Error)
			require.NoError(t, db.Order("id").Find(&beforeGroups).Error)
			require.NoError(t, db.Order("id").Find(&beforePolicies).Error)
			response = request("POST", "/api/v1/"+kind, payload(existingName))
			assert.Equal(t, http.StatusConflict, response.Code, response.Body.String())
			var apiError struct {
				Code int `json:"code"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &apiError))
			assert.Equal(t, 6, apiError.Code)
			require.NoError(t, db.Order("id").Find(&afterRoles).Error)
			require.NoError(t, db.Order("id").Find(&afterGroups).Error)
			require.NoError(t, db.Order("id").Find(&afterPolicies).Error)
			assert.Equal(t, beforeRoles, afterRoles)
			assert.Equal(t, beforeGroups, afterGroups)
			assert.Equal(t, beforePolicies, afterPolicies)
			assertRestricted()
		})
	}
}
