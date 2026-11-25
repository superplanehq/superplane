package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestFactoryContext_CreateWorkOrder_ConfiguredOriginURL(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	webhookIntake := func(t *testing.T) (*models.Factory, *models.Canvas, *models.CanvasNodeExecution) {
		t.Helper()
		factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		canvas, execution, _ := setupFactoryAppExecutionWithPayload(t, r, factory.ID, map[string]any{
			"type": "webhook",
			"data": map[string]any{
				"body":    map[string]any{"title": "Checkout fails", "docs": "https://docs.example.com/guide"},
				"headers": map[string]any{"Referer": []any{"https://sender.example.com/hooks"}},
			},
		})
		_, err = factory.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceWebhook)
		require.NoError(t, err)
		return factory, canvas, execution
	}

	t.Run("the configured link is the origin and a second request with it is skipped", func(t *testing.T) {
		factory, canvas, execution := webhookIntake(t)
		link := "https://tracker.example.com/items/42"

		created, inserted, err := NewFactoryContext(database.Conn(), canvas, execution).
			CreateWorkOrder(core.WorkOrderParams{Title: "Checkout fails", OriginURL: "  " + link + " "})
		require.NoError(t, err)
		require.True(t, inserted)
		persisted, err := factory.FindWorkOrder(database.Conn(), uuid.MustParse(created.ID))
		require.NoError(t, err)
		require.NotNil(t, persisted.Origin())
		assert.Equal(t, link, persisted.Origin().URL)

		retryCanvas, retryExecution, _ := setupFactoryAppExecutionWithPayload(t, r, factory.ID, map[string]any{"type": "webhook", "data": map[string]any{}})
		order, inserted, err := NewFactoryContext(database.Conn(), retryCanvas, retryExecution).
			CreateWorkOrder(core.WorkOrderParams{Title: "Checkout fails again", OriginURL: link})
		require.NoError(t, err)
		assert.False(t, inserted)
		assert.Nil(t, order)
		assert.Equal(t, created.ID, retryExecution.Metadata.Data()[models.CanvasNodeExecutionExistingTaskIDKey])

		orders, err := factory.ListWorkOrders(database.Conn(), models.ListFactoryWorkOrdersFilters{Limit: 10})
		require.NoError(t, err)
		assert.Len(t, orders, 1)
	})

	t.Run("a webhook intake without a link does not take one from the request", func(t *testing.T) {
		factory, canvas, execution := webhookIntake(t)

		created, inserted, err := NewFactoryContext(database.Conn(), canvas, execution).
			CreateWorkOrder(core.WorkOrderParams{Title: "Checkout fails", OriginURL: "not a link"})
		require.NoError(t, err)
		require.True(t, inserted)
		persisted, err := factory.FindWorkOrder(database.Conn(), uuid.MustParse(created.ID))
		require.NoError(t, err)
		assert.Nil(t, persisted.Origin())
	})
}
