package factories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__UpdateWorkOrderStatus__UnusedDraftRejectDeletesAndReturnsClosedSnapshot(t *testing.T) {
	r := support.Setup(t)
	ctx := authentication.SetUserIdInMetadata(context.Background(), r.User.String())
	db := database.DB(t.Context())

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	order, err := factoryModel.CreateWorkOrder(db, "Unused draft", "", &r.User, nil, nil)
	require.NoError(t, err)

	resp, err := UpdateWorkOrderStatus(ctx, r.Organization.ID.String(), &pb.UpdateWorkOrderStatusRequest{
		FactoryId: factoryModel.ID.String(),
		OrderId:   order.ID.String(),
		State:     pb.WorkOrder_STATE_CLOSED,
		Result:    pb.WorkOrder_RESULT_REJECTED,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Order)
	assert.Equal(t, pb.WorkOrder_STATE_CLOSED, resp.Order.State)
	assert.Equal(t, pb.WorkOrder_RESULT_REJECTED, resp.Order.Result)

	_, err = factoryModel.FindWorkOrder(db, order.ID)
	require.ErrorIs(t, err, models.ErrFactoryWorkOrderNotFound)
}
