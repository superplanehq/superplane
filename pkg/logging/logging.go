package logging

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

func ForEvent(logger *log.Entry, event models.CanvasEvent) *log.Entry {
	return logger.WithFields(log.Fields{
		"event_id": event.ID,
		"node_id":  event.NodeID,
		"channel":  event.Channel,
	})
}

func ForExecution(execution *models.CanvasNodeExecution) *log.Entry {
	return WithExecution(log.NewEntry(log.StandardLogger()), execution)
}

func WithExecution(
	logger *log.Entry,
	execution *models.CanvasNodeExecution,
) *log.Entry {
	return logger.WithFields(log.Fields{
		"root_event": execution.RootEventID,
		"execution":  execution.ID,
	})
}

func ForNode(node models.CanvasNode) *log.Entry {
	return WithNode(log.NewEntry(log.StandardLogger()), node)
}

func WithNode(logger *log.Entry, node models.CanvasNode) *log.Entry {
	return logger.WithFields(log.Fields{
		"node_id": node.NodeID,
	})
}

func WithQueueItem(logger *log.Entry, queueItem models.CanvasNodeQueueItem) *log.Entry {
	return logger.WithFields(log.Fields{
		"queue_item_id": queueItem.ID,
		"root_event":    queueItem.RootEventID,
	})
}

func ForIntegration(integration models.Integration) *log.Entry {
	return WithIntegration(log.NewEntry(log.StandardLogger()), integration)
}

func WithIntegration(logger *log.Entry, integration models.Integration) *log.Entry {
	return logger.WithFields(log.Fields{
		"integration_name": integration.AppName,
		"integration_id":   integration.ID,
	})
}

func WithWebhook(logger *log.Entry, webhook models.Webhook) *log.Entry {
	return logger.WithFields(log.Fields{
		"webhook_id": webhook.ID,
	})
}

func WithRun(logger *log.Entry, run models.CanvasRun) *log.Entry {
	return logger.WithFields(log.Fields{
		"run_id":      run.ID,
		"workflow_id": run.WorkflowID,
	})
}
