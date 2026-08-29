package me

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListTokens(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	response, err := ListTokens(ctx)
	require.NoError(t, err)
	assert.Empty(t, response.Tokens)

	first := models.NewUserAPIToken(r.User, "First token", "hash-first")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), first))

	second := models.NewUserAPIToken(r.User, "Second token", "hash-second")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), second))

	response, err = ListTokens(ctx)
	require.NoError(t, err)
	require.Len(t, response.Tokens, 2)

	for _, token := range response.Tokens {
		assert.NotContains(t, []string{"hash-first", "hash-second"}, token.Id, "list must never return the token hash")
		assert.Nil(t, token.LastUsedAt)
	}
}

func Test__ListTokens_OnlyReturnsCallersOwnTokens(t *testing.T) {
	r := support.Setup(t)
	ctx := meContext(r)

	otherAccount, err := models.CreateAccountInTransaction(database.Conn(), support.RandomName("user"), support.RandomName("account")+"@example.com")
	require.NoError(t, err)

	otherUser, err := models.CreateUserInTransaction(database.Conn(), r.Organization.ID, otherAccount.ID, otherAccount.Email, otherAccount.Name)
	require.NoError(t, err)

	ownToken := models.NewUserAPIToken(r.User, "My token", "hash-own")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), ownToken))

	otherToken := models.NewUserAPIToken(otherUser.ID, "Other user's token", "hash-other")
	require.NoError(t, models.CreateUserAPIToken(database.Conn(), otherToken))

	response, err := ListTokens(ctx)
	require.NoError(t, err)
	require.Len(t, response.Tokens, 1)
	assert.Equal(t, ownToken.ID.String(), response.Tokens[0].Id)
}
