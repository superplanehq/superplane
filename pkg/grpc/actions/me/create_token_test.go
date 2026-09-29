package me

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func Test__CreateToken(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	t.Run("returns the plaintext once and stores only the hash", func(t *testing.T) {
		response, err := CreateToken(ctx, &pb.CreateTokenRequest{Name: "CI token"})
		require.NoError(t, err)
		require.NotNil(t, response.Token)
		assert.Equal(t, "CI token", response.Token.Name)
		assert.NotEmpty(t, response.Plaintext)

		tokens, err := models.ListUserAPITokens(database.Conn(), r.User)
		require.NoError(t, err)
		require.Len(t, tokens, 1)
		assert.NotEqual(t, response.Plaintext, tokens[0].TokenHash)
		assert.NotEmpty(t, tokens[0].TokenHash)
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		_, err := CreateToken(ctx, &pb.CreateTokenRequest{Name: "   "})
		require.Error(t, err)
		assert.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})

	t.Run("a second token does not invalidate the first", func(t *testing.T) {
		first, err := CreateToken(ctx, &pb.CreateTokenRequest{Name: "First"})
		require.NoError(t, err)

		second, err := CreateToken(ctx, &pb.CreateTokenRequest{Name: "Second"})
		require.NoError(t, err)

		assert.NotEqual(t, first.Token.Id, second.Token.Id)

		_, err = models.FindUserAPITokenByHash(database.Conn(), first.Token.Id)
		assert.Error(t, err, "lookup by id (not hash) should not match")
	})

	t.Run("API keys cannot create personal tokens", func(t *testing.T) {
		apiKey, err := models.CreateAPIKey(database.Conn(), r.Organization.ID, "bot", nil, r.User, nil, nil)
		require.NoError(t, err)

		apiKeyCtx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				"x-organization-id", r.Organization.ID.String(),
				"x-user-id", apiKey.ID.String(),
			),
		)

		_, err = CreateToken(apiKeyCtx, &pb.CreateTokenRequest{Name: "not allowed"})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, grpcerrors.Code(err))
	})
}

func meContext(r *support.ResourceRegistry) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-organization-id", r.Organization.ID.String(),
			"x-user-id", r.User.String(),
		),
	)
}
