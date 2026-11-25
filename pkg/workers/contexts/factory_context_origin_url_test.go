package contexts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
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

	t.Run("a subscription workspace keeps the configured link", func(t *testing.T) {
		db := database.Conn()
		factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "")
		require.NoError(t, err)
		subscription := true
		require.NoError(t, factory.UpdateOnboarding(db, models.FactoryOnboardingPatch{CodexSubscription: &subscription}))
		connection, err := models.CreateCodexConnection(db, r.Organization.ID, r.User, time.Now())
		require.NoError(t, err)
		require.NoError(t, db.Model(connection).Updates(map[string]any{"credential_version": 1, "bundle_ciphertext": []byte("bundle")}).Error)
		canvas, execution, run := setupFactoryAppExecutionWithPayload(t, r, factory.ID, map[string]any{"type": "webhook", "data": map[string]any{}})
		version, err := models.FindCanvasVersionInTransaction(db, canvas.ID, run.VersionID)
		require.NoError(t, err)
		version.Nodes = datatypes.JSONSlice[models.Node]{{
			ID: "create-work-order", Type: models.NodeTypeComponent,
			Ref:           models.NodeRef{Component: &models.ComponentRef{Name: "createWorkOrder"}},
			Configuration: map[string]any{"defaultOwnerId": r.User.String()},
		}}
		require.NoError(t, models.SaveCanvasVersionInTransaction(db, version))
		_, err = factory.CreateIntake(db, canvas.ID, models.FactoryIntakeSourceWebhook)
		require.NoError(t, err)
		link := "https://tracker.example.com/items/7"

		created, inserted, err := NewFactoryContext(db, canvas, execution).
			CreateWorkOrder(core.WorkOrderParams{Title: "Checkout fails", OriginURL: link, DefaultOwnerID: r.User.String()})
		require.NoError(t, err)
		require.True(t, inserted)
		persisted, err := factory.FindWorkOrder(db, uuid.MustParse(created.ID))
		require.NoError(t, err)
		require.NotNil(t, persisted.Origin())
		assert.Equal(t, link, persisted.Origin().URL)
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
