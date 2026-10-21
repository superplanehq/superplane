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
	"gorm.io/gorm"
)

func TestListMCPOAuthRefreshTokensForFactory(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()

	newer := insertRefreshToken(t, r, []uuid.UUID{factory.ID}, "superplane-local", now.Add(time.Hour), now.Add(-time.Minute))
	older := insertRefreshToken(t, r, []uuid.UUID{factory.ID}, "other-client", now.Add(time.Hour), now.Add(-time.Hour))
	insertRefreshToken(t, r, []uuid.UUID{factory.ID}, "expired-client", now.Add(-time.Minute), now.Add(-2*time.Hour))
	insertRefreshToken(t, r, []uuid.UUID{otherFactory.ID}, "other-factory", now.Add(time.Hour), now)

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
	token := insertRefreshToken(t, r, []uuid.UUID{factory.ID}, "superplane-local", now.Add(time.Hour), now)

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
	factoryIDs []uuid.UUID,
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
		FactoryIDs:     models.NewMCPOAuthFactoryIDs(factoryIDs),
		Resource:       "http://localhost:8000/mcp",
		Scopes:         datatypes.NewJSONSlice([]string{"work_orders:read"}),
		ExpiresAt:      expiresAt,
		CreatedAt:      createdAt,
	}
	require.NoError(t, models.CreateMCPOAuthRefreshToken(database.DB(t.Context()), token))
	return token
}

func TestListMCPOAuthRefreshTokensForFactoryMatchesMultiWorkspaceGrants(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	third, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertRefreshToken(t, r, []uuid.UUID{first.ID, second.ID}, "superplane-local", now.Add(time.Hour), now)

	for _, factory := range []*models.Factory{first, second} {
		tokens, err := models.ListMCPOAuthRefreshTokensForFactory(db, r.Organization.ID, factory.ID, now)
		require.NoError(t, err)
		require.Len(t, tokens, 1)
		assert.Equal(t, token.ID, tokens[0].ID)
	}

	tokens, err := models.ListMCPOAuthRefreshTokensForFactory(db, r.Organization.ID, third.ID, now)
	require.NoError(t, err)
	assert.Empty(t, tokens)

	tokens, err = models.ListMCPOAuthRefreshTokensForFactory(db, uuid.New(), first.ID, now)
	require.NoError(t, err)
	assert.Empty(t, tokens)
}

func TestHasMCPOAuthRefreshTokenForClient(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	third, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	insertRefreshToken(t, r, []uuid.UUID{first.ID, second.ID}, "superplane-local", now.Add(time.Hour), now)

	tests := []struct {
		name       string
		orgID      uuid.UUID
		factoryIDs []uuid.UUID
		clientID   string
		want       bool
	}{
		{"one granted workspace", r.Organization.ID, []uuid.UUID{first.ID}, "superplane-local", true},
		{"every granted workspace", r.Organization.ID, []uuid.UUID{second.ID, first.ID}, "superplane-local", true},
		{"one workspace not granted", r.Organization.ID, []uuid.UUID{first.ID, third.ID}, "superplane-local", false},
		{"other client", r.Organization.ID, []uuid.UUID{first.ID}, "other-client", false},
		{"other organization", uuid.New(), []uuid.UUID{first.ID}, "superplane-local", false},
		{"no workspace", r.Organization.ID, nil, "superplane-local", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, err := models.HasMCPOAuthRefreshTokenForClient(db, test.orgID, test.factoryIDs, r.User, test.clientID, now)
			require.NoError(t, err)
			assert.Equal(t, test.want, ok)
		})
	}
}

func TestRemoveFactoryFromMCPOAuthGrant(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	both := []uuid.UUID{first.ID, second.ID}
	shared := insertRefreshToken(t, r, both, "superplane-local", now.Add(time.Hour), now)
	onlyFirst := insertRefreshToken(t, r, []uuid.UUID{first.ID}, "superplane-local", now.Add(time.Hour), now)
	otherClient := insertRefreshToken(t, r, both, "other-client", now.Add(time.Hour), now)
	sharedCode := insertAuthCode(t, r, both, "superplane-local", now.Add(time.Minute))
	otherClientCode := insertAuthCode(t, r, both, "other-client", now.Add(time.Minute))

	require.NoError(t, models.RemoveFactoryFromMCPOAuthGrant(db, r.Organization.ID, first.ID, r.User, "superplane-local"))

	var kept models.MCPOAuthRefreshToken
	require.NoError(t, db.First(&kept, "id = ?", shared.ID).Error)
	assert.Equal(t, []uuid.UUID{second.ID}, kept.FactoryUUIDs())
	assert.ErrorIs(t, db.First(&models.MCPOAuthRefreshToken{}, "id = ?", onlyFirst.ID).Error, gorm.ErrRecordNotFound)
	var untouched models.MCPOAuthRefreshToken
	require.NoError(t, db.First(&untouched, "id = ?", otherClient.ID).Error)
	assert.Equal(t, both, untouched.FactoryUUIDs())

	code, err := models.FindMCPOAuthCode(db, sharedCode.CodeHash, now)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{second.ID}, code.FactoryUUIDs())
	code, err = models.FindMCPOAuthCode(db, otherClientCode.CodeHash, now)
	require.NoError(t, err)
	assert.Equal(t, both, code.FactoryUUIDs())

	require.NoError(t, models.RemoveFactoryFromMCPOAuthGrant(db, r.Organization.ID, second.ID, r.User, "superplane-local"))
	assert.ErrorIs(t, db.First(&models.MCPOAuthRefreshToken{}, "id = ?", shared.ID).Error, gorm.ErrRecordNotFound)
	_, err = models.FindMCPOAuthCode(db, sharedCode.CodeHash, now)
	assert.ErrorIs(t, err, models.ErrMCPOAuthCodeNotFound)
}

func TestRemoveFactoryFromMCPOAuthGrantStaysInOrganization(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertRefreshToken(t, r, []uuid.UUID{factory.ID}, "superplane-local", now.Add(time.Hour), now)

	require.NoError(t, models.RemoveFactoryFromMCPOAuthGrant(db, uuid.New(), factory.ID, r.User, "superplane-local"))

	found, err := models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, token.ID)
	require.NoError(t, err)
	assert.Equal(t, token.ID, found.ID)
}

func insertAuthCode(
	t *testing.T,
	r *support.ResourceRegistry,
	factoryIDs []uuid.UUID,
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
		FactoryIDs:          models.NewMCPOAuthFactoryIDs(factoryIDs),
		Scopes:              datatypes.NewJSONSlice([]string{"work_orders:read"}),
		ExpiresAt:           expiresAt,
	}
	require.NoError(t, models.CreateMCPOAuthCode(database.DB(t.Context()), code))
	return code
}
