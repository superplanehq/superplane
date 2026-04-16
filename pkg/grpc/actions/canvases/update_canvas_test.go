package canvases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test__UpdateCanvas(t *testing.T) {
	r := support.Setup(t)

	t.Run("invalid canvas id -> error", func(t *testing.T) {
		name := "name"
		description := "description"
		_, err := UpdateCanvas(context.Background(), r.AuthService, r.Organization.ID.String(), "invalid-id", &name, &description, nil)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("canvas does not exist -> error", func(t *testing.T) {
		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			uuid.New().String(),
			stringPointer("updated-name"),
			stringPointer("updated-description"),
			nil,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, s.Code())
	})

	t.Run("empty name -> error", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			stringPointer("   "),
			stringPointer("description"),
			nil,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("updates canvas metadata", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		newName := support.RandomName("updated-canvas")
		newDescription := "Canvas description updated"

		response, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			&newName,
			&newDescription,
			nil,
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Metadata)
		assert.Equal(t, canvas.ID.String(), response.Canvas.Metadata.Id)
		assert.Equal(t, newName, response.Canvas.Metadata.Name)
		assert.Equal(t, newDescription, response.Canvas.Metadata.Description)

		updatedCanvas, findErr := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, findErr)
		assert.Equal(t, newName, updatedCanvas.Name)
		assert.Equal(t, newDescription, updatedCanvas.Description)
	})

	t.Run("duplicate name -> error", func(t *testing.T) {
		existingCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		targetCanvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			targetCanvas.ID.String(),
			&existingCanvas.Name,
			&targetCanvas.Description,
			nil,
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.AlreadyExists, s.Code())
	})

	t.Run("updates canvas change management setting", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("change_management_enabled", true).Error,
		)

		response, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{Enabled: true},
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Spec)
		require.NotNil(t, response.Canvas.Spec.ChangeManagement)
		assert.True(t, response.Canvas.Spec.ChangeManagement.Enabled)

		updatedCanvas, findErr := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, findErr)
		assert.True(t, updatedCanvas.ChangeManagementEnabled)
	})

	t.Run("organization change management enabled keeps effective canvas change management enabled", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("change_management_enabled", true).Error,
		)

		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{Enabled: true},
		)
		require.NoError(t, err)

		response, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{Enabled: false},
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Spec)
		require.NotNil(t, response.Canvas.Spec.ChangeManagement)
		assert.True(t, response.Canvas.Spec.ChangeManagement.Enabled)

		updatedCanvas, findErr := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, findErr)
		assert.False(t, updatedCanvas.ChangeManagementEnabled)
	})

	t.Run("organization change management disabled allows effective canvas change management to be enabled", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		require.NoError(
			t,
			database.Conn().Model(&models.Organization{}).Where("id = ?", r.Organization.ID).Update("change_management_enabled", false).Error,
		)

		response, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{Enabled: true},
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Spec)
		require.NotNil(t, response.Canvas.Spec.ChangeManagement)
		assert.True(t, response.Canvas.Spec.ChangeManagement.Enabled)

		updatedCanvas, findErr := models.FindCanvas(r.Organization.ID, canvas.ID)
		require.NoError(t, findErr)
		assert.True(t, updatedCanvas.ChangeManagementEnabled)
	})

	t.Run("updates change request approval config", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})
		user := support.CreateUser(t, r, r.Organization.ID)

		response, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{
				Approvals: []*pb.Canvas_ChangeManagement_Approver{
					{
						Type:   pb.Canvas_ChangeManagement_Approver_TYPE_USER,
						UserId: user.ID.String(),
					},
				},
			},
		)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Canvas)
		require.NotNil(t, response.Canvas.Spec)
		require.NotNil(t, response.Canvas.Spec.ChangeManagement)
		require.Len(t, response.Canvas.Spec.ChangeManagement.Approvals, 1)
		assert.Equal(t, pb.Canvas_ChangeManagement_Approver_TYPE_USER, response.Canvas.Spec.ChangeManagement.Approvals[0].Type)
		assert.Equal(t, user.ID.String(), response.Canvas.Spec.ChangeManagement.Approvals[0].UserId)
	})

	t.Run("invalid change request approval config user returns invalid argument", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{
				Approvals: []*pb.Canvas_ChangeManagement_Approver{
					{
						Type:   pb.Canvas_ChangeManagement_Approver_TYPE_USER,
						UserId: uuid.New().String(),
					},
				},
			},
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
	})

	t.Run("duplicate any-user approver returns invalid argument", func(t *testing.T) {
		canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{}, []models.Edge{})

		_, err := UpdateCanvas(
			context.Background(),
			r.AuthService,
			r.Organization.ID.String(),
			canvas.ID.String(),
			nil,
			nil,
			&pb.Canvas_ChangeManagement{
				Approvals: []*pb.Canvas_ChangeManagement_Approver{
					{Type: pb.Canvas_ChangeManagement_Approver_TYPE_ANYONE},
					{Type: pb.Canvas_ChangeManagement_Approver_TYPE_ANYONE},
				},
			},
		)
		s, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, s.Code())
		assert.Contains(t, s.Message(), "duplicate any-user approver")
	})
}

func stringPointer(value string) *string {
	return &value
}
