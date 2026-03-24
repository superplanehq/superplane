package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFindFirstHumanUserByOrganizationSkipsDeletedUsers(t *testing.T) {
	r := support.Setup(t)

	account, err := models.CreateAccount(support.RandomName("account")+"@example.com", support.RandomName("user"))
	require.NoError(t, err)

	secondUser, err := models.CreateUser(r.Organization.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)

	firstUser, err := models.FindActiveUserByEmail(r.Organization.ID.String(), r.Account.Email)
	require.NoError(t, err)
	require.NoError(t, firstUser.Delete())

	user, err := models.FindFirstHumanUserByOrganization(r.Organization.ID.String())
	require.NoError(t, err)
	assert.Equal(t, secondUser.ID, user.ID)
}

func TestCountOrganizationsByBillingAccountExcludesSoftDeletedOrgs(t *testing.T) {
	r := support.Setup(t)

	count, err := models.CountOrganizationsByBillingAccount(r.Account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	org2 := support.CreateOrganization(t, r, r.User)
	_, err = models.CreateUser(org2.ID, r.Account.ID, r.Account.Email, r.Account.Name)
	require.NoError(t, err)

	count, err = models.CountOrganizationsByBillingAccount(r.Account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	err = models.SoftDeleteOrganization(org2.ID.String())
	require.NoError(t, err)

	count, err = models.CountOrganizationsByBillingAccount(r.Account.ID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "soft-deleted org should not count toward billing account limit")
}
