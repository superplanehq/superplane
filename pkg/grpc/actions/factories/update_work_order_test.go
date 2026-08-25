package factories

import (
	"context"
	"strings"
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

func Test__UpdateWorkOrder(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())

	factoryModel, err := models.CreateFactory(database.DB(t.Context()), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(database.DB(t.Context()), "Old title", "Old body", &r.User, nil, nil)
	require.NoError(t, err)

	t.Run("updates title and description", func(t *testing.T) {
		title := "New title"
		description := "New body"
		resp, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			Title:       &title,
			Description: &description,
		})
		require.NoError(t, err)
		assert.Equal(t, "New title", resp.Order.Title)
		assert.Equal(t, "New body", resp.Order.Description)

		refreshed, err := factoryModel.FindWorkOrder(database.DB(t.Context()), order.ID)
		require.NoError(t, err)
		assert.Equal(t, "New title", refreshed.Title)
		assert.Equal(t, "New body", refreshed.Description)
	})

	t.Run("updates description only", func(t *testing.T) {
		description := "Body only"
		resp, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId:   factoryModel.ID.String(),
			OrderId:     order.ID.String(),
			Description: &description,
		})
		require.NoError(t, err)
		assert.Equal(t, "New title", resp.Order.Title)
		assert.Equal(t, "Body only", resp.Order.Description)
	})

	t.Run("rejects a blank title", func(t *testing.T) {
		title := "   "
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Title:     &title,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a title that is too long", func(t *testing.T) {
		title := strings.Repeat("a", 257)
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
			Title:     &title,
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("rejects a request with no fields", func(t *testing.T) {
		_, err := UpdateWorkOrder(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   order.ID.String(),
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})
}
