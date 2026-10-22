package factories

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func TestWorkOrderTaskCapacityAcrossWorkspaces(t *testing.T) {
	r := support.Setup(t)
	t.Setenv(models.BillingEnabledEnv, "yes")
	db := database.DB(t.Context())
	limit := 1
	plan := &models.BillingPlan{
		Key: "capacity_" + uuid.NewString()[:8], Name: "Task capacity", Category: "paid", Visibility: "private",
		ParallelTaskLimit: &limit, CreditFrequency: "none", CreditExpiration: "never",
	}
	require.NoError(t, models.SaveBillingPlan(db, plan))
	_, err := models.SetAdminOrganizationPlan(db, r.Organization.ID, plan.Key, nil)
	require.NoError(t, err)
	first, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("first"), "", "")
	require.NoError(t, err)
	second, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("second"), "", "")
	require.NoError(t, err)
	running, err := first.CreateWorkOrder(db, "Running", "", &r.User, nil, nil)
	require.NoError(t, err)
	draft, err := second.CreateWorkOrder(db, "Waiting to start", "", &r.User, nil, nil)
	require.NoError(t, err)
	app, entrypoint := support.CreateFactoryAppWithOnRunTrigger(t, r, first.ID, "implement", "start")
	line, err := first.CreateLine(db, "ship", []models.FactoryLineStep{
		{Type: models.FactoryLineStepTypeRunApp, AppID: app.ID, Entrypoint: entrypoint},
	})
	require.NoError(t, err)

	describe := func(factory *models.Factory, order *models.FactoryWorkOrder) *pb.WorkOrder {
		t.Helper()
		response, err := DescribeWorkOrder(t.Context(), r.Organization.ID.String(), &pb.DescribeWorkOrderRequest{
			FactoryId: factory.ID.String(), OrderId: order.ID.String(),
		})
		require.NoError(t, err)
		return response.Order
	}
	require.False(t, describe(second, draft).TaskLimitReached)
	_, err = DispatchWorkOrder(t.Context(), r.Organization.ID.String(), &pb.DispatchWorkOrderRequest{
		FactoryId: first.ID.String(), OrderId: running.ID.String(), LineName: line.Name,
	})
	require.NoError(t, err)
	require.True(t, describe(second, draft).TaskLimitReached)
	require.False(t, describe(first, running).TaskLimitReached, "work on an active task shares its existing slot")
	summaries, err := loadAndSerializeWorkOrders(t.Context(), second, []models.FactoryWorkOrder{*draft})
	require.NoError(t, err)
	require.True(t, summaries[0].TaskLimitReached)
	plan.ParallelTaskLimit = nil
	require.NoError(t, models.SaveBillingPlan(db, plan))
	require.False(t, describe(second, draft).TaskLimitReached)
}
