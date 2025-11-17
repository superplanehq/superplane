package logging

import (
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
)

func ForEvent(logger *log.Entry, event models.WorkflowEvent) *log.Entry {
	return logger.WithFields(log.Fields{
		"event_id": event.ID,
		"node_id":  event.NodeID,
		"channel":  event.Channel,
	})
}

func ForExecution(
	logger *log.Entry,
	execution *models.WorkflowNodeExecution,
	parent *models.WorkflowNodeExecution,
) *log.Entry {
	logEntry := logger.WithFields(log.Fields{
		"root_event": execution.RootEventID,
		"execution":  execution.ID,
	})

	if parent != nil {
		logEntry = logEntry.WithFields(log.Fields{
			"parent_execution": parent.ID,
			"parent":           parent.NodeID,
		})
	}

	return logEntry
}

func ForNode(logger *log.Entry, node models.WorkflowNode) *log.Entry {
	return logger.WithFields(log.Fields{
		"node_id": node.NodeID,
		"parent":  node.ParentNodeID,
	})
}

func ForQueueItem(logger *log.Entry, queueItem models.WorkflowNodeQueueItem) *log.Entry {
	return logger.WithFields(log.Fields{
		"queue_item_id": queueItem.ID,
		"root_event":    queueItem.RootEventID,
	})
}
