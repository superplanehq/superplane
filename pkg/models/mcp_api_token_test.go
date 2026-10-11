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

func Test__MCPAPIToken__CreateFindRevoke(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()
	factoryID := uuid.New()
	token := models.NewMCPAPIToken(r.User, r.Organization.ID, factoryID, "Build server", "http://localhost:8000/mcp", "hash-mcp", models.MCPGrantedScopes)
	require.NoError(t, models.CreateMCPAPIToken(tx, token))

	found, err := models.FindMCPAPITokenByHash(tx, "hash-mcp")
	require.NoError(t, err)
	assert.Equal(t, token.ID, found.ID)
	assert.Equal(t, models.MCPGrantedScopes, []string(found.Scopes))
	assert.Nil(t, found.LastUsedAt)

	now := time.Now()
	require.NoError(t, models.TouchMCPAPITokenLastUsed(tx, token.ID, now))
	found, err = models.FindMCPAPITokenByID(tx, token.ID)
	require.NoError(t, err)
	require.NotNil(t, found.LastUsedAt)
	assert.WithinDuration(t, now, *found.LastUsedAt, time.Second)

	require.NoError(t, found.HardDelete(tx))
	_, err = models.FindMCPAPITokenByHash(tx, "hash-mcp")
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNotFound)
}

func Test__MCPAPIToken__RejectsEmptyName(t *testing.T) {
	r := support.Setup(t)
	token := models.NewMCPAPIToken(r.User, r.Organization.ID, uuid.New(), "   ", "http://localhost:8000/mcp", "hash-empty", nil)
	err := models.CreateMCPAPIToken(database.Conn(), token)
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNameRequired)
}

func Test__DeleteMCPAPITokensForAccount(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()
	otherAccount, err := models.CreateAccountInTransaction(tx, support.RandomName("user"), support.RandomName("account")+"@example.com")
	require.NoError(t, err)
	otherUser, err := models.CreateUserInTransaction(tx, r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)

	own := models.NewMCPAPIToken(r.User, r.Organization.ID, uuid.New(), "Own", "http://localhost:8000/mcp", "hash-own-mcp", nil)
	require.NoError(t, models.CreateMCPAPIToken(tx, own))
	other := models.NewMCPAPIToken(otherUser.ID, r.Organization.ID, uuid.New(), "Other", "http://localhost:8000/mcp", "hash-other-mcp", nil)
	require.NoError(t, models.CreateMCPAPIToken(tx, other))

	require.NoError(t, models.DeleteMCPAPITokensForAccount(tx, r.Account.ID))

	_, err = models.FindMCPAPITokenByHash(tx, "hash-own-mcp")
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNotFound)
	found, err := models.FindMCPAPITokenByHash(tx, "hash-other-mcp")
	require.NoError(t, err)
	assert.Equal(t, other.ID, found.ID)
}

func Test__FactorySoftDeleteRemovesMCPAPITokens(t *testing.T) {
	r := support.Setup(t)
	tx := database.Conn()
	factory, err := models.CreateFactory(tx, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(tx, r.Organization.ID, support.RandomName("other"), "", "")
	require.NoError(t, err)

	deletedHash := support.RandomName("hash-deleted")
	keptHash := support.RandomName("hash-kept")
	own := models.NewMCPAPIToken(r.User, r.Organization.ID, factory.ID, "Own", "http://localhost:8000/mcp", deletedHash, nil)
	require.NoError(t, models.CreateMCPAPIToken(tx, own))
	kept := models.NewMCPAPIToken(r.User, r.Organization.ID, other.ID, "Kept", "http://localhost:8000/mcp", keptHash, nil)
	require.NoError(t, models.CreateMCPAPIToken(tx, kept))

	require.NoError(t, factory.SoftDelete(tx))

	_, err = models.FindMCPAPITokenByHash(tx, deletedHash)
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNotFound)
	found, err := models.FindMCPAPITokenByHash(tx, keptHash)
	require.NoError(t, err)
	assert.Equal(t, kept.ID, found.ID)
}

func Test__FactoryResourceCleanerRemovesLeftoverMCPAPITokens(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	require.NoError(t, factory.SoftDelete(db))

	leftoverHash := support.RandomName("hash-leftover")
	leftover := models.NewMCPAPIToken(r.User, r.Organization.ID, factory.ID, "Leftover", "http://localhost:8000/mcp", leftoverHash, nil)
	require.NoError(t, models.CreateMCPAPIToken(db, leftover))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, complete, cleanErr := models.NewFactoryResourceCleaner(tx, factory).WithLimit(500).Run()
		require.NoError(t, cleanErr)
		assert.True(t, complete)
		return nil
	}))

	_, err = models.FindMCPAPITokenByHash(db, leftoverHash)
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNotFound)
}
