package logging

import (
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func baseEntry() *log.Entry {
	return log.NewEntry(log.StandardLogger())
}

func TestForEvent(t *testing.T) {
	event := models.CanvasEvent{
		ID:      uuid.New(),
		NodeID:  "node-1",
		Channel: "default",
	}

	entry := ForEvent(baseEntry(), event)

	assert.Equal(t, event.ID, entry.Data["event_id"])
	assert.Equal(t, "node-1", entry.Data["node_id"])
	assert.Equal(t, "default", entry.Data["channel"])
}

func TestForAndWithExecution(t *testing.T) {
	execution := &models.CanvasNodeExecution{
		ID:          uuid.New(),
		RootEventID: uuid.New(),
	}

	entry := WithExecution(baseEntry(), execution)
	assert.Equal(t, execution.ID, entry.Data["execution"])
	assert.Equal(t, execution.RootEventID, entry.Data["root_event"])

	forEntry := ForExecution(execution)
	assert.Equal(t, execution.ID, forEntry.Data["execution"])
	assert.Equal(t, execution.RootEventID, forEntry.Data["root_event"])
}

func TestForAndWithNode(t *testing.T) {
	node := models.CanvasNode{NodeID: "node-42"}

	entry := WithNode(baseEntry(), node)
	assert.Equal(t, "node-42", entry.Data["node_id"])

	forEntry := ForNode(node)
	assert.Equal(t, "node-42", forEntry.Data["node_id"])
}

func TestWithQueueItem(t *testing.T) {
	queueItem := models.CanvasNodeQueueItem{
		ID:          uuid.New(),
		RootEventID: uuid.New(),
	}

	entry := WithQueueItem(baseEntry(), queueItem)
	assert.Equal(t, queueItem.ID, entry.Data["queue_item_id"])
	assert.Equal(t, queueItem.RootEventID, entry.Data["root_event"])
}

func TestForAndWithIntegration(t *testing.T) {
	integration := models.Integration{
		ID:      uuid.New(),
		AppName: "github",
	}

	entry := WithIntegration(baseEntry(), integration)
	assert.Equal(t, "github", entry.Data["integration_name"])
	assert.Equal(t, integration.ID, entry.Data["integration_id"])

	forEntry := ForIntegration(integration)
	assert.Equal(t, "github", forEntry.Data["integration_name"])
	assert.Equal(t, integration.ID, forEntry.Data["integration_id"])
}

func TestWithWebhook(t *testing.T) {
	webhook := models.Webhook{ID: uuid.New()}

	entry := WithWebhook(baseEntry(), webhook)
	assert.Equal(t, webhook.ID, entry.Data["webhook_id"])
}
