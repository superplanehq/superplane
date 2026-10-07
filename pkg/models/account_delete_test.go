package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func TestListOrganizationsPendingAccountDeletion(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	pending, err := models.ListOrganizationsPendingAccountDeletion(db, r.Account.ID)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, r.Organization.ID, pending[0].ID)

	second := support.CreateOrganization(t, r, r.User)
	pending, err = models.ListOrganizationsPendingAccountDeletion(db, r.Account.ID)
	require.NoError(t, err)
	require.Len(t, pending, 2)
	assert.LessOrEqual(t, pending[0].Name, pending[1].Name)

	otherAccount, err := models.CreateAccount("Other Owner", "other-pending-owner@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(db, second.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(db, otherUser.ID, true))

	pending, err = models.ListOrganizationsPendingAccountDeletion(db, r.Account.ID)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, r.Organization.ID, pending[0].ID)
}

func TestAccountSoftDelete_MarksOnlyLastOwnerOrganizations(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	kept := support.CreateOrganization(t, r, r.User)
	otherAccount, err := models.CreateAccount("Other Owner", "other-softdelete-owner@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(db, kept.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)
	require.NoError(t, models.SetUserIsOwner(db, otherUser.ID, true))

	require.NoError(t, r.Account.SoftDelete(db, time.Now()))

	_, err = models.FindOrganizationByID(r.Organization.ID.String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = models.FindOrganizationByID(kept.ID.String())
	require.NoError(t, err)
}
