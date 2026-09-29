package factories

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/datatypes"

	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

func Test__ListFactoryMCPClients(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertFactoryMCPRefreshToken(t, r, factory.ID, localMCPClientID, now.Add(time.Hour), now)

	response, err := ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, response.GetClients(), 1)
	client := response.GetClients()[0]
	assert.Equal(t, token.ID.String(), client.GetId())
	assert.Equal(t, localMCPClientName, client.GetClientName())
	assert.Equal(t, r.User.String(), client.GetUserId())
	assert.Equal(t, r.UserModel.Name, client.GetUserName())
	assert.Equal(t, r.UserModel.GetEmail(), client.GetUserEmail())
}

func Test__ListFactoryMCPClientsUsesRegisteredClientName(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	_, err = models.CreateMCPOAuthClient(db, "cursor-dcr", "Cursor Desktop", []string{"cursor://callback"})
	require.NoError(t, err)
	now := time.Now()
	insertFactoryMCPRefreshToken(t, r, factory.ID, "cursor-dcr", now.Add(time.Hour), now)

	response, err := ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, response.GetClients(), 1)
	assert.Equal(t, "Cursor Desktop", response.GetClients()[0].GetClientName())
}

func Test__ListFactoryMCPClientsRequiresFeature(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, grpcerrors.Code(err))
}

func Test__RevokeFactoryMCPClient(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	otherFactory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	token := insertFactoryMCPRefreshToken(t, r, factory.ID, localMCPClientID, now.Add(time.Hour), now)

	_, err = RevokeFactoryMCPClient(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPClientRequest{
		FactoryId: otherFactory.ID.String(),
		ClientId:  token.ID.String(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	_, err = RevokeFactoryMCPClient(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPClientRequest{
		FactoryId: factory.ID.String(),
		ClientId:  token.ID.String(),
	})
	require.NoError(t, err)

	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, token.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)

	listed, err := ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Empty(t, listed.GetClients())
}

func Test__RevokeFactoryMCPClientDeletesRotatedTokens(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	now := time.Now()
	listed := insertFactoryMCPRefreshToken(t, r, factory.ID, localMCPClientID, now.Add(time.Hour), now)
	rotated := insertFactoryMCPRefreshToken(t, r, factory.ID, localMCPClientID, now.Add(time.Hour), now.Add(time.Second))
	other := insertFactoryMCPRefreshToken(t, r, factory.ID, "other-client", now.Add(time.Hour), now)

	_, err = RevokeFactoryMCPClient(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPClientRequest{
		FactoryId: factory.ID.String(),
		ClientId:  listed.ID.String(),
	})
	require.NoError(t, err)

	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, listed.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)
	_, err = models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, rotated.ID)
	assert.ErrorIs(t, err, models.ErrMCPOAuthRefreshNotFound)
	found, err := models.FindMCPOAuthRefreshTokenForFactory(db, r.Organization.ID, factory.ID, other.ID)
	require.NoError(t, err)
	assert.Equal(t, other.ID, found.ID)
}

func Test__RevokeFactoryMCPClientRejectsInvalidID(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = RevokeFactoryMCPClient(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPClientRequest{
		FactoryId: factory.ID.String(),
		ClientId:  "not-a-uuid",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
}

func insertFactoryMCPRefreshToken(
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
	require.NoError(t, models.CreateMCPOAuthRefreshToken(database.Conn(), token))
	return token
}
