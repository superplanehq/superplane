package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__DescribeNotificationSettings(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	t.Run("missing row returns defaults with notifications on", func(t *testing.T) {
		resp, err := DescribeNotificationSettings(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		require.NotNil(t, resp.Settings)
		assert.True(t, resp.Settings.Enabled)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.WorkspaceScope)
		assert.True(t, resp.Settings.WorkOrderAssigned)
		assert.True(t, resp.Settings.WorkOrderCommentOwned)
		assert.True(t, resp.Settings.WorkOrderCommentCreated)
		assert.True(t, resp.Settings.WorkOrderStatusOwned)
		assert.True(t, resp.Settings.WorkOrderArtifactOwned)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := DescribeNotificationSettings(context.Background(), r.Organization.ID.String())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}

func Test__UpdateNotificationSettings(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("persists enabled settings for all workspaces", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, r.Organization.ID.String(), &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:                 true,
				WorkspaceScope:          pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				WorkOrderAssigned:       true,
				WorkOrderCommentOwned:   false,
				WorkOrderCommentCreated: true,
				WorkOrderStatusOwned:    true,
				WorkOrderArtifactOwned:  false,
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.Settings.Enabled)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.WorkspaceScope)
		assert.False(t, resp.Settings.WorkOrderCommentOwned)
		assert.False(t, resp.Settings.WorkOrderArtifactOwned)

		described, err := DescribeNotificationSettings(ctx, r.Organization.ID.String())
		require.NoError(t, err)
		assert.Equal(t, resp.Settings.Enabled, described.Settings.Enabled)
		assert.False(t, described.Settings.WorkOrderCommentOwned)
	})

	t.Run("selected scope requires a workspace when enabled", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, r.Organization.ID.String(), &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:        true,
				WorkspaceScope: pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED,
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("disabled selected scope allows an empty workspace list", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, r.Organization.ID.String(), &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:        false,
				WorkspaceScope: pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED,
			},
		})
		require.NoError(t, err)
		assert.False(t, resp.Settings.Enabled)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED, resp.Settings.WorkspaceScope)
		assert.Empty(t, resp.Settings.FactoryIds)
	})

	t.Run("selected scope stores factory ids", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, r.Organization.ID.String(), &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:        true,
				WorkspaceScope: pb.NotificationSettings_WORKSPACE_SCOPE_SELECTED,
				FactoryIds:     []string{factoryModel.ID.String()},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []string{factoryModel.ID.String()}, resp.Settings.FactoryIds)
	})

	t.Run("settings is required", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, r.Organization.ID.String(), &pb.UpdateNotificationSettingsRequest{})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}
