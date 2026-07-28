package models_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestMapAPIKeyNameUniqueConstraintError(t *testing.T) {
	t.Run("maps api key name unique violation", func(t *testing.T) {
		err := models.MapAPIKeyNameUniqueConstraintError(&pgconn.PgError{
			Code:           "23505",
			ConstraintName: "unique_api_key_in_organization",
		})

		assert.ErrorIs(t, err, models.ErrAPIKeyNameAlreadyExists)
	})

	t.Run("preserves unique violations on other constraints", func(t *testing.T) {
		original := &pgconn.PgError{
			Code:           "23505",
			ConstraintName: "unique_human_user_in_organization",
		}

		err := models.MapAPIKeyNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})

	t.Run("preserves unrelated errors", func(t *testing.T) {
		original := errors.New("other error")

		err := models.MapAPIKeyNameUniqueConstraintError(original)

		assert.ErrorIs(t, err, original)
	})

	t.Run("preserves nil", func(t *testing.T) {
		assert.NoError(t, models.MapAPIKeyNameUniqueConstraintError(nil))
	})
}

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
