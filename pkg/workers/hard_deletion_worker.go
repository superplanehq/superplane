package workers

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

const (
	// Maximum number of items to process per tick
	MaxHardDeletionBatchSize = 10
)

type HardDeletionWorker struct {
	Registry       *registry.Registry
	CleanupService *ResourceCleanupService
	BatchSize      int
}

func NewHardDeletionWorker(registry *registry.Registry, cleanupService *ResourceCleanupService) *HardDeletionWorker {
	return &HardDeletionWorker{
		Registry:       registry,
		CleanupService: cleanupService,
		BatchSize:      MaxHardDeletionBatchSize,
	}
}

func (w *HardDeletionWorker) Start() {
	for {
		err := w.Tick()
		if err != nil {
			log.Errorf("Error processing hard deletions: %v", err)
		}

		time.Sleep(30 * time.Second)
	}
}

func (w *HardDeletionWorker) Tick() error {
	// Process stages first since they have the most complex dependency chain
	if err := w.processStages(); err != nil {
		log.Errorf("Error processing stages for hard deletion: %v", err)
	}

	// Process event sources
	if err := w.processEventSources(); err != nil {
		log.Errorf("Error processing event sources for hard deletion: %v", err)
	}

	// Process connection groups
	if err := w.processConnectionGroups(); err != nil {
		log.Errorf("Error processing connection groups for hard deletion: %v", err)
	}

	return nil
}

// processStages handles hard deletion of soft-deleted stages following hierarchical dependency chain:
// stages -> stage_events -> events, stage_executions -> execution_resources, connections
func (w *HardDeletionWorker) processStages() error {
	stages, err := models.ListUnscopedSoftDeletedStages(w.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to list soft deleted stages: %v", err)
	}

	for _, stage := range stages {
		logger := log.WithFields(log.Fields{
			"stage_id":   stage.ID,
			"stage_name": stage.Name,
			"canvas_id":  stage.CanvasID,
		})

		logger.Info("Starting hard deletion of stage")

		if err := w.hardDeleteStage(logger, &stage); err != nil {
			logger.Errorf("Failed to hard delete stage: %v", err)
			continue
		}

		logger.Info("Successfully hard deleted stage")
	}

	return nil
}

// hardDeleteStage handles hard deletion of soft-deleted stages following hierarchical dependency chain:
// 1. Delete stage executions and their execution resources FIRST
// (executions have foreign key to stage_events, so must be deleted first)
// 2. Now delete stage events
// 3. Delete connections where this stage is the target
// 4. Clean up integration webhooks if stage has a resource
// 5. Finally, hard delete the stage itself
func (w *HardDeletionWorker) hardDeleteStage(logger *log.Entry, stage *models.Stage) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := stage.DeleteStageExecutionsInTransaction(tx); err != nil {
			return err
		}

		if err := stage.DeleteStageEventsInTransaction(tx); err != nil {
			return err
		}

		if err := stage.DeleteConnectionsInTransaction(tx); err != nil {
			return err
		}

		if stage.ResourceID != nil {
			if err := w.CleanupService.CleanupStageWebhooks(stage); err != nil {
				logger.Warnf("Failed to cleanup stage webhooks: %v", err)
				// Don't fail the transaction, just log the warning
			}
		}

		if err := stage.HardDeleteInTransaction(tx); err != nil {
			return fmt.Errorf("failed to hard delete stage: %v", err)
		}

		logger.Info("Hard deleted stage with all dependencies")
		return nil
	})
}

// processEventSources handles hard deletion of soft-deleted event sources:
// event_sources -> stage_events -> events, connections
func (w *HardDeletionWorker) processEventSources() error {
	eventSources, err := models.ListUnscopedSoftDeletedEventSources(w.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to list soft deleted event sources: %v", err)
	}

	for _, eventSource := range eventSources {
		logger := log.WithFields(log.Fields{
			"event_source_id":   eventSource.ID,
			"event_source_name": eventSource.Name,
			"canvas_id":         eventSource.CanvasID,
		})

		logger.Info("Starting hard deletion of event source")

		if err := w.hardDeleteEventSource(logger, &eventSource); err != nil {
			logger.Errorf("Failed to hard delete event source: %v", err)
			continue
		}

		logger.Info("Successfully hard deleted event source")
	}

	return nil
}

// hardDeleteEventSource handles hard deletion of soft-deleted event sources:
// 1. Delete connections where this event source is the source
// 2. Clean up integration webhooks if event source has a resource
// 3. Finally, hard delete the event source itself
func (w *HardDeletionWorker) hardDeleteEventSource(logger *log.Entry, eventSource *models.EventSource) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := eventSource.DeleteConnectionsInTransaction(tx); err != nil {
			return err
		}

		if eventSource.ResourceID != nil {
			if err := w.CleanupService.CleanupEventSourceWebhooks(eventSource); err != nil {
				logger.Warnf("Failed to cleanup event source webhooks: %v", err)
				// Don't fail the transaction, just log the warning
			}
		}

		if err := eventSource.HardDeleteInTransaction(tx); err != nil {
			return fmt.Errorf("failed to hard delete event source: %v", err)
		}

		logger.Info("Hard deleted event source with all dependencies")
		return nil
	})
}

// processConnectionGroups handles hard deletion of soft-deleted connection groups:
// connection_groups -> connection_group_field_sets -> connection_group_field_set_events, connections
func (w *HardDeletionWorker) processConnectionGroups() error {
	connectionGroups, err := models.ListUnscopedSoftDeletedConnectionGroups(w.BatchSize)
	if err != nil {
		return fmt.Errorf("failed to list soft deleted connection groups: %v", err)
	}

	for _, connectionGroup := range connectionGroups {
		logger := log.WithFields(log.Fields{
			"connection_group_id":   connectionGroup.ID,
			"connection_group_name": connectionGroup.Name,
			"canvas_id":             connectionGroup.CanvasID,
		})

		logger.Info("Starting hard deletion of connection group")

		if err := w.hardDeleteConnectionGroup(logger, &connectionGroup); err != nil {
			logger.Errorf("Failed to hard delete connection group: %v", err)
			continue
		}

		logger.Info("Successfully hard deleted connection group")
	}

	return nil
}

// hardDeleteConnectionGroup handles hard deletion of soft-deleted connection groups:
// 1. Delete connection group field sets (but preserve their events)
// 2. Delete connections where this connection group is the source or target
// 3. Finally, hard delete the connection group itself
func (w *HardDeletionWorker) hardDeleteConnectionGroup(logger *log.Entry, connectionGroup *models.ConnectionGroup) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := connectionGroup.DeleteFieldSetsInTransaction(tx); err != nil {
			return err
		}

		if err := connectionGroup.DeleteConnectionsInTransaction(tx); err != nil {
			return err
		}

		if err := connectionGroup.HardDeleteInTransaction(tx); err != nil {
			return fmt.Errorf("failed to hard delete connection group: %v", err)
		}

		logger.Info("Hard deleted connection group with all dependencies")
		return nil
	})
}
