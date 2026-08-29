package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestFactory_ListWorkOrders_Mine(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	org, callerID, factoryModel := setupFactoryWithUser(t, "mine-filter")
	otherUser := createOrgUser(t, org.ID, "mine-filter-other")

	assigneeOnly, err := factoryModel.CreateWorkOrder(database.Conn(), "Assignee only", "", &otherUser.ID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	creatorOnly, err := factoryModel.CreateWorkOrder(database.Conn(), "Creator only", "", &callerID, nil, nil)
	require.NoError(t, err)

	both, err := factoryModel.CreateWorkOrder(database.Conn(), "Creator and assignee", "", &callerID, []uuid.UUID{callerID}, nil)
	require.NoError(t, err)

	_, err = factoryModel.CreateWorkOrder(database.Conn(), "Other user only", "", &otherUser.ID, []uuid.UUID{otherUser.ID}, nil)
	require.NoError(t, err)

	orders, err := factoryModel.ListWorkOrders(database.Conn(), ListFactoryWorkOrdersFilters{Mine: &callerID})
	require.NoError(t, err)

	ids := make([]uuid.UUID, 0, len(orders))
	for _, order := range orders {
		ids = append(ids, order.ID)
	}

	assert.ElementsMatch(t, []uuid.UUID{assigneeOnly.ID, creatorOnly.ID, both.ID}, ids)
}
