package factories

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/test/support"
)

func Test__ListWorkOrders_PagesByUpdatedAt(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Paged Orders", "", "PG")
	require.NoError(t, err)

	first, err := factoryModel.CreateWorkOrder(db, "Oldest", "", &r.User, nil, nil)
	require.NoError(t, err)
	second, err := factoryModel.CreateWorkOrder(db, "Middle", "", &r.User, nil, nil)
	require.NoError(t, err)
	third, err := factoryModel.CreateWorkOrder(db, "Newest", "", &r.User, nil, nil)
	require.NoError(t, err)

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("id = ?", first.ID).UpdateColumn("updated_at", base).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("id = ?", second.ID).UpdateColumn("updated_at", base.Add(time.Hour)).Error)
	require.NoError(t, db.Model(&models.FactoryWorkOrder{}).Where("id = ?", third.ID).UpdateColumn("updated_at", base.Add(2*time.Hour)).Error)

	page, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
		Limit:     2,
	})
	require.NoError(t, err)
	require.Len(t, page.Orders, 2)
	assert.True(t, page.HasNextPage)
	assert.Equal(t, third.ID.String(), page.Orders[0].GetId())
	assert.Equal(t, second.ID.String(), page.Orders[1].GetId())

	next, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
		Limit:     2,
		BeforeId:  second.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, next.Orders, 1)
	assert.False(t, next.HasNextPage)
	assert.Equal(t, first.ID.String(), next.Orders[0].GetId())
}

func Test__ListWorkOrders_KeepsStatesWhenUserIDIsSet(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "User State Filter", "", "US")
	require.NoError(t, err)

	draft, err := factoryModel.CreateWorkOrder(db, "Draft for me", "", &r.User, nil, nil)
	require.NoError(t, err)
	open, err := factoryModel.CreateWorkOrder(db, "Open for me", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = open.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	userID := r.User.String()
	resp, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
		UserId:    &userID,
		States:    []pb.WorkOrder_State{pb.WorkOrder_STATE_DRAFT},
	})
	require.NoError(t, err)
	require.Len(t, resp.Orders, 1)
	assert.Equal(t, draft.ID.String(), resp.Orders[0].GetId())
}

func Test__ListWorkOrders_KeepsStatesWhenUserIDAndUnassignedAreSet(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "User Unassigned State Filter", "", "UU")
	require.NoError(t, err)

	otherUser := support.CreateUser(t, r, r.Organization.ID)

	draftMine, err := factoryModel.CreateWorkOrder(db, "Draft for me", "", &r.User, nil, nil)
	require.NoError(t, err)
	draftUnassigned, err := factoryModel.CreateWorkOrder(db, "Draft unassigned", "", &otherUser.ID, nil, nil)
	require.NoError(t, err)

	openMine, err := factoryModel.CreateWorkOrder(db, "Open for me", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = openMine.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	openUnassigned, err := factoryModel.CreateWorkOrder(db, "Open unassigned", "", &otherUser.ID, nil, nil)
	require.NoError(t, err)
	_, err = openUnassigned.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)

	userID := r.User.String()
	unassigned := true
	resp, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId:  factoryModel.ID.String(),
		UserId:     &userID,
		Unassigned: &unassigned,
		States:     []pb.WorkOrder_State{pb.WorkOrder_STATE_DRAFT},
	})
	require.NoError(t, err)
	require.Len(t, resp.Orders, 2)
	assert.ElementsMatch(t, []string{draftMine.ID.String(), draftUnassigned.ID.String()}, []string{
		resp.Orders[0].GetId(),
		resp.Orders[1].GetId(),
	})
}

func Test__ListWorkOrders_UsesDefaultLimit(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Default Limit", "", "DL")
	require.NoError(t, err)
	_, err = factoryModel.CreateWorkOrder(db, "One", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = factoryModel.CreateWorkOrder(db, "Two", "", &r.User, nil, nil)
	require.NoError(t, err)

	resp, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
	})
	require.NoError(t, err)
	assert.Len(t, resp.Orders, 2)
	assert.False(t, resp.HasNextPage)
}

func Test__ListWorkOrders_FiltersByLine(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Line Filter", "", "LF")
	require.NoError(t, err)
	lineA, err := factoryModel.CreateLine(db, "alpha", nil)
	require.NoError(t, err)
	lineB, err := factoryModel.CreateLine(db, "beta", nil)
	require.NoError(t, err)

	onA, err := factoryModel.CreateWorkOrder(db, "On alpha", "", &r.User, nil, nil)
	require.NoError(t, err)
	onB, err := factoryModel.CreateWorkOrder(db, "On beta", "", &r.User, nil, nil)
	require.NoError(t, err)
	unassigned, err := factoryModel.CreateWorkOrder(db, "No line", "", &r.User, nil, nil)
	require.NoError(t, err)

	support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, onA.ID, lineA.ID, lineA.Name, nil)
	support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, onB.ID, lineB.ID, lineB.Name, nil)

	resp, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
		LineId:    lineA.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Orders, 2)
	ids := []string{resp.Orders[0].GetId(), resp.Orders[1].GetId()}
	assert.ElementsMatch(t, []string{onA.ID.String(), unassigned.ID.String()}, ids)
}

func Test__ListWorkOrders_LineFilterKeepsFactoryAndState(t *testing.T) {
	r := support.Setup(t)
	ctx := t.Context()
	db := database.DB(ctx)

	factoryModel, err := models.CreateFactory(db, r.Organization.ID, "Line Scope", "", "LS")
	require.NoError(t, err)
	line, err := factoryModel.CreateLine(db, "alpha", nil)
	require.NoError(t, err)

	closedOnLine, err := factoryModel.CreateWorkOrder(db, "Closed on line", "", &r.User, nil, nil)
	require.NoError(t, err)
	_, err = closedOnLine.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{ToState: models.FactoryWorkOrderStateOpen})
	require.NoError(t, err)
	_, err = closedOnLine.UpdateStatus(db, models.FactoryWorkOrderStatusUpdate{
		ToState: models.FactoryWorkOrderStateClosed,
		Result:  models.FactoryWorkOrderResultFailed,
	})
	require.NoError(t, err)
	support.CreateFactoryLineDispatch(t, r.Organization.ID, factoryModel.ID, closedOnLine.ID, line.ID, line.Name, nil)

	draftHere, err := factoryModel.CreateWorkOrder(db, "Draft here", "", &r.User, nil, nil)
	require.NoError(t, err)

	otherFactory, err := models.CreateFactory(db, r.Organization.ID, "Other Line Scope", "", "OL")
	require.NoError(t, err)
	otherDraft, err := otherFactory.CreateWorkOrder(db, "Other factory draft", "", &r.User, nil, nil)
	require.NoError(t, err)

	resp, err := ListWorkOrders(ctx, r.Organization.ID.String(), &pb.ListWorkOrdersRequest{
		FactoryId: factoryModel.ID.String(),
		LineId:    line.ID.String(),
		States:    []pb.WorkOrder_State{pb.WorkOrder_STATE_CLOSED},
	})
	require.NoError(t, err)
	require.Len(t, resp.Orders, 1)
	assert.Equal(t, closedOnLine.ID.String(), resp.Orders[0].GetId())
	assert.NotEqual(t, draftHere.ID.String(), resp.Orders[0].GetId())
	assert.NotEqual(t, otherDraft.ID.String(), resp.Orders[0].GetId())
}
