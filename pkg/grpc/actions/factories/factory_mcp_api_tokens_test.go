package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func Test__CreateFactoryMCPAPIToken(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	ctx := mcpAPITokenContext(r)
	resource := "http://localhost:8000/mcp"

	first, err := CreateFactoryMCPAPIToken(ctx, r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  resource,
	})
	require.NoError(t, err)
	require.NotNil(t, first.GetToken())
	assert.True(t, len(first.GetPlaintext()) > len(models.MCPAPITokenPrefix))
	assert.Equal(t, models.MCPAPITokenPrefix, first.GetPlaintext()[:len(models.MCPAPITokenPrefix)])
	assert.Equal(t, mcpClientKindAPIToken, first.GetToken().GetKind())
	assert.Equal(t, "Build server", first.GetToken().GetClientName())

	stored, err := models.FindMCPAPITokenByHash(db, crypto.HashToken(first.GetPlaintext()))
	require.NoError(t, err)
	assert.Equal(t, crypto.HashToken(first.GetPlaintext()), stored.TokenHash)
	assert.NotEqual(t, first.GetPlaintext(), stored.TokenHash)
	assert.Equal(t, models.MCPGrantedScopes, []string(stored.Scopes))
	assert.Equal(t, resource, stored.Resource)
	assert.Equal(t, r.User, stored.UserID)

	second, err := CreateFactoryMCPAPIToken(ctx, r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  resource,
	})
	require.NoError(t, err)
	assert.NotEqual(t, first.GetToken().GetId(), second.GetToken().GetId())

	listed, err := ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, listed.GetClients(), 2)
	assert.Equal(t, mcpClientKindAPIToken, listed.GetClients()[0].GetKind())
	assert.Equal(t, "Build server", listed.GetClients()[0].GetClientName())
}

func Test__CreateFactoryMCPAPITokenRejectsEmptyNameAndAPIKey(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	_, err = CreateFactoryMCPAPIToken(mcpAPITokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "   ",
		Resource:  "http://localhost:8000/mcp",
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))

	apiKey, err := models.CreateAPIKey(db, r.Organization.ID, "bot", nil, r.User, nil, nil)
	require.NoError(t, err)
	apiKeyCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", r.Organization.ID.String(),
		"x-user-id", apiKey.ID.String(),
	))
	_, err = CreateFactoryMCPAPIToken(apiKeyCtx, r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "not allowed",
		Resource:  "http://localhost:8000/mcp",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
}

func Test__RevokeFactoryMCPAPIToken(t *testing.T) {
	r := support.Setup(t)
	require.NoError(t, models.EnableExperimentalFeature(r.Organization.ID, features.FeatureSuperPlaneMCPServer))
	db := database.DB(t.Context())
	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	other, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)
	created, err := CreateFactoryMCPAPIToken(mcpAPITokenContext(r), r.Organization.ID.String(), &pb.CreateFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		Name:      "Build server",
		Resource:  "http://localhost:8000/mcp",
	})
	require.NoError(t, err)

	_, err = RevokeFactoryMCPAPIToken(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPAPITokenRequest{
		FactoryId: other.ID.String(),
		TokenId:   created.GetToken().GetId(),
	})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	_, err = RevokeFactoryMCPAPIToken(t.Context(), r.Organization.ID.String(), &pb.RevokeFactoryMCPAPITokenRequest{
		FactoryId: factory.ID.String(),
		TokenId:   created.GetToken().GetId(),
	})
	require.NoError(t, err)

	_, err = models.FindMCPAPITokenByHash(db, crypto.HashToken(created.GetPlaintext()))
	assert.ErrorIs(t, err, models.ErrMCPAPITokenNotFound)

	listed, err := ListFactoryMCPClients(t.Context(), r.Organization.ID.String(), &pb.ListFactoryMCPClientsRequest{
		FactoryId: factory.ID.String(),
	})
	require.NoError(t, err)
	assert.Empty(t, listed.GetClients())
}

func mcpAPITokenContext(r *support.ResourceRegistry) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", r.Organization.ID.String(),
		"x-user-id", r.User.String(),
	))
}
