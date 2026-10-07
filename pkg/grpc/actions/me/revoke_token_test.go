package me

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__RevokeToken(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	kept := models.NewUserAPIToken(r.User, "Kept token", "hash-kept")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), kept))

	revoked := models.NewUserAPIToken(r.User, "Revoked token", "hash-revoked")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), revoked))

	_, err := RevokeToken(ctx, &pb.RevokeTokenRequest{Id: revoked.ID.String()})
	require.NoError(t, err)

	tokens, err := models.ListUserAPITokens(database.Conn(), r.User)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, kept.ID, tokens[0].ID)
}

func Test__RevokeToken_NotFound(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	_, err := RevokeToken(ctx, &pb.RevokeTokenRequest{Id: "00000000-0000-0000-0000-000000000000"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))
}

func Test__RevokeToken_CannotRevokeAnotherUsersToken(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	otherAccount, err := models.CreateAccountInTransaction(database.Conn(), support.RandomName("user"), support.RandomName("account")+"@example.com")
	require.NoError(t, err)

	otherUser, err := models.CreateUserInTransaction(database.Conn(), r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)

	otherToken := models.NewUserAPIToken(otherUser.ID, "Not mine", "hash-not-mine")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), otherToken))

	_, err = RevokeToken(ctx, &pb.RevokeTokenRequest{Id: otherToken.ID.String()})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, grpcerrors.Code(err))

	tokens, err := models.ListUserAPITokens(database.Conn(), otherUser.ID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
}
