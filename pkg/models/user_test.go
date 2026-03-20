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
