package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/gorm"
)

func TestAccountBlockAndUnblock(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	account, err := CreateAccount("Blocked User", "blocked-user@example.com")
	require.NoError(t, err)
	assert.False(t, account.IsBlocked())

	org, err := CreateOrganization("Block Org", "")
	require.NoError(t, err)

	human, err := CreateUser(org.ID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
	require.NoError(t, human.UpdateTokenHash("human-token-hash"))

	personalToken := NewUserAPIToken(human.ID, "CI token", "personal-token-hash")
	require.NoError(t, CreateUserAPIToken(database.Conn(), personalToken))

	description := "org key"
	apiKey, err := CreateAPIKey(database.Conn(), org.ID, "org-key", &description, human.ID, nil, nil)
	require.NoError(t, err)
	require.NoError(t, apiKey.UpdateTokenHash("api-key-token-hash"))
	require.NoError(t, human.Delete())

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return account.Block(tx, time.Now())
	}))

	blocked, err := FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.True(t, blocked.IsBlocked())
	require.NotNil(t, blocked.BlockedAt)
	require.NotNil(t, blocked.PasswordChangedAt)

	refreshedHuman, err := FindMaybeDeletedUserByID(org.ID.String(), human.ID.String())
	require.NoError(t, err)
	assert.Empty(t, refreshedHuman.TokenHash)

	refreshedKey, err := FindActiveUserByID(org.ID.String(), apiKey.ID.String())
	require.NoError(t, err)
	assert.Empty(t, refreshedKey.TokenHash)

	remainingTokens, err := ListUserAPITokens(database.Conn(), human.ID)
	require.NoError(t, err)
	assert.Empty(t, remainingTokens, "personal API tokens should be revoked when the account is blocked")

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return blocked.Unblock(tx)
	}))

	unblocked, err := FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.False(t, unblocked.IsBlocked())
	assert.Nil(t, unblocked.BlockedAt)
	// Sessions stay invalidated after unblock.
	assert.NotNil(t, unblocked.PasswordChangedAt)
}

func TestBlockAccountIsIdempotent(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	account, err := CreateAccount("Idempotent", "idempotent-block@example.com")
	require.NoError(t, err)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return account.Block(tx, time.Now())
	}))
	first, err := FindAccountByID(account.ID.String())
	require.NoError(t, err)
	require.NotNil(t, first.BlockedAt)

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		return account.Block(tx, time.Now())
	}))
	second, err := FindAccountByID(account.ID.String())
	require.NoError(t, err)
	assert.True(t, second.IsBlocked())
	assert.Equal(t, first.BlockedAt.Unix(), second.BlockedAt.Unix())
}

func TestIsBlocked(t *testing.T) {
	assert.False(t, (*Account)(nil).IsBlocked())
	assert.False(t, (&Account{}).IsBlocked())

	blockedAt := time.Now()
	assert.True(t, (&Account{BlockedAt: &blockedAt}).IsBlocked())
}
