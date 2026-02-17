package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListUsers(t *testing.T) {
	r := support.Setup(t)

	resp, err := ListUsers(context.Background(), models.DomainTypeOrganization, r.Organization.ID.String(), false, r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 1)

	for _, user := range resp.Users {
		assert.NotEmpty(t, user.Metadata.Id)
		assert.NotEmpty(t, user.Status.RoleAssignments)
		for _, roleAssignment := range user.Status.RoleAssignments {
			assert.NotEmpty(t, roleAssignment.RoleName)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION, roleAssignment.DomainType)
			assert.Equal(t, r.Organization.ID.String(), roleAssignment.DomainId)
		}
	}
}
