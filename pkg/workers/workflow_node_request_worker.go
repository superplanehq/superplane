package workers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeRequestWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
	encryptor crypto.Encryptor
}

func NewNodeRequestWorker(encryptor crypto.Encryptor, registry *registry.Registry) *NodeRequestWorker {
	return &NodeRequestWorker{
		encryptor: encryptor,
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *NodeRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			requests, err := models.ListNodeRequests()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordNodeRequestWorkerRequestsCount(context.Background(), len(requests))

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.WorkflowNodeRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.log("Error processing request %s: %v", request.ID, err)
					}

					if request.ExecutionID != nil {
						messages.NewWorkflowExecutionMessage(request.WorkflowID.String(), request.ExecutionID.String(), request.NodeID).Publish()
					}
				}(request)
			}

			telemetry.RecordNodeRequestWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *NodeRequestWorker) LockAndProcessRequest(request models.WorkflowNodeRequest) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockNodeRequest(tx, request.ID)
		if err != nil {
			w.log("Request %s already being processed - skipping", request.ID)
			return nil
		}

		return w.processRequest(tx, r)
	})
}

func (w *NodeRequestWorker) processRequest(tx *gorm.DB, request *models.WorkflowNodeRequest) error {
	switch request.Type {
	case models.NodeRequestTypeInvokeAction:
		return w.invokeAction(tx, request)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (w *NodeRequestWorker) invokeAction(tx *gorm.DB, request *models.WorkflowNodeRequest) error {
	if request.ExecutionID == nil {
		return w.invokeTriggerAction(tx, request)
	}

	return w.invokeComponentAction(tx, request)
}

func (w *NodeRequestWorker) invokeTriggerAction(tx *gorm.DB, request *models.WorkflowNodeRequest) error {
	node, err := models.FindWorkflowNode(tx, request.WorkflowID, request.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	trigger, err := w.registry.GetTrigger(node.Ref.Data().Trigger.Name)
	if err != nil {
		return fmt.Errorf("trigger not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(trigger.Actions(), actionName)
	if actionDef == nil {
		return fmt.Errorf("action '%s' not found for trigger '%s'", actionName, trigger.Name())
	}

	actionCtx := core.TriggerActionContext{
		Name:            actionName,
		Parameters:      spec.InvokeAction.Parameters,
		Configuration:   node.Configuration.Data(),
		MetadataContext: contexts.NewNodeMetadataContext(tx, node),
		EventContext:    contexts.NewEventContext(tx, node),
		RequestContext:  contexts.NewNodeRequestContext(tx, node),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				w.log("app installation %s not found - completing request", *node.AppInstallationID)
				return request.Complete(tx)
			}

			return fmt.Errorf("failed to find app installation: %v", err)
		}

		actionCtx.AppInstallationContext = contexts.NewAppInstallationContext(tx, node, appInstallation, w.encryptor, w.registry)
	}

	err = trigger.HandleAction(actionCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&node).Error
	if err != nil {
		return fmt.Errorf("error saving node after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeComponentAction(tx *gorm.DB, request *models.WorkflowNodeRequest) error {
	execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, *request.ExecutionID)
	if err != nil {
		return fmt.Errorf("execution %s not found: %w", request.ExecutionID, err)
	}

	if execution.ParentExecutionID == nil {
		return w.invokeParentNodeComponentAction(tx, request, execution)
	}

	return w.invokeChildNodeComponentAction(tx, request, execution)
}

func (w *NodeRequestWorker) invokeParentNodeComponentAction(tx *gorm.DB, request *models.WorkflowNodeRequest, execution *models.WorkflowNodeExecution) error {
	node, err := models.FindWorkflowNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Data().Component.Name)
	if err != nil {
		return fmt.Errorf("component not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(component.Actions(), actionName)
	if actionDef == nil {
		return fmt.Errorf("action '%s' not found for component '%s'", actionName, component.Name())
	}

	actionCtx := core.ActionContext{
		Name:                  actionName,
		Configuration:         node.Configuration.Data(),
		Parameters:            spec.InvokeAction.Parameters,
		MetadataContext:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
		RequestContext:        contexts.NewExecutionRequestContext(tx, execution),
		IntegrationContext:    contexts.NewIntegrationContext(tx, w.registry),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		actionCtx.AppInstallationContext = contexts.NewAppInstallationContext(tx, node, appInstallation, w.encryptor, w.registry)
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) invokeChildNodeComponentAction(tx *gorm.DB, request *models.WorkflowNodeRequest, execution *models.WorkflowNodeExecution) error {
	parentExecution, err := models.FindNodeExecutionInTransaction(tx, execution.WorkflowID, *execution.ParentExecutionID)
	if err != nil {
		return fmt.Errorf("parent execution %s not found: %w", execution.ParentExecutionID, err)
	}

	parentNode, err := models.FindWorkflowNode(tx, execution.WorkflowID, parentExecution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, parentNode.Ref.Data().Blueprint.ID)
	if err != nil {
		return fmt.Errorf("blueprint not found: %w", err)
	}

	childNodeID := strings.Split(execution.NodeID, ":")[1]
	childNode, err := blueprint.FindNode(childNodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	component, err := w.registry.GetComponent(childNode.Ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(component.Actions(), actionName)
	if actionDef == nil {
		return fmt.Errorf("action '%s' not found for component '%s'", actionName, component.Name())
	}

	actionCtx := core.ActionContext{
		Name:                  actionName,
		Configuration:         childNode.Configuration,
		Parameters:            spec.InvokeAction.Parameters,
		MetadataContext:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
		RequestContext:        contexts.NewExecutionRequestContext(tx, execution),
		IntegrationContext:    contexts.NewIntegrationContext(tx, w.registry),
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeRequestWorker) log(format string, v ...any) {
	log.Printf("[NodeRequestWorker] "+format, v...)
}

func findAction(actions []core.Action, actionName string) *core.Action {
	for _, action := range actions {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
