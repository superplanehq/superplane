package factories

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/models/factory"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
)

func Test__UpdateWorkOrderAssignees(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	userA := support.CreateUser(t, r, r.Organization.ID)
	userB := support.CreateUser(t, r, r.Organization.ID)

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "")
	require.NoError(t, err)

	t.Run("unassigning everyone clears all assignees", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		resp, err := UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{userA.ID.String(), userB.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.Order.Assignees, 2)

		resp, err = UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{},
		})
		require.NoError(t, err)
		assert.Empty(t, resp.Order.Assignees)

		refreshed, err := factoryModel.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Empty(t, refreshed.Assignees)
	})

	t.Run("partial removal keeps the correct assignee and records the diff", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{userA.ID.String(), userB.ID.String()},
		})
		require.NoError(t, err)

		resp, err := UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{userA.ID.String()},
		})
		require.NoError(t, err)
		require.Len(t, resp.Order.Assignees, 1)
		assert.Equal(t, userA.ID.String(), resp.Order.Assignees[0].Id)

		refreshed, err := factoryModel.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		events, err := refreshed.ListEvents(database.DB(t.Context()), 0, nil)
		require.NoError(t, err)

		var diff *factory.WorkOrderAssigneesUpdated
		for _, event := range events {
			if event.Type != factory.EventTypeOrderAssigneesUpdated {
				continue
			}

			var payload factory.WorkOrderAssigneesUpdated
			require.NoError(t, json.Unmarshal(event.Data, &payload))
			if len(payload.Unassigned) > 0 {
				diff = &payload
				break
			}
		}

		require.NotNil(t, diff, "expected an assignees-updated event recording the unassignment")
		require.Len(t, diff.Unassigned, 1)
		assert.Equal(t, userB.ID, diff.Unassigned[0].ID)
		assert.Empty(t, diff.Assigned)
	})

	t.Run("rejects unknown assignee id", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = UpdateWorkOrderAssignees(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{"00000000-0000-0000-0000-000000000099"},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("unauthenticated", func(t *testing.T) {
		order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Ship it", "", &r.User, nil, nil)
		require.NoError(t, err)

		_, err = UpdateWorkOrderAssignees(context.Background(), r.Organization.ID.String(), &pb.UpdateWorkOrderAssigneesRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			AssigneeIds: []string{},
		})
		code, _, ok := grpcerrors.HandlerStatus(err)
		require.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, code)
	})
}
