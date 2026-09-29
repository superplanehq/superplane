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
	"gorm.io/datatypes"
)

func TestListMCPOAuthRefreshTokensForFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()

	newer := insertRefreshToken(t, r, factory.ID, "superplane-local", now.Add(time.Hour), now.Add(-time.Minute))
	older := insertRefreshToken(t, r, factory.ID, "other-client", now.Add(time.Hour), now.Add(-time.Hour))
	insertRefreshToken(t, r, factory.ID, "expired-client", now.Add(-time.Minute), now.Add(-2*time.Hour))
	insertRefreshToken(t, r, otherFactory.ID, "other-factory", now.Add(time.Hour), now)

	tokens, err := models.ListMCPOAuthRefreshTokensForFactory(db, r.Organization.ID, factory.ID, now)
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	assert.Equal(t, newer.ID, tokens[0].ID)
	assert.Equal(t, older.ID, tokens[1].ID)
}

func TestFindMCPOAuthRefreshTokenForFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertRefreshToken(t, r, factory.ID, "superplane-local", now.Add(time.Hour), now)

	found, err := models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, token.ID)
	require.NoError(t, err)
	assert.Equal(t, token.ID, found.ID)

	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, otherFactory.ID, token.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)

	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, uuid.New())
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)
}

func TestListMCPOAuthClientsByClientIDs(t *testing.T) {
	support.Setup(t)
	db := database.DB(t.Context())
	created, err := models.CreateMCPOAuthClient(db, "cursor-dcr", "Cursor Desktop", []string{"cursor://callback"})
	require.NoError(t, err)

	clients, err := models.ListMCPOAuthClientsByClientIDs(db, []string{created.ClientID, "missing"})
	require.NoError(t, err)
	require.Len(t, clients, 1)
	assert.Equal(t, "Cursor Desktop", clients[0].ClientName)

	empty, err := models.ListMCPOAuthClientsByClientIDs(db, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func insertRefreshToken(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	clientID string,
	expiresAt time.Time,
	createdAt time.Time,
) *models.MCPOAuthRefreshToken {
	t.Helper()
	token := &models.MCPOAuthRefreshToken{
		TokenHash:      uuid.NewString(),
		ClientID:       clientID,
		UserID:         r.User,
		OrganizationID: r.Organization.ID,
		FactoryID:      factoryID,
		Resource:       "http://localhost:8000/mcp",
		Scopes:         datatypes.NewJSONSlice([]string{"work_orders:read"}),
		ExpiresAt:      expiresAt,
		CreatedAt:      createdAt,
	}
	require.NoError(t, models.CreateMCPOAuthRefreshToken(database.DB(t.Context()), token))
	return token
}

func TestHasMCPOAuthRefreshTokenForClient(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	insertRefreshToken(t, r, factory.ID, "superplane-local", now.Add(time.Hour), now)

	ok, err := models.HasMCPOAuthRefreshTokenForClient(db, r.Organization.ID, factory.ID, r.User, "superplane-local", now)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = models.HasMCPOAuthRefreshTokenForClient(db, r.Organization.ID, factory.ID, r.User, "other-client", now)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDeleteMCPOAuthRefreshTokensForClient(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	first := insertRefreshToken(t, r, factory.ID, "superplane-local", now.Add(time.Hour), now)
	insertRefreshToken(t, r, factory.ID, "superplane-local", now.Add(time.Hour), now.Add(time.Second))
	other := insertRefreshToken(t, r, factory.ID, "other-client", now.Add(time.Hour), now)

	require.NoError(t, models.DeleteMCPOAuthRefreshTokensForClient(db, r.Organization.ID, factory.ID, r.User, "superplane-local"))
	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, first.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)
	found, err := models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, other.ID, found.ID)
}
