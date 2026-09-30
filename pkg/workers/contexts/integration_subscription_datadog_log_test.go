package contexts

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/datadog"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__IntegrationSubscriptionContext_DatadogWebhookDeliveryLog(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	r.Registry.Integrations["datadog"] = &datadog.Datadog{}
	triggerName := (&datadog.OnErrorTrackingAlert{}).Name()

	factory, err := models.CreateFactory(database.Conn(), r.Organization.ID, support.RandomName("factory"), "", "")
	require.NoError(t, err)

	t.Run("delivered error tracking alert names workspace and intake", func(t *testing.T) {
		canvas, node, integration := datadogAlertNode(t, r, triggerName, map[string]any{})
		require.NoError(t, database.Conn().Model(&models.Canvas{}).Where("id = ?", canvas.ID).Update("factory_id", factory.ID).Error)
		intake, err := factory.CreateIntake(database.Conn(), canvas.ID, models.FactoryIntakeSourceDatadog)
		require.NoError(t, err)

		logs := captureDatadogWebhookLogs(t)
		events := sendDatadogAlert(t, r, integration, node, map[string]any{
			"event_type":       datadog.ErrorTrackingAlertEventType,
			"alert_transition": datadog.AlertTransitionTriggered,
			"title":            "checkout new issues",
			"body":             "InventoryTimeout: checkout failed",
		})

		require.Len(t, events, 1)
		lines := datadogWebhookLogLines(t, logs.String())
		require.Len(t, lines, 1)
		assert.Equal(t, "info", lines[0]["level"])
		assert.Equal(t, "delivered", lines[0]["outcome"])
		assert.Equal(t, logging.WebhookLogType, lines[0]["type"])
		assert.Equal(t, logging.DatadogIntegration, lines[0]["integration"])
		assert.Equal(t, datadog.ErrorTrackingAlertEventType, lines[0]["event_type"])
		assert.Equal(t, datadog.AlertTransitionTriggered, lines[0]["alert_transition"])
		assert.Equal(t, r.Organization.ID.String(), lines[0]["organization_id"])
		assert.Equal(t, r.Organization.Name, lines[0]["organization_name"])
		assert.Equal(t, integration.ID.String(), lines[0]["integration_id"])
		assert.Equal(t, factory.ID.String(), lines[0]["workspace_id"])
		assert.Equal(t, factory.Name, lines[0]["workspace_name"])
		assert.Equal(t, intake.ID.String(), lines[0]["intake_id"])
		assert.Equal(t, canvas.Name, lines[0]["intake_name"])
		assert.NotContains(t, logs.String(), "InventoryTimeout")
	})

	t.Run("filtered alert does not write a delivered line", func(t *testing.T) {
		_, node, integration := datadogAlertNode(t, r, triggerName, map[string]any{
			"service": "checkout",
		})

		logs := captureDatadogWebhookLogs(t)
		events := sendDatadogAlert(t, r, integration, node, map[string]any{
			"event_type":       datadog.ErrorTrackingAlertEventType,
			"alert_transition": datadog.AlertTransitionTriggered,
			"title":            "billing new issues",
			"tags":             "service:billing",
		})

		assert.Empty(t, events)
		assert.Empty(t, datadogWebhookLogLines(t, logs.String()))
	})
}

func datadogAlertNode(
	t *testing.T,
	r *support.ResourceRegistry,
	triggerName string,
	configuration map[string]any,
) (*models.Canvas, models.CanvasNode, *models.Integration) {
	t.Helper()

	integration, err := models.CreateIntegration(
		uuid.New(),
		r.Organization.ID,
		datadogIntegrationApp,
		support.RandomName("installation"),
		map[string]any{},
	)
	require.NoError(t, err)

	canvas, nodes := support.CreateCanvas(
		t,
		r.Organization.ID,
		r.User,
		[]models.CanvasNode{
			{
				NodeID:        "trigger-1",
				Name:          "trigger-1",
				Type:          models.NodeTypeTrigger,
				Ref:           datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
				Configuration: datatypes.NewJSONType(configuration),
			},
		},
		nil,
	)
	require.NotNil(t, canvas)
	require.Len(t, nodes, 1)

	node := nodes[0]
	node.AppInstallationID = &integration.ID
	require.NoError(t, database.Conn().Save(&node).Error)
	return canvas, node, integration
}

func sendDatadogAlert(
	t *testing.T,
	r *support.ResourceRegistry,
	integration *models.Integration,
	node models.CanvasNode,
	message map[string]any,
) []models.CanvasEvent {
	t.Helper()

	newEvents := []models.CanvasEvent{}
	ctx := NewIntegrationContext(database.Conn(), &node, integration, r.Encryptor, r.Registry, func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	})
	_, err := ctx.Subscribe(map[string]any{})
	require.NoError(t, err)

	subscriptions, err := ctx.ListSubscriptions()
	require.NoError(t, err)
	require.NotEmpty(t, subscriptions)

	require.NoError(t, subscriptions[0].SendMessage(message))
	return newEvents
}

func captureDatadogWebhookLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	logger := logging.DatadogWebhookLogger()
	previous := logger.Out
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	t.Cleanup(func() {
		logger.SetOutput(previous)
	})
	return buffer
}

func datadogWebhookLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()

	lines := []map[string]any{}
	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		payload := map[string]any{}
		require.NoError(t, json.Unmarshal(line, &payload))
		lines = append(lines, payload)
	}
	return lines
}
