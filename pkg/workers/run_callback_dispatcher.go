package workers

import (
	"fmt"
	"slices"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

type RunCallbackDispatcher struct {
	tx                 *gorm.DB
	logger             *log.Entry
	registry           *registry.Registry
	run                *models.CanvasRun
	eventCollector     func([]models.CanvasEvent)
	executionCollector func([]models.CanvasNodeExecution)
}

func NewRunCallbackDispatcher(tx *gorm.DB, registry *registry.Registry, run *models.CanvasRun) *RunCallbackDispatcher {
	return &RunCallbackDispatcher{
		tx:       tx,
		logger:   logging.ForRun(*run),
		registry: registry,
		run:      run,
	}
}

func (d *RunCallbackDispatcher) WithEventCollector(fn func([]models.CanvasEvent)) *RunCallbackDispatcher {
	d.eventCollector = fn
	return d
}

func (d *RunCallbackDispatcher) WithExecutionCollector(fn func([]models.CanvasNodeExecution)) *RunCallbackDispatcher {
	d.executionCollector = fn
	return d
}

func (d *RunCallbackDispatcher) DispatchPending() error {
	pendingIndex := slices.IndexFunc(d.run.Callbacks, func(callback core.RunCallback) bool {
		return callback.When == core.RunCallbackWhenPending
	})

	if pendingIndex == -1 {
		return nil
	}

	err := d.dispatchPendingCallback(d.run.Callbacks[pendingIndex])
	if err != nil {
		return fmt.Errorf("dispatch pending callback: %w", err)
	}

	return nil
}

func (d *RunCallbackDispatcher) dispatchPendingCallback(callback core.RunCallback) error {
	switch callback.On {
	case core.RunCallbackOnEntry:
		return d.dispatchPendingCallbackOnEntry(callback)
	case core.RunCallbackOnParent:
		return d.dispatchPendingCallbackOnParent(callback)
	}

	return fmt.Errorf("unsupported callback on: %s", callback.On)
}

func (d *RunCallbackDispatcher) dispatchPendingCallbackOnEntry(callback core.RunCallback) error {
	input, err := d.runInputParameters()
	if err != nil {
		return err
	}

	targetNode, err := d.run.FindTargetNode(d.tx)
	if err != nil {
		return fmt.Errorf("find entry node: %w", err)
	}

	return d.dispatchPendingHookOnEntryNode(callback, targetNode, input)
}

func (d *RunCallbackDispatcher) dispatchPendingCallbackOnParent(callback core.RunCallback) error {
	input, err := d.runInputParameters()
	if err != nil {
		return err
	}

	execution, node, err := d.resolveParent()
	if err != nil {
		return err
	}

	return d.dispatchActionHook(callback, node, execution, input)
}

func (d *RunCallbackDispatcher) dispatchPendingHookOnEntryNode(callback core.RunCallback, node *models.CanvasNode, input map[string]any) error {
	ref := node.Ref.Data()
	if ref.Trigger == nil || ref.Trigger.Name == "" {
		return fmt.Errorf("entry node %s is not a trigger", node.NodeID)
	}

	trigger, err := d.registry.GetTrigger(ref.Trigger.Name)
	if err != nil {
		return fmt.Errorf("get trigger: %w", err)
	}

	_, err = trigger.HandleHook(core.TriggerHookContext{
		Name:          callback.Hook,
		Logger:        logging.ForNode(*node),
		Configuration: node.Configuration.Data(),
		HTTP:          d.registry.HTTPContextInTransaction(d.tx),
		Metadata:      contexts.NewNodeMetadataContext(d.tx, node),
		Requests:      contexts.NewNodeRequestContext(d.tx, node),
		Events:        contexts.NewEventContext(d.tx, node, d.run, d.eventCollector),
		Parameters:    input,
	})
	if err != nil {
		return fmt.Errorf("handle hook: %w", err)
	}

	return nil
}

func (d *RunCallbackDispatcher) DispatchFinished() error {
	finishedIndex := slices.IndexFunc(d.run.Callbacks, func(callback core.RunCallback) bool {
		return callback.When == core.RunCallbackWhenFinished
	})

	if finishedIndex == -1 {
		return nil
	}

	err := d.dispatchFinishedCallback(d.run.Callbacks[finishedIndex])
	if err != nil {
		return fmt.Errorf("dispatch finished callback: %w", err)
	}

	return nil
}

func (d *RunCallbackDispatcher) dispatchFinishedCallback(callback core.RunCallback) error {
	switch callback.On {
	case core.RunCallbackOnEntry:
		return d.dispatchFinishedCallbackOnEntry(callback)
	case core.RunCallbackOnParent:
		return d.dispatchFinishedCallbackOnParent(callback)
	}

	return fmt.Errorf("unsupported callback on: %s", callback.On)
}

func (d *RunCallbackDispatcher) dispatchFinishedCallbackOnEntry(callback core.RunCallback) error {
	params, err := d.runFinishedParameters()
	if err != nil {
		return err
	}

	targetNode, err := d.run.FindTargetNode(d.tx)
	if err != nil {
		return fmt.Errorf("find entry node: %w", err)
	}

	return d.dispatchFinishedHookOnEntryNode(callback, targetNode, params)
}

func (d *RunCallbackDispatcher) dispatchFinishedCallbackOnParent(callback core.RunCallback) error {
	params, err := d.runFinishedParameters()
	if err != nil {
		return err
	}

	execution, node, err := d.resolveParent()
	if err != nil {
		return err
	}

	if execution.State == models.CanvasNodeExecutionStateCancelling {
		return nil
	}

	if execution.State == models.CanvasNodeExecutionStateFinished &&
		execution.Result == models.CanvasNodeExecutionResultCancelled {
		return nil
	}

	return d.dispatchActionHook(callback, node, execution, params)
}

func (d *RunCallbackDispatcher) dispatchFinishedHookOnEntryNode(callback core.RunCallback, node *models.CanvasNode, params map[string]any) error {
	ref := node.Ref.Data()

	if ref.Trigger != nil && ref.Trigger.Name != "" {
		trigger, err := d.registry.GetTrigger(ref.Trigger.Name)
		if err != nil {
			return fmt.Errorf("get trigger: %w", err)
		}

		_, err = trigger.HandleHook(core.TriggerHookContext{
			Name:          callback.Hook,
			Logger:        logging.ForNode(*node),
			Configuration: node.Configuration.Data(),
			HTTP:          d.registry.HTTPContextInTransaction(d.tx),
			Metadata:      contexts.NewNodeMetadataContext(d.tx, node),
			Requests:      contexts.NewNodeRequestContext(d.tx, node),
			Events:        contexts.NewEventContext(d.tx, node, d.run, d.eventCollector),
			Parameters:    params,
		})
		if err != nil {
			return fmt.Errorf("handle hook: %w", err)
		}

		return nil
	}

	if ref.Component == nil || ref.Component.Name == "" {
		return fmt.Errorf("entry node %s has no trigger or component", node.NodeID)
	}

	execution, err := d.findEntryExecution(node.NodeID)
	if err != nil {
		return err
	}

	return d.dispatchActionHook(callback, node, execution, params)
}

func (d *RunCallbackDispatcher) dispatchActionHook(
	callback core.RunCallback,
	node *models.CanvasNode,
	execution *models.CanvasNodeExecution,
	params map[string]any,
) error {
	ref := node.Ref.Data()
	if ref.Component == nil || ref.Component.Name == "" {
		return fmt.Errorf("node %s is not a component", node.NodeID)
	}

	action, err := d.registry.GetAction(ref.Component.Name)
	if err != nil {
		return fmt.Errorf("get action: %w", err)
	}

	err = action.HandleHook(core.ActionHookContext{
		Name:           callback.Hook,
		Logger:         logging.ForNode(*node),
		Configuration:  node.Configuration.Data(),
		HTTP:           d.registry.HTTPContextInTransaction(d.tx),
		Metadata:       contexts.NewExecutionMetadataContext(d.tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(d.tx, execution, d.eventCollector),
		Requests:       contexts.NewExecutionRequestContext(d.tx, execution),
		Parameters:     params,
	})
	if err != nil {
		return fmt.Errorf("handle hook: %w", err)
	}

	if err := d.tx.Save(execution).Error; err != nil {
		return fmt.Errorf("save execution: %w", err)
	}

	if d.executionCollector != nil {
		d.executionCollector([]models.CanvasNodeExecution{*execution})
	}

	return nil
}

func (d *RunCallbackDispatcher) resolveParent() (*models.CanvasNodeExecution, *models.CanvasNode, error) {
	if d.run.ParentExecutionID == nil || d.run.ParentWorkflowID == nil {
		return nil, nil, fmt.Errorf("no parent information")
	}

	execution, err := models.FindNodeExecutionInTransaction(d.tx, *d.run.ParentWorkflowID, *d.run.ParentExecutionID)
	if err != nil {
		return nil, nil, fmt.Errorf("find parent execution: %w", err)
	}

	node, err := models.FindCanvasNode(d.tx, *d.run.ParentWorkflowID, execution.NodeID)
	if err != nil {
		return nil, nil, fmt.Errorf("find parent node: %w", err)
	}

	return execution, node, nil
}

func (d *RunCallbackDispatcher) findEntryExecution(nodeID string) (*models.CanvasNodeExecution, error) {
	var execution models.CanvasNodeExecution
	err := d.tx.
		Where("workflow_id = ?", d.run.WorkflowID).
		Where("run_id = ?", d.run.ID).
		Where("node_id = ?", nodeID).
		First(&execution).
		Error
	if err != nil {
		return nil, fmt.Errorf("find entry execution: %w", err)
	}

	return &execution, nil
}

func (d *RunCallbackDispatcher) runInputParameters() (map[string]any, error) {
	data, err := d.run.Input.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	var input map[string]any
	if err := models.UnmarshalJSONValue(data, &input); err != nil {
		return nil, fmt.Errorf("unmarshal input: %w", err)
	}

	return input, nil
}

func (d *RunCallbackDispatcher) runFinishedParameters() (map[string]any, error) {
	params, err := core.NewRunFinishedCallback(core.NewRun(d.run.ID, d.run.WorkflowID, d.run.Result, nil)).ToParameters()
	if err != nil {
		return nil, fmt.Errorf("build run finished callback: %w", err)
	}

	return params, nil
}
