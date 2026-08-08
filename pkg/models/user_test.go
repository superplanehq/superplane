package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
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

func TestListActiveHumanUsersByIDs(t *testing.T) {
	r := support.Setup(t)

	eligibleAccount, err := models.CreateAccount("eligible@example.com", "Eligible")
	require.NoError(t, err)
	eligibleUser, err := models.CreateUser(r.Organization.ID, eligibleAccount.ID, eligibleAccount.Email, eligibleAccount.Name)
	require.NoError(t, err)

	deletedAccount, err := models.CreateAccount("deleted@example.com", "Deleted")
	require.NoError(t, err)
	deletedUser, err := models.CreateUser(r.Organization.ID, deletedAccount.ID, deletedAccount.Email, deletedAccount.Name)
	require.NoError(t, err)
	require.NoError(t, deletedUser.Delete())

	apiKey, err := models.CreateAPIKey(database.DB(t.Context()), r.Organization.ID, "Bot", nil, r.UserModel.ID, nil, nil)
	require.NoError(t, err)

	blankEmail := "   "
	blankEmailUser := &models.User{
		OrganizationID: r.Organization.ID,
		Email:          &blankEmail,
		Name:           "Blank Email",
		Type:           models.UserTypeHuman,
	}
	require.NoError(t, database.DB(t.Context()).Create(blankEmailUser).Error)

	users, err := models.ListActiveHumanUsersByIDs(database.DB(t.Context()), r.Organization.ID.String(), []string{
		r.UserModel.ID.String(), eligibleUser.ID.String(), deletedUser.ID.String(), apiKey.ID.String(), blankEmailUser.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, users, 2)
	assert.ElementsMatch(t, []string{r.UserModel.ID.String(), eligibleUser.ID.String()}, []string{users[0].ID.String(), users[1].ID.String()})

	users, err = models.ListActiveHumanUsersByIDs(database.DB(t.Context()), r.Organization.ID.String(), nil)
	require.NoError(t, err)
	assert.Empty(t, users)
}
