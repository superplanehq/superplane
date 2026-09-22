package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactory_ListWorkOrders_UserID(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, callerID, factoryModel := setupFactoryWithUser(t, "user-filter")
	otherUser := createOrgUser(t, org.ID, "user-filter-other")
	unassignedTrue := true

	assigneeOnly, err := factoryModel.CreateWorkOrder(database.Conn(), "Assignee only", "", &otherUser.ID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	creatorOnly, err := factoryModel.CreateWorkOrder(database.Conn(), "Creator only", "", &callerID, nil, nil)
	require.NoError(t, err)

	both, err := factoryModel.CreateWorkOrder(database.Conn(), "Creator and assignee", "", &callerID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	_, err = factoryModel.CreateWorkOrder(database.Conn(), "Other user only", "", &otherUser.ID, []uuid.UUID{otherUser.ID}, nil)
	require.NoError(t, err)

	otherUnassigned, err := factoryModel.CreateWorkOrder(database.Conn(), "Other unassigned", "", &otherUser.ID, nil, nil)
	require.NoError(t, err)

	orders, err := factoryModel.ListWorkOrders(database.Conn(), ListFactoryWorkOrdersFilters{UserID: &callerID, Limit: 10})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{assigneeOnly.ID, creatorOnly.ID, both.ID}, workOrderIDs(orders))

	nobody, err := factoryModel.ListWorkOrders(database.Conn(), ListFactoryWorkOrdersFilters{
		Unassigned: &unassignedTrue,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{creatorOnly.ID, otherUnassigned.ID}, workOrderIDs(nobody))

	userOrNobody, err := factoryModel.ListWorkOrders(database.Conn(), ListFactoryWorkOrdersFilters{
		UserID:     &callerID,
		Unassigned: &unassignedTrue,
		Limit:      10,
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{assigneeOnly.ID, creatorOnly.ID, both.ID, otherUnassigned.ID}, workOrderIDs(userOrNobody))
}

func workOrderIDs(orders []FactoryWorkOrder) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}
	return ids
}

func TestFactory_ListWorkOrders_PagesByUpdatedAt(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "page-orders")
	db := database.Conn()

	first, err := factoryModel.CreateWorkOrder(db, "Oldest update", "", &callerID, nil, nil)
	require.NoError(t, err)
	second, err := factoryModel.CreateWorkOrder(db, "Middle update", "", &callerID, nil, nil)
	require.NoError(t, err)
	third, err := factoryModel.CreateWorkOrder(db, "Newest update", "", &callerID, nil, nil)
	require.NoError(t, err)

	base := time.Now().UTC().Add(-3 * time.Hour).Truncate(time.Millisecond)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", first.ID).UpdateColumn("updated_at", base).Error)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", second.ID).UpdateColumn("updated_at", base.Add(time.Hour)).Error)
	require.NoError(t, db.Model(&FactoryWorkOrder{}).Where("id = ?", third.ID).UpdateColumn("updated_at", base.Add(2*time.Hour)).Error)

	page, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{Limit: 2})
	require.NoError(t, err)
	require.Len(t, page, 2)
	assert.Equal(t, []uuid.UUID{third.ID, second.ID}, []uuid.UUID{page[0].ID, page[1].ID})

	next, err := factoryModel.ListWorkOrders(db, ListFactoryWorkOrdersFilters{
		Limit:    2,
		BeforeID: &second.ID,
	})
	require.NoError(t, err)
	require.Len(t, next, 1)
	assert.Equal(t, first.ID, next[0].ID)
}

func TestFactory_ListWorkOrders_UsesDefaultLimit(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	_, callerID, factoryModel := setupFactoryWithUser(t, "default-limit")
	created, err := factoryModel.CreateWorkOrder(database.Conn(), "Default page", "", &callerID, nil, nil)
	require.NoError(t, err)

	orders, err := factoryModel.ListWorkOrders(database.Conn(), ListFactoryWorkOrdersFilters{})
	require.NoError(t, err)
	require.Len(t, orders, 1)
	assert.Equal(t, created.ID, orders[0].ID)
}
