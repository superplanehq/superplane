package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeExecutor struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
}

func NewWorkflowNodeExecutor(registry *registry.Registry) *WorkflowNodeExecutor {
	return &WorkflowNodeExecutor{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *WorkflowNodeExecutor) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nodes, err := models.ListWorkflowNodesReady()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, node := range nodes {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(node models.WorkflowNode) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNode(node); err != nil {
						w.log("Error processing workflow node - workflow=%s, node=%s: %v", node.WorkflowID, node.NodeID, err)
					}
				}(node)
			}
		}
	}
}

func (w *WorkflowNodeExecutor) LockAndProcessNode(node models.WorkflowNode) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		n, err := models.LockWorkflowNode(tx, node.WorkflowID, node.NodeID)
		if err != nil {
			w.log("Node already being processed - skipping")
			return nil
		}

		return w.processNode(tx, n)
	})
}

func (w *WorkflowNodeExecutor) processNode(tx *gorm.DB, node *models.WorkflowNode) error {
	execution, err := node.FirstPendingExecution(tx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w.log("No pending execution found for node=%s workflow=%s - skipping", node.NodeID, node.WorkflowID)
			return nil
		}

		return err
	}

	if node.Type == models.NodeTypeBlueprint {
		return w.executeBlueprintNode(tx, execution, node)
	}

	return w.executeComponentNode(tx, execution, node)
}

func (w *WorkflowNodeExecutor) executeBlueprintNode(tx *gorm.DB, execution *models.WorkflowNodeExecution, node *models.WorkflowNode) error {
	ref := node.Ref.Data()
	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, ref.Blueprint.ID)
	if err != nil {
		return fmt.Errorf("blueprint %s not found: %w", ref.Blueprint.ID, err)
	}

	firstNode := blueprint.FindRootNode()
	if firstNode == nil {
		return fmt.Errorf("blueprint %s has no start node", blueprint.ID)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		return fmt.Errorf("error finding input: %v", err)
	}

	config, err := contexts.NewNodeConfigurationBuilder(tx, execution.WorkflowID).
		WithRootEvent(&execution.RootEventID).
		WithPreviousExecution(&execution.ID).
		ForBlueprintNode(node).
		WithInput(input).
		Build(firstNode.Configuration)

	if err != nil {
		return err
	}

	_, err = models.CreatePendingChildExecution(tx, execution, firstNode.ID, config)
	if err != nil {
		return fmt.Errorf("failed to create child execution: %w", err)
	}

	return execution.StartInTransaction(tx)
}

func (w *WorkflowNodeExecutor) executeComponentNode(tx *gorm.DB, execution *models.WorkflowNodeExecution, node *models.WorkflowNode) error {
	err := execution.StartInTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	ref := node.Ref.Data()
	component, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		return fmt.Errorf("failed to get execution inputs: %w", err)
	}

	ctx := components.ExecutionContext{
		Configuration:         execution.Configuration.Data(),
		Data:                  input,
		MetadataContext:       contexts.NewMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
	}

	if err := component.Execute(ctx); err != nil {
		return execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, err.Error())
	}

	w.log("Execute() returned for execution=%s, node=%s", execution.ID, node.NodeID)
	return tx.Save(execution).Error
}

func (w *WorkflowNodeExecutor) log(format string, v ...any) {
	log.Printf("[WorkflowNodeExecutor] "+format, v...)
}
