package workers

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__releaseRetiredDatadogMonitor__ClearsMonitorWhenIntegrationIsGone(t *testing.T) {
	r := support.Setup(t)
	canvas, _ := support.CreateCanvas(t, r.Organization.ID, r.User, []models.CanvasNode{
		{
			NodeID: "intake",
			Name:   "Datadog",
			Type:   models.NodeTypeTrigger,
			Ref: datatypes.NewJSONType(models.NodeRef{
				Trigger: &models.TriggerRef{Name: "datadog.onErrorTrackingAlert"},
			}),
			Configuration: datatypes.NewJSONType(map[string]any{"service": "checkout"}),
			Metadata: datatypes.NewJSONType(map[string]any{
				"monitorId":      "9",
				"monitorService": "checkout",
				"subscriptionId": "sub-1",
			}),
		},
	}, nil)

	require.NoError(t, canvas.SoftDelete())

	worker := NewWebhookCleanupWorker(r.Encryptor, r.Registry, "https://example.com")
	worker.releaseRetiredDatadogMonitors()

	retired, err := models.FindUnscopedCanvasNode(database.Conn(), canvas.ID, "intake")
	require.NoError(t, err)
	metadata := retired.Metadata.Data()
	_, hasMonitor := metadata["monitorId"]
	require.False(t, hasMonitor)
	require.Equal(t, "sub-1", metadata["subscriptionId"])
}
