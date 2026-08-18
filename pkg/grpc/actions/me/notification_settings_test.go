package me

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/me"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func Test__DescribeNotificationSettings(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	t.Run("missing row returns defaults with notifications on", func(t *testing.T) {
		resp, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp.Settings)
		assert.True(t, resp.Settings.Enabled)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.WorkspaceScope)
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_CREATED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_ARTIFACT_OWNED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_MENTIONED))
	})

	t.Run("unauthenticated", func(t *testing.T) {
		_, err := DescribeNotificationSettings(context.Background())
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}

func Test__UpdateNotificationSettings(t *testing.T) {
	r := support.Setup(t)
	ctx := notificationSettingsContext(r.User.String(), r.Organization.ID.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("persists enabled settings for all workspaces", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:        true,
				WorkspaceScope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				Types: []*pb.NotificationSettings_TypeToggle{
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED, true),
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED, false),
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_CREATED, true),
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED, true),
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_ARTIFACT_OWNED, false),
					notificationTypeToggle(pb.NotificationSettings_TYPE_WORK_ORDER_MENTIONED, true),
				},
			},
		})
		require.NoError(t, err)
		assert.True(t, resp.Settings.Enabled)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.WorkspaceScope)
		assert.False(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED))
		assert.False(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_ARTIFACT_OWNED))
		assert.True(t, notificationTypeToggleEnabled(resp.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_MENTIONED))

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, resp.Settings.Enabled, described.Settings.Enabled)
		assert.False(t, notificationTypeToggleEnabled(described.Settings, pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED))
	})

	t.Run("selected scope requires a workspace when enabled", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
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
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
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
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
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
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unspecified notification type is rejected", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Enabled:        true,
				WorkspaceScope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				Types: []*pb.NotificationSettings_TypeToggle{
					notificationTypeToggle(pb.NotificationSettings_TYPE_UNSPECIFIED, true),
				},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}

func notificationSettingsContext(userID, organizationID string) context.Context {
	return metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-user-id", userID,
			"x-organization-id", organizationID,
		),
	)
}

func notificationTypeToggle(notificationType pb.NotificationSettings_Type, enabled bool) *pb.NotificationSettings_TypeToggle {
	return &pb.NotificationSettings_TypeToggle{Type: notificationType, Enabled: enabled}
}

func notificationTypeToggleEnabled(settings *pb.NotificationSettings, notificationType pb.NotificationSettings_Type) bool {
	for _, toggle := range settings.GetTypes() {
		if toggle.GetType() == notificationType {
			return toggle.GetEnabled()
		}
	}
	return true
}
