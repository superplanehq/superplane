package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListUsers(t *testing.T) {
	r := support.Setup(t)

	ctx := authentication.SetOrganizationIdInMetadata(context.Background(), r.Organization.ID.String())
	resp, err := ListUsers(ctx, models.DomainTypeCanvas, "*", r.AuthService)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Len(t, resp.Users, 1)

	for _, user := range resp.Users {
		assert.NotEmpty(t, user.Metadata.Id)
		assert.NotEmpty(t, user.Status.RoleAssignments)
		for _, roleAssignment := range user.Status.RoleAssignments {
			assert.NotEmpty(t, roleAssignment.RoleName)
			assert.Equal(t, pbAuth.DomainType_DOMAIN_TYPE_CANVAS, roleAssignment.DomainType)
			assert.Equal(t, "*", roleAssignment.DomainId)
		}
	}
}
