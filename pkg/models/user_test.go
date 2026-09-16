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

func TestRefuseIfLastOrganizationOwner(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	err := models.RefuseIfLastOrganizationOwner(db, r.Organization.ID, r.User)
	require.ErrorIs(t, err, models.ErrLastOrganizationOwner)

	member := support.CreateUser(t, r, r.Organization.ID)
	require.NoError(t, models.SetUserIsOwner(db, member.ID, true))
	require.NoError(t, models.RefuseIfLastOrganizationOwner(db, r.Organization.ID, r.User))
}
