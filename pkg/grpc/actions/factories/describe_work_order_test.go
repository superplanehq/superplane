package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__DescribeWorkOrder_AcceptsNumberAndKey(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Numbers", "", "NUM")
	require.NoError(t, err)
	order, err := factoryModel.CreateWorkOrder(db, "one", "", &r.User, nil, nil)
	require.NoError(t, err)

	t.Run("describes by task number", func(t *testing.T) {
		resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: "num",
			OrderId:   "1",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), resp.Order.GetId())
		assert.Equal(t, int64(1), resp.Order.GetNumber())
	})

	t.Run("describes by task key", func(t *testing.T) {
		resp, err := DescribeWorkOrder(ctx, r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: factoryModel.ID.String(),
			OrderId:   "num-1",
		})
		require.NoError(t, err)
		assert.Equal(t, order.ID.String(), resp.Order.GetId())
	})
}
