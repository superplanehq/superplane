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

	t.Run("missing row returns all workspaces", func(t *testing.T) {
		resp, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		require.NotNil(t, resp.Settings)
		require.NotNil(t, resp.Settings.Workspaces)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Workspaces.Scope)
		assert.Empty(t, resp.Settings.Workspaces.Filters)
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

	t.Run("persists all workspaces", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Workspaces.Scope)
		assert.Empty(t, resp.Settings.Workspaces.Filters)

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, resp.Settings.Workspaces.Scope, described.Settings.Workspaces.Scope)
	})

	t.Run("filtered scope requires a workspace", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
				},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("none scope ignores filters", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_NONE,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceId: factoryModel.ID.String(),
						EventTypes:  []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED},
					}},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_NONE, resp.Settings.Workspaces.Scope)
		assert.Empty(t, resp.Settings.Workspaces.Filters)
	})

	t.Run("filtered scope stores workspace id and event types", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceId: factoryModel.ID.String(),
						EventTypes: []pb.NotificationSettings_Type{
							pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED,
							pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED,
						},
					}},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Settings.Workspaces.Filters, 1)
		assert.Equal(t, factoryModel.ID.String(), resp.Settings.Workspaces.Filters[0].WorkspaceId)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_ASSIGNED,
			pb.NotificationSettings_TYPE_WORK_ORDER_COMMENT_OWNED,
		}, resp.Settings.Workspaces.Filters[0].EventTypes)
	})

	t.Run("filtered scope accepts a workspace key", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceKey: factoryModel.Key,
						EventTypes:   []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED},
					}},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Settings.Workspaces.Filters, 1)
		assert.Equal(t, factoryModel.ID.String(), resp.Settings.Workspaces.Filters[0].WorkspaceId)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
		}, resp.Settings.Workspaces.Filters[0].EventTypes)
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
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceId: factoryModel.ID.String(),
						EventTypes:  []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_UNSPECIFIED},
					}},
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
