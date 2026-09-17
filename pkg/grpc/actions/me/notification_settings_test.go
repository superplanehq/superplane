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
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_NOTE_OWNED,
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
			pb.NotificationSettings_TYPE_WORK_ORDER_PLAN_READY,
		}, resp.Settings.Workspaces.EventTypes)
		require.NotNil(t, resp.Settings.Browser)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_NONE, resp.Settings.Browser.Scope)
		assert.True(t, resp.Settings.Browser.ShowWhileViewing)
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
					EventTypes: []pb.NotificationSettings_Type{
						pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
						pb.NotificationSettings_TYPE_WORK_ORDER_PLAN_READY,
					},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Workspaces.Scope)
		assert.Empty(t, resp.Settings.Workspaces.Filters)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
			pb.NotificationSettings_TYPE_WORK_ORDER_PLAN_READY,
		}, resp.Settings.Workspaces.EventTypes)
		require.NotNil(t, resp.Settings.Browser)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_NONE, resp.Settings.Browser.Scope)
		assert.True(t, resp.Settings.Browser.ShowWhileViewing)

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, resp.Settings.Workspaces.Scope, described.Settings.Workspaces.Scope)
		assert.Equal(t, resp.Settings.Workspaces.EventTypes, described.Settings.Workspaces.EventTypes)
	})

	t.Run("persists the status note type", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
					EventTypes: []pb.NotificationSettings_Type{
						pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_NOTE_OWNED,
					},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_NOTE_OWNED,
		}, resp.Settings.Workspaces.EventTypes)
	})

	t.Run("persists the agent question and plan ready types", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
					EventTypes: []pb.NotificationSettings_Type{
						pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
						pb.NotificationSettings_TYPE_WORK_ORDER_PLAN_READY,
					},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
			pb.NotificationSettings_TYPE_WORK_ORDER_PLAN_READY,
		}, resp.Settings.Workspaces.EventTypes)
	})

	t.Run("legacy removed type strings load and save without error", func(t *testing.T) {
		_, err := models.UpsertUserNotificationSettings(database.DB(t.Context()), r.Organization.ID, r.User, models.UserNotificationSettingsParams{
			WorkspaceScope: models.NotificationWorkspaceScopeAll,
			EventTypes: []string{
				"work_order_assigned",
				"work_order_comment_owned",
				"work_order_comment_created",
				"work_order_artifact_owned",
				"work_order_mention",
				models.NotificationTypeWorkOrderStatusOwned,
			},
		})
		require.NoError(t, err)

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
		}, described.Settings.Workspaces.EventTypes)

		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: described.Settings,
		})
		require.NoError(t, err)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
		}, resp.Settings.Workspaces.EventTypes)
	})

	t.Run("legacy only removed types round-trip without enabling current types", func(t *testing.T) {
		_, err := models.UpsertUserNotificationSettings(database.DB(t.Context()), r.Organization.ID, r.User, models.UserNotificationSettingsParams{
			WorkspaceScope:        models.NotificationWorkspaceScopeAll,
			EventTypes:            []string{"work_order_mention"},
			BrowserWorkspaceScope: models.NotificationWorkspaceScopeAll,
			BrowserEventTypes:     []string{"work_order_mention"},
		})
		require.NoError(t, err)

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, described.Settings.Workspaces.Scope)
		assert.Empty(t, described.Settings.Workspaces.EventTypes)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, described.Settings.Browser.Scope)
		assert.Empty(t, described.Settings.Browser.EventTypes)

		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: described.Settings,
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Settings.Workspaces.EventTypes)
		assert.Empty(t, resp.Settings.Browser.EventTypes)

		settings, err := models.FindUserNotificationSettings(database.DB(t.Context()), r.Organization.ID, r.User)
		require.NoError(t, err)
		assert.False(t, settings.Notifies(factoryModel.ID, models.NotificationTypeWorkOrderStatusOwned))
		assert.False(t, settings.Notifies(factoryModel.ID, models.NotificationTypeWorkOrderPlanReady))
		assert.False(t, settings.NotifiesChannel(
			models.NotificationChannelBrowser,
			factoryModel.ID,
			models.NotificationTypeWorkOrderAgentQuestion,
		))
	})

	t.Run("empty all-scope types stay off", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				},
				Browser: &pb.NotificationSettings_Browser{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				},
			},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Settings.Workspaces.EventTypes)
		assert.Empty(t, resp.Settings.Browser.EventTypes)

		settings, err := models.FindUserNotificationSettings(database.DB(t.Context()), r.Organization.ID, r.User)
		require.NoError(t, err)
		assert.False(t, settings.Notifies(factoryModel.ID, models.NotificationTypeWorkOrderStatusOwned))
		assert.False(t, settings.NotifiesChannel(
			models.NotificationChannelBrowser,
			factoryModel.ID,
			models.NotificationTypeWorkOrderPlanReady,
		))
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
						EventTypes:  []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED},
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
							pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
							pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
						},
					}},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Settings.Workspaces.Filters, 1)
		assert.Equal(t, factoryModel.ID.String(), resp.Settings.Workspaces.Filters[0].WorkspaceId)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED,
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
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

	t.Run("invalid workspace key is rejected", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceKey: "TOOLONG",
						EventTypes:   []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED},
					}},
				},
			},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("blank workspace key is rejected", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceKey: "   ",
						EventTypes:   []pb.NotificationSettings_Type{pb.NotificationSettings_TYPE_WORK_ORDER_STATUS_OWNED},
					}},
				},
			},
		})
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

	t.Run("persists browser channel settings", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_NONE,
				},
				Browser: &pb.NotificationSettings_Browser{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
					EventTypes: []pb.NotificationSettings_Type{
						pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
					},
					ShowWhileViewing: false,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Settings.Browser)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Browser.Scope)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
		}, resp.Settings.Browser.EventTypes)
		assert.False(t, resp.Settings.Browser.ShowWhileViewing)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_NONE, resp.Settings.Workspaces.Scope)

		described, err := DescribeNotificationSettings(ctx)
		require.NoError(t, err)
		assert.Equal(t, resp.Settings.Browser.Scope, described.Settings.Browser.Scope)
		assert.Equal(t, resp.Settings.Browser.EventTypes, described.Settings.Browser.EventTypes)
		assert.Equal(t, resp.Settings.Browser.ShowWhileViewing, described.Settings.Browser.ShowWhileViewing)
	})

	t.Run("browser filtered scope stores workspace id and event types", func(t *testing.T) {
		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				},
				Browser: &pb.NotificationSettings_Browser{
					Scope:            pb.NotificationSettings_WORKSPACE_SCOPE_FILTERED,
					ShowWhileViewing: true,
					Filters: []*pb.NotificationSettings_WorkspaceFilter{{
						WorkspaceId: factoryModel.ID.String(),
						EventTypes: []pb.NotificationSettings_Type{
							pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
						},
					}},
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resp.Settings.Browser.Filters, 1)
		assert.Equal(t, factoryModel.ID.String(), resp.Settings.Browser.Filters[0].WorkspaceId)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
		}, resp.Settings.Browser.Filters[0].EventTypes)
		assert.True(t, resp.Settings.Browser.ShowWhileViewing)
	})

	t.Run("omitted browser channel keeps stored browser settings", func(t *testing.T) {
		_, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_NONE,
				},
				Browser: &pb.NotificationSettings_Browser{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
					EventTypes: []pb.NotificationSettings_Type{
						pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
					},
					ShowWhileViewing: false,
				},
			},
		})
		require.NoError(t, err)

		resp, err := UpdateNotificationSettings(ctx, &pb.UpdateNotificationSettingsRequest{
			Settings: &pb.NotificationSettings{
				Workspaces: &pb.NotificationSettings_Workspaces{
					Scope: pb.NotificationSettings_WORKSPACE_SCOPE_ALL,
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Settings.Browser)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Workspaces.Scope)
		assert.Equal(t, pb.NotificationSettings_WORKSPACE_SCOPE_ALL, resp.Settings.Browser.Scope)
		assert.Equal(t, []pb.NotificationSettings_Type{
			pb.NotificationSettings_TYPE_WORK_ORDER_AGENT_QUESTION,
		}, resp.Settings.Browser.EventTypes)
		assert.False(t, resp.Settings.Browser.ShowWhileViewing)
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
