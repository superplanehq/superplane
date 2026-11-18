package enterprisefeatures

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/admin/enterprise_features"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

type staticLicense struct {
	status licensing.Status
}

func (s staticLicense) Status() licensing.Status { return s.status }

func TestEnterpriseFeatureRPCsDenyNonAdministrators(t *testing.T) {
	r := support.Setup(t)
	memberAccount, err := models.CreateAccount("Member", "member-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)
	member, err := models.CreateUser(r.Organization.ID, memberAccount.ID, memberAccount.Email, memberAccount.Name)
	require.NoError(t, err)
	require.NoError(t, r.AuthService.AssignRole(member.ID.String(), models.RoleOrgOperator, r.Organization.ID.String(), models.DomainTypeOrganization))
	apiKey, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, "pipeline", nil, r.User, nil, nil)
	require.NoError(t, err)

	features := enterprise.NewOrganizationFeatures(staticLicense{})
	callers := []context.Context{
		authentication.SetAccountIDInMetadata(context.Background(), r.Account.ID.String()),
		authentication.SetAccountIDInMetadata(context.Background(), memberAccount.ID.String()),
		authentication.SetUserIdInMetadata(context.Background(), apiKey.ID.String()),
	}
	codesByCaller := []codes.Code{codes.PermissionDenied, codes.PermissionDenied, codes.Unauthenticated}

	for i, ctx := range callers {
		_, err = ListOrganizationEnterpriseFeatures(ctx, features, &pb.ListOrganizationEnterpriseFeaturesRequest{OrganizationId: r.Organization.ID.String()})
		assert.Equal(t, codesByCaller[i], grpcerrors.Code(err))
		_, err = SetOrganizationEnterpriseFeatures(ctx, features, &pb.SetOrganizationEnterpriseFeaturesRequest{OrganizationId: r.Organization.ID.String(), Features: []string{"groups"}})
		assert.Equal(t, codesByCaller[i], grpcerrors.Code(err))
		_, err = ListEnterpriseFeatureOrganizations(ctx, &pb.ListEnterpriseFeatureOrganizationsRequest{FeatureKey: "groups"})
		assert.Equal(t, codesByCaller[i], grpcerrors.Code(err))
		_, err = ListLicenseActivations(ctx, &pb.ListLicenseActivationsRequest{})
		assert.Equal(t, codesByCaller[i], grpcerrors.Code(err))
	}
}

func TestEnterpriseFeatureAdminRPCs(t *testing.T) {
	r := support.Setup(t)
	admin, err := models.CreateAccount("Install Admin", "admin-"+uuid.NewString()+"@example.com")
	require.NoError(t, err)
	require.NoError(t, models.PromoteToInstallationAdmin(admin.ID.String()))
	ctx := authentication.SetAccountIDInMetadata(context.Background(), admin.ID.String())
	second := createAdminOrg(t, "feature-org-b")
	third := createAdminOrg(t, "feature-org-c")

	t.Run("set and list report the grant and a stale grant", func(t *testing.T) {
		inserted, err := models.InsertOpenLicenseActivation(database.Conn(), &models.InstallationLicenseActivation{
			LicenseID:       uuid.New(),
			LicenseFeatures: []string{"custom_roles", "groups"},
		})
		require.NoError(t, err)
		require.True(t, inserted)
		features := enterprise.NewOrganizationFeatures(staticLicense{status: licensing.Status{
			State:   licensing.StateActive,
			License: &licensing.License{Features: []licensing.Feature{licensing.FeatureCustomRoles, licensing.FeatureGroups}},
		}})

		_, err = SetOrganizationEnterpriseFeatures(ctx, features, &pb.SetOrganizationEnterpriseFeaturesRequest{
			OrganizationId: r.Organization.ID.String(),
			Features:       []string{"not_a_feature"},
		})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Equal(t, "The license does not include not_a_feature.", grpcerrors.StatusMessage(err))

		enabled, err := SetOrganizationEnterpriseFeatures(ctx, features, &pb.SetOrganizationEnterpriseFeaturesRequest{
			OrganizationId: r.Organization.ID.String(),
			Features:       []string{"groups"},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"groups"}, enabled.GetAdded())
		require.Len(t, enabled.GetFeatures(), 2)
		assert.Equal(t, "groups", enabled.Features[1].GetKey())
		assert.True(t, enabled.Features[1].GetGranted())
		assert.True(t, enabled.Features[1].GetEffective())
		assert.Equal(t, admin.ID.String(), enabled.Features[1].GetGrantedBy())
		assert.Equal(t, admin.Name, enabled.Features[1].GetGrantedByName())
		assert.True(t, enabled.Features[0].GetLicensed())
		assert.False(t, enabled.Features[0].GetGranted())

		require.NoError(t, database.Conn().Exec("DELETE FROM installation_license_activations").Error)
		_, err = models.GrantEnterpriseFeatures(database.Conn(), r.Organization.ID, []string{"groups"}, models.EnterpriseFeatureSourceAdmin, &admin.ID)
		require.NoError(t, err)
		expired := enterprise.NewOrganizationFeatures(staticLicense{status: licensing.Status{State: licensing.StateExpired}})
		listed, err := ListOrganizationEnterpriseFeatures(ctx, expired, &pb.ListOrganizationEnterpriseFeaturesRequest{OrganizationId: r.Organization.ID.String()})
		require.NoError(t, err)
		assert.False(t, listed.GetLicenseActive())
		require.Len(t, listed.GetFeatures(), 1)
		assert.True(t, listed.Features[0].GetGranted())
		assert.False(t, listed.Features[0].GetLicensed())
		assert.False(t, listed.Features[0].GetEffective())

		_, err = SetOrganizationEnterpriseFeatures(ctx, expired, &pb.SetOrganizationEnterpriseFeaturesRequest{
			OrganizationId: r.Organization.ID.String(),
			Features:       []string{"custom_roles"},
		})
		assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
		assert.Contains(t, grpcerrors.StatusMessage(err), "active Enterprise license")
	})

	t.Run("lists organizations for a feature one page at a time", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec("DELETE FROM organization_enterprise_features").Error)
		for _, org := range []*models.Organization{r.Organization, second, third} {
			_, err := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"groups"}, models.EnterpriseFeatureSourceAdmin, &admin.ID)
			require.NoError(t, err)
		}

		_, err := ListEnterpriseFeatureOrganizations(ctx, &pb.ListEnterpriseFeatureOrganizationsRequest{FeatureKey: "missing"})
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

		var seen []string
		token := ""
		for {
			page, err := ListEnterpriseFeatureOrganizations(ctx, &pb.ListEnterpriseFeatureOrganizationsRequest{
				FeatureKey: "groups",
				PageSize:   1,
				PageToken:  token,
			})
			require.NoError(t, err)
			assert.Equal(t, int64(3), page.GetTotal())
			if token == "" {
				require.Len(t, page.GetOrganizations(), 1)
			}
			for _, org := range page.GetOrganizations() {
				seen = append(seen, org.GetOrganizationId())
				assert.NotEmpty(t, org.GetName())
				assert.Equal(t, models.EnterpriseFeatureSourceAdmin, org.GetSource())
			}
			if page.GetNextPageToken() == "" {
				break
			}
			token = page.GetNextPageToken()
		}
		assert.ElementsMatch(t, []string{r.Organization.ID.String(), second.ID.String(), third.ID.String()}, seen)
	})

	t.Run("lists activations with the grants a license removed", func(t *testing.T) {
		require.NoError(t, database.Conn().Exec("DELETE FROM installation_license_activations").Error)
		require.NoError(t, database.Conn().Exec("DELETE FROM organization_enterprise_features").Error)
		_, err := models.GrantEnterpriseFeatures(database.Conn(), r.Organization.ID, []string{"audit_logs"}, models.EnterpriseFeatureSourceAdmin, &admin.ID)
		require.NoError(t, err)
		lifecycle := enterprise.NewLicenseLifecycle(database.Conn)
		require.NoError(t, lifecycle.HandleStatus(context.Background(), licensing.Status{
			State: licensing.StateActive,
			License: &licensing.License{
				ID:       uuid.New(),
				Features: []licensing.Feature{licensing.FeatureCustomRoles},
			},
		}))

		page, err := ListLicenseActivations(ctx, &pb.ListLicenseActivationsRequest{PageSize: 10})
		require.NoError(t, err)
		require.NotEmpty(t, page.GetActivations())
		newest := page.Activations[0]
		assert.False(t, newest.GetAutoGranted())
		require.Len(t, newest.GetRemovedGrants(), 1)
		assert.Equal(t, "audit_logs", newest.RemovedGrants[0].GetFeatureKey())
		assert.Equal(t, "activation", newest.RemovedGrants[0].GetRemovedBy())
		assert.Equal(t, r.Organization.Name, newest.RemovedGrants[0].GetOrganizationName())
	})
}

func createAdminOrg(t *testing.T, name string) *models.Organization {
	t.Helper()
	org, err := models.CreateOrganization(name, "")
	require.NoError(t, err)
	return org
}
