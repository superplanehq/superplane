package organizations

import (
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__GetInviteLink(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns existing invite link", func(t *testing.T) {
		existing, err := models.FindInviteLinkByOrganizationID(r.Organization.ID.String())
		require.NoError(t, err)

		response, err := GetInviteLink(r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.InviteLink)
		assert.Equal(t, existing.ID.String(), response.InviteLink.Id)
		assert.Equal(t, r.Organization.ID.String(), response.InviteLink.OrganizationId)
		assert.Equal(t, existing.Token.String(), response.InviteLink.Token)
	})

	t.Run("invalid organization id returns InvalidArgument", func(t *testing.T) {
		_, err := GetInviteLink("not-a-uuid")
		require.Error(t, err)
		s, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("creates an invite link when one does not yet exist", func(t *testing.T) {
		org := support.CreateOrganization(t, r, r.User)

		// Simulate an organization that somehow has no invite link by
		// deleting the one created during CreateOrganization.
		require.NoError(t, database.Conn().
			Where("organization_id = ?", org.ID).
			Delete(&models.OrganizationInviteLink{}).
			Error)

		response, err := GetInviteLink(org.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.InviteLink)
		assert.Equal(t, org.ID.String(), response.InviteLink.OrganizationId)
		assert.True(t, response.InviteLink.Enabled)
		_, parseErr := uuid.Parse(response.InviteLink.Token)
		assert.NoError(t, parseErr)
	})

	t.Run("concurrent first-time fetches do not 500 on the unique constraint", func(t *testing.T) {
		org := support.CreateOrganization(t, r, r.User)

		require.NoError(t, database.Conn().
			Where("organization_id = ?", org.ID).
			Delete(&models.OrganizationInviteLink{}).
			Error)

		const concurrency = 8
		errs := make([]error, concurrency)
		tokens := make([]string, concurrency)
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func(i int) {
				defer wg.Done()
				resp, err := GetInviteLink(org.ID.String())
				errs[i] = err
				if resp != nil && resp.InviteLink != nil {
					tokens[i] = resp.InviteLink.Token
				}
			}(i)
		}
		wg.Wait()

		for i, err := range errs {
			require.NoError(t, err, "call %d returned an error", i)
		}

		first := tokens[0]
		require.NotEmpty(t, first)
		for i, tok := range tokens {
			assert.Equal(t, first, tok, "call %d returned a different token", i)
		}
	})
}
