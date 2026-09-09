package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
)

func TestNextFactoryWorkOrderPosition(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.DB(t.Context())

	_, userID, factoryModel := setupFactoryWithUser(t, "next-position-empty")

	position, err := NextFactoryWorkOrderPosition(tx, factoryModel.ID)
	require.NoError(t, err)
	assert.Zero(t, position, "an empty factory starts its first order at position 0")

	first, err := factoryModel.CreateWorkOrder(tx, "First", "", &userID, nil, nil)
	require.NoError(t, err)

	second, err := factoryModel.CreateWorkOrder(tx, "Second", "", &userID, nil, nil)
	require.NoError(t, err)

	assert.Less(t, second.Position, first.Position, "a newly created order starts before every existing one")
}

func TestFactoryWorkOrder_Reorder(t *testing.T) {
	require.NoError(t, database.TruncateTables())
	tx := database.DB(t.Context())

	// Every subtest gets its own three orders so reorders in one subtest
	// cannot change the fixture another subtest relies on.
	newTrio := func(t *testing.T, prefix string) (a, b, c *FactoryWorkOrder) {
		t.Helper()
		_, userID, factoryModel := setupFactoryWithUser(t, prefix)

		// Ascending position order after creation is c, b, a (newest first).
		a, err := factoryModel.CreateWorkOrder(tx, "A", "", &userID, nil, nil)
		require.NoError(t, err)
		b, err = factoryModel.CreateWorkOrder(tx, "B", "", &userID, nil, nil)
		require.NoError(t, err)
		c, err = factoryModel.CreateWorkOrder(tx, "C", "", &userID, nil, nil)
		require.NoError(t, err)
		require.Less(t, c.Position, b.Position)
		require.Less(t, b.Position, a.Position)
		return a, b, c
	}

	t.Run("moves to the top when there is no previous neighbor", func(t *testing.T) {
		a, _, c := newTrio(t, "reorder-top")
		require.NoError(t, a.Reorder(tx, nil, c))
		assert.Less(t, a.Position, c.Position)
	})

	t.Run("moves to the bottom when there is no next neighbor", func(t *testing.T) {
		a, b, _ := newTrio(t, "reorder-bottom")
		require.NoError(t, b.Reorder(tx, a, nil))
		assert.Greater(t, b.Position, a.Position)
	})

	t.Run("moves between two neighbors", func(t *testing.T) {
		a, b, c := newTrio(t, "reorder-middle")
		// Current ascending order is c, b, a. Move c strictly between b and a.
		require.NoError(t, c.Reorder(tx, b, a))
		assert.Greater(t, c.Position, b.Position)
		assert.Less(t, c.Position, a.Position)
	})

	t.Run("is a no-op when the position does not change", func(t *testing.T) {
		a, _, c := newTrio(t, "reorder-noop")
		require.NoError(t, a.Reorder(tx, nil, c))
		before := a.Position

		require.NoError(t, a.Reorder(tx, nil, c))
		assert.Equal(t, before, a.Position)
	})

	t.Run("persists across a reload", func(t *testing.T) {
		a, b, c := newTrio(t, "reorder-reload")
		require.NoError(t, b.Reorder(tx, c, a))

		reloaded, err := FindUnscopedWorkOrder(tx, b.ID)
		require.NoError(t, err)
		assert.Equal(t, b.Position, reloaded.Position)
	})
}

func TestFactoryWorkOrder_Lane(t *testing.T) {
	draft := &FactoryWorkOrder{State: FactoryWorkOrderStateDraft}
	assert.Equal(t, FactoryWorkOrderLaneBacklog, draft.Lane(false))

	closed := &FactoryWorkOrder{State: FactoryWorkOrderStateClosed}
	assert.Equal(t, FactoryWorkOrderLaneDone, closed.Lane(false))
	// A dispatch outliving the order's closing state (shouldn't happen, but
	// closed always wins so the board never shows a done card as running).
	assert.Equal(t, FactoryWorkOrderLaneDone, closed.Lane(true))

	openRunning := &FactoryWorkOrder{State: FactoryWorkOrderStateOpen}
	assert.Equal(t, FactoryWorkOrderLaneRunning, openRunning.Lane(true))

	openWaiting := &FactoryWorkOrder{State: FactoryWorkOrderStateOpen}
	assert.Equal(t, FactoryWorkOrderLaneReview, openWaiting.Lane(false))
}
