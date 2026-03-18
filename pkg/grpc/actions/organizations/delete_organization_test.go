package organizations

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func Test__DeleteOrganization(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("organization does not exist -> error", func(t *testing.T) {
		_, err := DeleteOrganization(ctx, r.AuthService, uuid.New().String())
		require.Error(t, err)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
		assert.Equal(t, "organization not found", s.Message())
	})

	t.Run("unauthenticated user -> error", func(t *testing.T) {
		_, err := DeleteOrganization(context.Background(), r.AuthService, r.Organization.ID.String())
		require.Error(t, err)
		assert.ErrorContains(t, err, "user not authenticated")
	})

	t.Run("organization is deleted", func(t *testing.T) {
		response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, response)

		_, err = models.FindOrganizationByID(r.Organization.ID.String())
		assert.Error(t, err)
	})
}

func Test__DeleteOrganization_RenamesOrg(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	orgName := r.Organization.Name

	response, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)
	require.NotNil(t, response)

	var org models.Organization
	err = database.Conn().Unscoped().Where("id = ?", r.Organization.ID).First(&org).Error
	require.NoError(t, err)
	assert.True(t, org.DeletedAt.Valid)
	assert.Contains(t, org.Name, orgName)
	assert.Contains(t, org.Name, "(deleted-")
}

func Test__DeleteOrganization_CascadesCanvases(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	canvas, _ := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID: "node-1",
				Type:   models.NodeTypeComponent,
				Ref: datatypes.NewJSONType(models.NodeRef{
					Component: &models.ComponentRef{Name: "noop"},
				}),
			},
		},
		[]models.Edge{},
	)

	_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	_, err = models.FindCanvas(r.Organization.ID, canvas.ID)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var deletedCanvas models.Canvas
	err = database.Conn().Unscoped().Where("id = ?", canvas.ID).First(&deletedCanvas).Error
	require.NoError(t, err)
	assert.True(t, deletedCanvas.DeletedAt.Valid)
	assert.Contains(t, deletedCanvas.Name, "(deleted-")
}

func Test__DeleteOrganization_CascadesIntegrations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		"github",
		"my-integration",
		map[string]any{"key": "value"},
	)
	require.NoError(t, err)

	_, err = DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var deletedIntegration models.Integration
	err = database.Conn().Unscoped().Where("id = ?", integration.ID).First(&deletedIntegration).Error
	require.NoError(t, err)
	assert.True(t, deletedIntegration.DeletedAt.Valid)
	assert.Contains(t, deletedIntegration.InstallationName, "(deleted-")
}

func Test__DeleteOrganization_CascadesUsers(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	extraUser := support.CreateUser(t, r, r.Organization.ID)

	_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var user models.User
	err = database.Conn().Unscoped().Where("id = ?", extraUser.ID).First(&user).Error
	require.NoError(t, err)
	assert.True(t, user.DeletedAt.Valid)
}

func Test__DeleteOrganization_DeletesBlueprints(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	blueprint := support.CreateBlueprint(t, r.Organization.ID, []models.Node{}, []models.Edge{}, nil)

	_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var count int64
	database.Conn().Model(&models.Blueprint{}).Where("id = ?", blueprint.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func Test__DeleteOrganization_DeletesInvitations(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err := models.CreateInvitation(r.Organization.ID, r.User, "test-invite@example.com", models.InvitationStatePending)
	require.NoError(t, err)

	_, err = DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var count int64
	database.Conn().Model(&models.OrganizationInvitation{}).Where("organization_id = ?", r.Organization.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func Test__DeleteOrganization_DeletesInviteLinks(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	_, err := DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var count int64
	database.Conn().Model(&models.OrganizationInviteLink{}).Where("organization_id = ?", r.Organization.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func Test__DeleteOrganization_DeletesSecrets(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	data, err := json.Marshal(map[string]string{"key": "value"})
	require.NoError(t, err)
	_, err = models.CreateSecret("test-secret", "local", r.User.String(), models.DomainTypeOrganization, r.Organization.ID, data)
	require.NoError(t, err)

	_, err = DeleteOrganization(ctx, r.AuthService, r.Organization.ID.String())
	require.NoError(t, err)

	var count int64
	database.Conn().Model(&models.Secret{}).Where("domain_type = ? AND domain_id = ?", models.DomainTypeOrganization, r.Organization.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func Test__DeleteOrganization_TransactionRollback(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("auth service failure rolls back organization soft-deletion", func(t *testing.T) {
		foundOrg, err := models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)

		mockAuth := &mockAuthService{
			Authorization: r.AuthService,
			Error:         errors.New("ooops"),
		}

		_, err = DeleteOrganization(ctx, mockAuth, r.Organization.ID.String())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete organization roles")

		foundOrg, err = models.FindOrganizationByID(r.Organization.ID.String())
		require.NoError(t, err)
		assert.False(t, foundOrg.DeletedAt.Valid)
	})
}
