package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ReorderWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	// Both start as drafts, so both are in the Backlog column. Ascending
	// position order after creation is b, a (newest first).
	a, err := factoryModel.CreateWorkOrder(db, "A", "", &r.User, nil, nil)
	require.NoError(t, err)
	b, err := factoryModel.CreateWorkOrder(db, "B", "", &r.User, nil, nil)
	require.NoError(t, err)
	require.Less(t, b.Position, a.Position)

	t.Run("moves an order to the top of its column", func(t *testing.T) {
		bID := b.ID.String()
		resp, err := ReorderWorkOrder(ctx, r.Organization.ID.String(), &pb.ReorderWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     a.ID.String(),
			NextOrderId: &bID,
		})
		require.NoError(t, err)
		assert.Less(t, resp.Order.Position, b.Position)

		reloaded, err := models.FindUnscopedWorkOrder(db, a.ID)
		require.NoError(t, err)
		assert.Equal(t, resp.Order.Position, reloaded.Position)
	})

	t.Run("moves an order to the bottom of its column", func(t *testing.T) {
		aID := a.ID.String()
		resp, err := ReorderWorkOrder(ctx, r.Organization.ID.String(), &pb.ReorderWorkOrderRequest{
			FactoryId:       factoryModel.ID.String(),
			OrderId:         b.ID.String(),
			PreviousOrderId: &aID,
		})
		require.NoError(t, err)

		reloaded, err := models.FindUnscopedWorkOrder(db, a.ID)
		require.NoError(t, err)
		assert.Greater(t, resp.Order.Position, reloaded.Position)
	})

	t.Run("rejects a reorder that crosses into another column", func(t *testing.T) {
		// Close b so it moves into the Done column, out of Backlog with a.
		closed, err := models.FindUnscopedWorkOrder(db, b.ID)
		require.NoError(t, err)
		_, err = closed.Close(db, models.FactoryWorkOrderResultRejected, &r.User)
		require.NoError(t, err)

		closedID := closed.ID.String()
		_, err = ReorderWorkOrder(ctx, r.Organization.ID.String(), &pb.ReorderWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     a.ID.String(),
			NextOrderId: &closedID,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.FailedPrecondition, code)
	})

	t.Run("rejects an unknown neighbor id", func(t *testing.T) {
		bogus := "not-a-uuid"
		_, err := ReorderWorkOrder(ctx, r.Organization.ID.String(), &pb.ReorderWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     a.ID.String(),
			NextOrderId: &bogus,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}
