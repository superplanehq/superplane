package public

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/enterprise"
	"github.com/superplanehq/superplane/pkg/licensing"
	"github.com/superplanehq/superplane/pkg/licensing/licensingtest"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/encryption"
)

func TestAdminLicenseDryRunMatchesTheAppliedChange(t *testing.T) {
	server, registry, token := setupAdminTestServer(t)
	issuer := licensingtest.NewIssuer("test-key")
	server.SetLicenseService(licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		licensing.NewDatabaseSource(encryption.NewEncryptor()),
		licensing.WithLifecycleHandler(enterprise.NewLicenseLifecycle(database.Conn)),
	))
	second, err := models.CreateOrganization("second-org", "")
	require.NoError(t, err)
	adminID := registry.Account.ID
	for _, orgID := range []string{registry.Organization.ID.String(), second.ID.String()} {
		org, findErr := models.FindOrganizationByID(orgID)
		require.NoError(t, findErr)
		_, grantErr := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"groups"}, models.EnterpriseFeatureSourceAdmin, &adminID)
		require.NoError(t, grantErr)
	}

	raw := issuer.License(licensing.FeatureCustomRoles)
	beforeGrants := enterpriseGrantCount(t)
	beforeActivations := licenseActivationCount(t)

	dryRun := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath + "?dry_run=true",
		body: licenseBody(t, raw), contentType: "application/json", authCookie: token,
	})
	require.Equal(t, http.StatusOK, dryRun.Code, dryRun.Body.String())
	var planned licenseInstallDryRunResponse
	require.NoError(t, json.Unmarshal(dryRun.Body.Bytes(), &planned))
	assert.Equal(t, enterprise.LicenseEventActivation, planned.Event)
	assert.Empty(t, planned.AutoGrantOrganizationID)
	require.Len(t, planned.RemovedGrants, 2)
	assert.Equal(t, "groups", planned.RemovedGrants[0].FeatureKey)
	assert.Equal(t, beforeGrants, enterpriseGrantCount(t))
	assert.Equal(t, beforeActivations, licenseActivationCount(t))

	installed := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath,
		body: licenseBody(t, raw), contentType: "application/json", authCookie: token,
	})
	require.Equal(t, http.StatusOK, installed.Code, installed.Body.String())
	var applied installationLicenseResponse
	require.NoError(t, json.Unmarshal(installed.Body.Bytes(), &applied))
	assert.Equal(t, planned.Event, applied.Event)
	assert.Equal(t, planned.AutoGrantOrganizationID, applied.AutoGrantOrganizationID)
	assert.ElementsMatch(t, planned.RemovedGrants, applied.RemovedGrants)
	assert.Equal(t, "active", applied.State)
	assert.Equal(t, int64(0), applied.FeatureCounts["groups"])
	assert.Equal(t, int64(2), applied.OrganizationCount)
	assert.Equal(t, int64(0), enterpriseGrantCount(t))

	same := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath + "?dry_run=true",
		body: licenseBody(t, raw), contentType: "application/json", authCookie: token,
	})
	require.Equal(t, http.StatusOK, same.Code, same.Body.String())
	var unchanged licenseInstallDryRunResponse
	require.NoError(t, json.Unmarshal(same.Body.Bytes(), &unchanged))
	assert.Equal(t, enterprise.LicenseEventNone, unchanged.Event)
	assert.Empty(t, unchanged.RemovedGrants)
	assert.Equal(t, int64(1), licenseActivationCount(t))

	for _, orgID := range []string{registry.Organization.ID.String(), second.ID.String()} {
		org, findErr := models.FindOrganizationByID(orgID)
		require.NoError(t, findErr)
		_, grantErr := models.GrantEnterpriseFeatures(database.Conn(), org.ID, []string{"custom_roles"}, models.EnterpriseFeatureSourceAdmin, &adminID)
		require.NoError(t, grantErr)
	}
	removalPreview := execRequest(server, requestParams{
		method: http.MethodDelete, path: licensePath + "?dry_run=true", authCookie: token,
	})
	require.Equal(t, http.StatusOK, removalPreview.Code, removalPreview.Body.String())
	var removal licenseRemovalDryRunResponse
	require.NoError(t, json.Unmarshal(removalPreview.Body.Bytes(), &removal))
	assert.Len(t, removal.RemovedGrants, 2)
	assert.Equal(t, int64(2), enterpriseGrantCount(t))

	removed := execRequest(server, requestParams{method: http.MethodDelete, path: licensePath, authCookie: token})
	require.Equal(t, http.StatusOK, removed.Code, removed.Body.String())
	var afterRemoval installationLicenseResponse
	require.NoError(t, json.Unmarshal(removed.Body.Bytes(), &afterRemoval))
	assert.Equal(t, "none", afterRemoval.State)
	assert.ElementsMatch(t, removal.RemovedGrants, afterRemoval.RemovedGrants)
	assert.Equal(t, int64(0), enterpriseGrantCount(t))
}

func TestAdminLicenseDryRunForOneOrganization(t *testing.T) {
	server, registry, token := setupAdminTestServer(t)
	issuer := licensingtest.NewIssuer("test-key")
	server.SetLicenseService(licensing.NewService(
		licensing.NewVerifier(licensingtest.KeySet(issuer)),
		licensing.NewDatabaseSource(encryption.NewEncryptor()),
		licensing.WithLifecycleHandler(enterprise.NewLicenseLifecycle(database.Conn)),
	))

	raw := issuer.License(licensing.FeatureCustomRoles, licensing.FeatureGroups)
	dryRun := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath + "?dry_run=true",
		body: licenseBody(t, raw), contentType: "application/json", authCookie: token,
	})
	require.Equal(t, http.StatusOK, dryRun.Code, dryRun.Body.String())
	var planned licenseInstallDryRunResponse
	require.NoError(t, json.Unmarshal(dryRun.Body.Bytes(), &planned))
	assert.Equal(t, enterprise.LicenseEventActivation, planned.Event)
	assert.Equal(t, registry.Organization.ID.String(), planned.AutoGrantOrganizationID)
	assert.Empty(t, planned.RemovedGrants)
	assert.Equal(t, int64(0), enterpriseGrantCount(t))

	installed := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath,
		body: licenseBody(t, raw), contentType: "application/json", authCookie: token,
	})
	require.Equal(t, http.StatusOK, installed.Code, installed.Body.String())
	var applied installationLicenseResponse
	require.NoError(t, json.Unmarshal(installed.Body.Bytes(), &applied))
	assert.Equal(t, planned.AutoGrantOrganizationID, applied.AutoGrantOrganizationID)
	assert.Equal(t, int64(1), applied.FeatureCounts["custom_roles"])
	assert.Equal(t, int64(1), applied.FeatureCounts["groups"])
	assert.Equal(t, int64(2), enterpriseGrantCount(t))

	claims := licensingtest.Claims(licensing.FeatureGroups)
	past := time.Now().Add(-48 * time.Hour).Unix()
	claims["iat"] = past
	claims["nbf"] = past
	claims["exp"] = time.Now().Add(-24 * time.Hour).Unix()
	expired := execRequest(server, requestParams{
		method: http.MethodPut, path: licensePath + "?dry_run=true",
		body: licenseBody(t, issuer.Sign(claims)), contentType: "application/json", authCookie: token,
	})
	assert.Equal(t, http.StatusUnprocessableEntity, expired.Code)
	assert.Equal(t, int64(2), enterpriseGrantCount(t))
	assert.Equal(t, int64(1), licenseActivationCount(t))
}

func enterpriseGrantCount(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Conn().Model(&models.OrganizationEnterpriseFeature{}).Count(&count).Error)
	return count
}

func licenseActivationCount(t *testing.T) int64 {
	t.Helper()
	var count int64
	require.NoError(t, database.Conn().Model(&models.InstallationLicenseActivation{}).Count(&count).Error)
	return count
}
