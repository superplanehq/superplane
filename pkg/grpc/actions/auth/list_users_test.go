package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListUsers(t *testing.T) {
	r := support.Setup(t)

	t.Run("returns role assignments", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), true, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)

		for _, user := range resp.Users {
			assert.NotEmpty(t, user.Metadata.Id)
			assert.NotEmpty(t, user.Status.Roles)
			assert.NotEmpty(t, user.Status.AccountProviders)
			for _, role := range user.Status.Roles {
				assert.NotEmpty(t, role)
				assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, role.DomainType)
				assert.Equal(t, r.Organization.ID.String(), role.DomainId)
			}
		}
	})

	t.Run("returns no role assignments", func(t *testing.T) {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), false, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Len(t, resp.Users, 1)
		assert.Empty(t, resp.Users[0].Status.Roles)
	})
}

func TestListUsersSeatMetadata(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(models.BillingEnabledEnv, "yes")
	limit := 3
	require.NoError(t, models.SetOrganizationSeatLimitOverride(database.Conn(), r.Organization.ID, &limit))
	for _, roles := range []bool{false, true} {
		resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), roles, r.AuthService)
		require.NoError(t, err)
		require.NotNil(t, resp.SeatLimit)
		require.EqualValues(t, 3, *resp.SeatLimit)
		require.Len(t, resp.Users, 1)
	}
	require.NoError(t, models.SetOrganizationSeatLimitOverride(database.Conn(), r.Organization.ID, nil))
	resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), false, r.AuthService)
	require.NoError(t, err)
	require.Nil(t, resp.SeatLimit)
}
