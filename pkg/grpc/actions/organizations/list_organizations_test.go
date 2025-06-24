package organizations

import (
	"context"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/auth"
	"github.com/superplanehq/superplane/pkg/models"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
)

func Test__ListOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	userID := uuid.New()
	authService := auth.SetupTestAuthService(t)
	ctx := context.Background()
	ctx = authentication.SetUserIdInMetadata(ctx, userID.String())

	organization, err := models.CreateOrganization(userID, "test-org", "Test Organization")
	require.NoError(t, err)
	authService.SetupOrganizationRoles(organization.ID.String())
	authService.AssignRole(userID.String(), authorization.RoleOrgOwner, organization.ID.String(), authorization.DomainOrg)

	res, err := ListOrganizations(ctx, &protos.ListOrganizationsRequest{}, authService)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Organizations, 1)
	require.NotNil(t, res.Organizations[0].Metadata)
	assert.Equal(t, organization.ID.String(), res.Organizations[0].Metadata.Id)
	assert.Equal(t, organization.Name, res.Organizations[0].Metadata.Name)
	assert.Equal(t, organization.DisplayName, res.Organizations[0].Metadata.DisplayName)
	assert.Equal(t, organization.CreatedBy.String(), res.Organizations[0].Metadata.CreatedBy)
	assert.NotNil(t, res.Organizations[0].Metadata.CreatedAt)
	assert.NotNil(t, res.Organizations[0].Metadata.UpdatedAt)
}
