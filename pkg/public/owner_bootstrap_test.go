package public

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
	"github.com/superplanehq/superplane/test/support"
)

func setBootstrapEnv(t *testing.T, email, firstName, lastName, password, orgName string) {
	t.Helper()
	t.Setenv(config.OwnerBootstrapEmailEnv, email)
	t.Setenv(config.OwnerBootstrapPasswordEnv, password)
	t.Setenv(config.OwnerBootstrapFirstNameEnv, firstName)
	t.Setenv(config.OwnerBootstrapLastNameEnv, lastName)
	t.Setenv(config.OwnerBootstrapOrgNameEnv, orgName)
}

func TestTryBootstrapOwnerFromEnvCreatesOwner(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())
	middleware.ResetOwnerSetupStateForTests()
	require.True(t, middleware.IsOwnerSetupRequired())

	setBootstrapEnv(t, "owner@example.com", "Owner", "User", "Password1", "")

	assert.True(t, TryBootstrapOwnerFromEnv(r.AuthService))
	assert.False(t, middleware.IsOwnerSetupRequired())

	account, err := models.FindAccountByEmail("owner@example.com")
	require.NoError(t, err)
	assert.True(t, account.IsInstallationAdmin())

	organization, err := models.FindOrganizationByName(config.DefaultOwnerBootstrapOrgName)
	require.NoError(t, err)
	assert.NotEmpty(t, organization.Slug)
}

func TestTryBootstrapOwnerFromEnvCustomOrgName(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())
	middleware.ResetOwnerSetupStateForTests()

	setBootstrapEnv(t, "owner@example.com", "Owner", "User", "Password1", "Acme")

	assert.True(t, TryBootstrapOwnerFromEnv(r.AuthService))

	_, err := models.FindOrganizationByName("Acme")
	require.NoError(t, err)
}

func TestTryBootstrapOwnerFromEnvPartialFallsBack(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())
	middleware.ResetOwnerSetupStateForTests()

	setBootstrapEnv(t, "owner@example.com", "", "", "", "")

	assert.False(t, TryBootstrapOwnerFromEnv(r.AuthService))
	assert.True(t, middleware.IsOwnerSetupRequired())
}

func TestTryBootstrapOwnerFromEnvInvalidPasswordFallsBack(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, database.TruncateTables())
	middleware.ResetOwnerSetupStateForTests()

	setBootstrapEnv(t, "owner@example.com", "Owner", "User", "weak", "")

	assert.False(t, TryBootstrapOwnerFromEnv(r.AuthService))
	assert.True(t, middleware.IsOwnerSetupRequired())
}

func TestTryBootstrapOwnerFromEnvSkipsWhenInitialized(t *testing.T) {
	r := support.Setup(t)
	middleware.ResetOwnerSetupStateForTests()

	// support.Setup seeds a user, so setup is not required. Bootstrap must
	// stay out of the way even when env vars are fully set.
	setBootstrapEnv(t, "owner@example.com", "Owner", "User", "Password1", "")

	assert.False(t, TryBootstrapOwnerFromEnv(r.AuthService))
}
