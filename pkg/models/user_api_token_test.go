package models_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/gorm"
)

func Test__UserAPIToken__CreateListFindCount(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()

	tokens, err := models.ListUserAPITokens(tx, r.User)
	require.NoError(t, err)
	assert.Empty(t, tokens)

	count, err := models.CountUserAPITokens(tx, r.User)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	first := models.NewUserAPIToken(r.User, "First token", "hash-1")
	require.NoError(t, models.CreateUserAPIToken(tx, first))

	second := models.NewUserAPIToken(r.User, "Second token", "hash-2")
	require.NoError(t, models.CreateUserAPIToken(tx, second))

	tokens, err = models.ListUserAPITokens(tx, r.User)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, first.ID, tokens[0].ID)
	assert.Equal(t, second.ID, tokens[1].ID)

	count, err = models.CountUserAPITokens(tx, r.User)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	found, err := models.FindUserAPITokenByHash(tx, "hash-1")
	require.NoError(t, err)
	assert.Equal(t, first.ID, found.ID)
	assert.Nil(t, found.LastUsedAt)

	_, err = models.FindUserAPITokenByHash(tx, "does-not-exist")
	require.Error(t, err)

	scoped, err := models.FindUserAPIToken(tx, r.User, first.ID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, scoped.ID)

	_, err = models.FindUserAPIToken(tx, uuid.New(), first.ID)
	require.Error(t, err)
}

func Test__UserAPIToken__TouchLastUsed(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()

	token := models.NewUserAPIToken(r.User, "My token", "hash-touch")
	require.NoError(t, models.CreateUserAPIToken(tx, token))

	now := time.Now()
	require.NoError(t, models.TouchUserAPITokenLastUsed(tx, token.ID, now))

	found, err := models.FindUserAPITokenByHash(tx, "hash-touch")
	require.NoError(t, err)
	require.NotNil(t, found.LastUsedAt)
	assert.WithinDuration(t, now, *found.LastUsedAt, time.Second)
}

func Test__UserAPIToken__HardDelete(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()

	kept := models.NewUserAPIToken(r.User, "Kept token", "hash-kept")
	require.NoError(t, models.CreateUserAPIToken(tx, kept))

	removed := models.NewUserAPIToken(r.User, "Removed token", "hash-removed")
	require.NoError(t, models.CreateUserAPIToken(tx, removed))

	require.NoError(t, removed.HardDelete(tx))

	tokens, err := models.ListUserAPITokens(tx, r.User)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, kept.ID, tokens[0].ID)

	_, err = models.FindUserAPITokenByHash(tx, "hash-removed")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func Test__DeleteUserAPITokensForAccount(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()

	otherAccount, err := models.CreateAccountInTransaction(tx, support.RandomName("user"), support.RandomName("account")+"@example.com")
	require.NoError(t, err)

	otherUser, err := models.CreateUserInTransaction(tx, r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)

	ownToken := models.NewUserAPIToken(r.User, "Own token", "hash-own")
	require.NoError(t, models.CreateUserAPIToken(tx, ownToken))

	otherToken := models.NewUserAPIToken(otherUser.ID, "Other token", "hash-other")
	require.NoError(t, models.CreateUserAPIToken(tx, otherToken))

	require.NoError(t, models.DeleteUserAPITokensForAccount(tx, r.Account.ID))

	tokens, err := models.ListUserAPITokens(tx, r.User)
	require.NoError(t, err)
	assert.Empty(t, tokens)

	tokens, err = models.ListUserAPITokens(tx, otherUser.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, otherToken.ID, tokens[0].ID)
}
