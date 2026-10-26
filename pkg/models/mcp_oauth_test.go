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

func TestDeleteMCPOAuthCodesForClient(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	keep := insertAuthCode(t, r, factory.ID, "other-client", now.Add(time.Minute))
	drop := insertAuthCode(t, r, factory.ID, "superplane-local", now.Add(time.Minute))

	require.NoError(t, models.DeleteMCPOAuthCodesForClient(db, r.Organization.ID, factory.ID, r.User, "superplane-local"))
	_, err = models.FindMCPOAuthCode(db, drop.CodeHash, now)
	assert.ErrorIs(t, err, models.ErrMCPOAuthCodeNotFound)
	found, err := models.FindMCPOAuthCode(db, keep.CodeHash, now)
	require.NoError(t, err)
	assert.Equal(t, keep.ID, found.ID)
}

func insertAuthCode(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryID uuid.UUID,
	clientID string,
	expiresAt time.Time,
) *models.MCPOAuthCode {
	t.Helper()
	code := &models.MCPOAuthCode{
		CodeHash:            uuid.NewString(),
		ClientID:            clientID,
		RedirectURI:         "http://localhost:8787/callback",
		Resource:            "http://localhost:8000/mcp",
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		UserID:              r.User,
		OrganizationID:      r.Organization.ID,
		FactoryID:           factoryID,
		Scopes:              datatypes.NewJSONSlice([]string{"work_orders:read"}),
		ExpiresAt:           expiresAt,
	}
	require.NoError(t, models.CreateMCPOAuthCode(database.DB(t.Context()), code))
	return code
}

func TestListMCPOAuthRefreshTokensIncludesEveryGrantedWorkspace(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertRefreshTokenWithWorkspaces(t, r, []uuid.UUID{second.ID, first.ID}, "superplane-local", now.Add(time.Hour), now)
	old := insertRefreshToken(t, r, other.ID, "old-client", now.Add(time.Hour), now)
	require.NoError(t, db.Exec("UPDATE mcp_oauth_refresh_tokens SET factory_ids = NULL WHERE id = ?", old.ID).Error)

	for _, factoryID := range []uuid.UUID{first.ID, second.ID} {
		listed, err := models.ListMCPOAuthRefreshTokensForFactory(db, r.Organization.ID, factoryID, now)
		require.NoError(t, err)
		require.Len(t, listed, 1)
		assert.Equal(t, token.ID, listed[0].ID)
		assert.Equal(t, int32(2), listed[0].WorkspaceCount())
	}
	listed, err := models.ListMCPOAuthRefreshTokensForFactory(db, r.Organization.ID, other.ID, now)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, old.ID, listed[0].ID)
	assert.Equal(t, []uuid.UUID{other.ID}, listed[0].WorkspaceIDs())
}

func TestDeleteMCPOAuthRefreshTokensForWorkspaceSetKeepsOtherSets(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	shared := insertRefreshTokenWithWorkspaces(t, r, []uuid.UUID{first.ID, second.ID}, "superplane-local", now.Add(time.Hour), now)
	narrow := insertRefreshToken(t, r, first.ID, "superplane-local", now.Add(time.Hour), now.Add(time.Second))

	require.NoError(t, models.DeleteMCPOAuthRefreshTokensForWorkspaceSet(db, r.Organization.ID, r.User, "superplane-local", []uuid.UUID{second.ID, first.ID}))
	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, first.ID, shared.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)
	found, err := models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, first.ID, narrow.ID)
	require.NoError(t, err)
	assert.Equal(t, narrow.ID, found.ID)
}

func insertRefreshTokenWithWorkspaces(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryIDs []uuid.UUID,
	clientID string,
	expiresAt time.Time,
	createdAt time.Time,
) *models.MCPOAuthRefreshToken {
	t.Helper()
	ids := make([]string, len(factoryIDs))
	for i, id := range factoryIDs {
		ids[i] = id.String()
	}
	token := &models.MCPOAuthRefreshToken{
		TokenHash:      uuid.NewString(),
		ClientID:       clientID,
		UserID:         r.User,
		OrganizationID: r.Organization.ID,
		FactoryIDs:     datatypes.NewJSONSlice(ids),
		Resource:       "http://localhost:8000/mcp",
		Scopes:         datatypes.NewJSONSlice([]string{"work_orders:read"}),
		ExpiresAt:      expiresAt,
		CreatedAt:      createdAt,
	}
	require.NoError(t, models.CreateMCPOAuthRefreshToken(database.DB(t.Context()), token))
	return token
}
