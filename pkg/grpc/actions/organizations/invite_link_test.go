package organizations

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetInviteLink(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns existing invite link for organization", func(t *testing.T) {
		response, err := GetInviteLink(r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.InviteLink)
		assert.Equal(t, r.Organization.ID.String(), response.InviteLink.OrganizationId)
		assert.NotEmpty(t, response.InviteLink.Token)
		assert.True(t, response.InviteLink.Enabled)
	})

	t.Run("creates invite link when none exists for organization", func(t *testing.T) {
		// Use an org UUID that has no existing invite link
		newOrgID := uuid.New()

		response, err := GetInviteLink(newOrgID.String())
		require.NoError(t, err)
		require.NotNil(t, response.InviteLink)
		assert.Equal(t, newOrgID.String(), response.InviteLink.OrganizationId)
		assert.NotEmpty(t, response.InviteLink.Token)
		assert.True(t, response.InviteLink.Enabled)

		// Calling it again returns the same link
		response2, err := GetInviteLink(newOrgID.String())
		require.NoError(t, err)
		assert.Equal(t, response.InviteLink.Token, response2.InviteLink.Token)
	})
}

func Test__UpdateInviteLink(t *testing.T) {
	r := support.Setup(t)

	t.Run("disables invite link", func(t *testing.T) {
		response, err := UpdateInviteLink(r.Organization.ID.String(), false)
		require.NoError(t, err)
		require.NotNil(t, response.InviteLink)
		assert.False(t, response.InviteLink.Enabled)
		assert.Equal(t, r.Organization.ID.String(), response.InviteLink.OrganizationId)
	})

	t.Run("re-enables invite link", func(t *testing.T) {
		// Disable first, then re-enable
		_, err := UpdateInviteLink(r.Organization.ID.String(), false)
		require.NoError(t, err)

		response, err := UpdateInviteLink(r.Organization.ID.String(), true)
		require.NoError(t, err)
		require.NotNil(t, response.InviteLink)
		assert.True(t, response.InviteLink.Enabled)
	})

	t.Run("invite link not found -> error", func(t *testing.T) {
		_, err := UpdateInviteLink(uuid.New().String(), true)
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invite link not found", s.Message())
	})
}

func Test__ResetInviteLink(t *testing.T) {
	r := support.Setup(t)

	t.Run("resets token to a new value", func(t *testing.T) {
		original, err := models.FindInviteLinkByOrganizationID(r.Organization.ID.String())
		require.NoError(t, err)

		response, err := ResetInviteLink(r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response.InviteLink)
		assert.Equal(t, r.Organization.ID.String(), response.InviteLink.OrganizationId)
		assert.NotEqual(t, original.Token.String(), response.InviteLink.Token, "token should change after reset")
	})

	t.Run("invite link not found -> error", func(t *testing.T) {
		_, err := ResetInviteLink(uuid.New().String())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "invite link not found", s.Message())
	})
}
