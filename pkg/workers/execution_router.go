package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
)

// ExecutionRouter replaces WorkflowEventRouter
// It routes completed executions to next nodes or completes the workflow
type ExecutionRouter struct{}

func NewExecutionRouter() *ExecutionRouter {
	return &ExecutionRouter{}
}

func (w *ExecutionRouter) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processExecutions(); err != nil {
				log.Printf("[ExecutionRouter] Error processing executions: %v", err)
			}
		}
	}
}

func (w *ExecutionRouter) processExecutions() error {
	executions, err := models.FindRoutingNodeExecutions()
	if err != nil {
		return err
	}

	for _, execution := range executions {
		if err := w.routeExecution(&execution); err != nil {
			w.log("Error routing execution %s: %v", execution.ID, err)
			if err := execution.Fail(models.WorkflowNodeExecutionResultReasonError, err.Error()); err != nil {
				w.log("Error marking execution %s as failed: %v", execution.ID, err)
			}
		}
	}

	return nil
}

func (w *ExecutionRouter) routeExecution(exec *models.WorkflowNodeExecution) error {
	nodes, edges, err := w.findNodesAndEdges(exec)
	if err != nil {
		return fmt.Errorf("failed to determine nodes and edges for execution: %v", err)
	}

	nextNodes, err := w.findNextNodes(exec, edges)
	if err != nil {
		return fmt.Errorf("failed to find next nodes: %w", err)
	}

	// Route to next nodes if they exist
	if len(nextNodes) > 0 {
		w.log("Execution %s: routing to %d next nodes", exec.ID, len(nextNodes))
		return w.createNextExecutions(exec, nextNodes, nodes)
	}

	// No more nodes - handle completion
	w.log("Execution %s: no more nodes", exec.ID)

	if exec.BlueprintID != nil {
		return w.checkBlueprintCompletion(exec, edges)
	}

	return exec.Complete()
}

func (w *ExecutionRouter) findNodesAndEdges(exec *models.WorkflowNodeExecution) ([]models.Node, []models.Edge, error) {
	if exec.BlueprintID != nil {
		return w.findBlueprintNodesAndEdges(exec)
	}

	return w.findWorkflowNodesAndEdges(exec)
}

func (w *ExecutionRouter) findBlueprintNodesAndEdges(exec *models.WorkflowNodeExecution) ([]models.Node, []models.Edge, error) {
	w.log("Execution %s: routing through blueprint %s", exec.ID, *exec.BlueprintID)

	blueprint, err := models.FindBlueprintByID(exec.BlueprintID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find blueprint %s: %w", *exec.BlueprintID, err)
	}

	return blueprint.Nodes, blueprint.Edges, nil
}

func (w *ExecutionRouter) findWorkflowNodesAndEdges(exec *models.WorkflowNodeExecution) ([]models.Node, []models.Edge, error) {
	w.log("Execution %s: routing through workflow %s", exec.ID, exec.WorkflowID)

	workflow, err := models.FindWorkflow(exec.WorkflowID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find workflow %s: %w", exec.WorkflowID, err)
	}

	return workflow.Nodes, workflow.Edges, nil
}

type nextNodeInfo struct {
	NodeID       string
	OutputBranch string
	OutputIndex  int
}

func (w *ExecutionRouter) findNextNodes(exec *models.WorkflowNodeExecution, edges []models.Edge) ([]nextNodeInfo, error) {
	outputs := exec.Outputs.Data()
	var nextNodes []nextNodeInfo

	for _, edge := range edges {
		if edge.SourceID != exec.NodeID {
			continue
		}

		branchData, exists := outputs[edge.Branch]
		if !exists || len(branchData) == 0 {
			continue
		}

		if edge.TargetType == models.EdgeTargetTypeOutputBranch {
			// Blueprint exit - handled in checkBlueprintCompletion
			continue
		}

		if edge.TargetType != models.EdgeTargetTypeNode {
			continue
		}

		// Create child executions for each item in branch (fan-out)
		for idx := range branchData {
			nextNodes = append(nextNodes, nextNodeInfo{
				NodeID:       edge.TargetID,
				OutputBranch: edge.Branch,
				OutputIndex:  idx,
			})
		}
	}

	return nextNodes, nil
}

func (w *ExecutionRouter) createNextExecutions(
	parent *models.WorkflowNodeExecution,
	nextNodes []nextNodeInfo,
	nodes []models.Node,
) error {
	now := time.Now()

	return database.Conn().Transaction(func(tx *gorm.DB) error {
		for _, next := range nextNodes {
			if err := w.createNextExecution(tx, parent, next, nodes, now); err != nil {
				return err
			}
		}

		return parent.CompleteInTransaction(tx)
	})
}

func (w *ExecutionRouter) createNextExecution(
	tx *gorm.DB,
	previous *models.WorkflowNodeExecution,
	next nextNodeInfo,
	nodes []models.Node,
	now time.Time,
) error {
	node := w.findNodeByID(nodes, next.NodeID)
	if node == nil {
		return fmt.Errorf("node %s not found", next.NodeID)
	}

	// Resolve configuration using ConfigurationBuilder if inside a blueprint
	config, err := w.resolveNodeConfiguration(tx, previous, node)
	if err != nil {
		return fmt.Errorf("failed to resolve node configuration: %w", err)
	}

	childExec := models.WorkflowNodeExecution{
		ID:                   uuid.New(),
		WorkflowID:           previous.WorkflowID,
		NodeID:               next.NodeID,
		RootEventID:          previous.RootEventID,
		PreviousExecutionID:  &previous.ID,
		PreviousOutputBranch: &next.OutputBranch,
		PreviousOutputIndex:  &next.OutputIndex,
		ParentExecutionID:    previous.ParentExecutionID,
		BlueprintID:          previous.BlueprintID,
		State:                models.WorkflowNodeExecutionStatePending,
		Configuration:        datatypes.NewJSONType(config),
		CreatedAt:            &now,
		UpdatedAt:            &now,
	}

	if err := tx.Create(&childExec).Error; err != nil {
		return fmt.Errorf("failed to create child execution: %w", err)
	}

	w.log("Created child execution %s for node %s", childExec.ID, next.NodeID)
	return nil
}

func (w *ExecutionRouter) resolveNodeConfiguration(
	tx *gorm.DB,
	previous *models.WorkflowNodeExecution,
	node *models.Node,
) (map[string]any, error) {
	// If not inside a blueprint, use the node configuration as-is
	if previous.BlueprintID == nil {
		return node.Configuration, nil
	}

	// Inside a blueprint: resolve using the parent blueprint node's configuration
	if previous.ParentExecutionID == nil {
		return nil, fmt.Errorf("execution inside blueprint but has no parent execution")
	}

	var parentExec models.WorkflowNodeExecution
	if err := tx.First(&parentExec, "id = ?", previous.ParentExecutionID).Error; err != nil {
		return nil, fmt.Errorf("failed to find parent blueprint execution: %w", err)
	}

	// Use ConfigurationBuilder to resolve expressions
	configBuilder := components.ConfigurationBuilder{}
	resolvedConfig, err := configBuilder.Build(node.Configuration, parentExec.Configuration.Data())
	if err != nil {
		return nil, fmt.Errorf("failed to build configuration: %w", err)
	}

	return resolvedConfig, nil
}

func (w *ExecutionRouter) findNodeByID(nodes []models.Node, nodeID string) *models.Node {
	for _, n := range nodes {
		if n.ID == nodeID {
			return &n
		}
	}
	return nil
}

func (w *ExecutionRouter) checkBlueprintCompletion(exec *models.WorkflowNodeExecution, edges []models.Edge) error {
	w.log("Checking blueprint completion for execution %s", exec.ID)

	if err := exec.Complete(); err != nil {
		return fmt.Errorf("failed to complete terminal execution: %w", err)
	}

	activeCount, err := w.countActiveExecutionsInBlueprint(exec.BlueprintID)
	if err != nil {
		return err
	}

	if activeCount > 0 {
		w.log("Blueprint %s still has %d active executions, not completing yet", *exec.BlueprintID, activeCount)
		return nil
	}

	return w.completeBlueprintWithAllOutputs(exec, edges)
}

func (w *ExecutionRouter) countActiveExecutionsInBlueprint(blueprintID *uuid.UUID) (int64, error) {
	var count int64
	err := database.Conn().Model(&models.WorkflowNodeExecution{}).
		Where("blueprint_id = ?", blueprintID).
		Where("state IN ?", []string{
			models.WorkflowNodeExecutionStatePending,
			models.WorkflowNodeExecutionStateStarted,
			models.WorkflowNodeExecutionStateRouting,
		}).
		Count(&count).
		Error

	if err != nil {
		return 0, fmt.Errorf("failed to count active executions: %w", err)
	}

	return count, nil
}

func (w *ExecutionRouter) completeBlueprintWithAllOutputs(
	terminalExec *models.WorkflowNodeExecution,
	edges []models.Edge,
) error {
	w.log("Completing blueprint %s with all outputs", *terminalExec.BlueprintID)

	blueprintExec, err := w.findBlueprintNodeExecution(terminalExec)
	if err != nil {
		return err
	}

	w.log("Found blueprint node execution: %s", blueprintExec.ID)

	blueprintOutputs, err := w.collectBlueprintOutputs(terminalExec.BlueprintID, edges)
	if err != nil {
		return err
	}

	w.log("Blueprint outputs: %v", blueprintOutputs)

	return w.finalizeBlueprintExecution(blueprintExec, blueprintOutputs)
}

func (w *ExecutionRouter) findBlueprintNodeExecution(terminalExec *models.WorkflowNodeExecution) (*models.WorkflowNodeExecution, error) {
	if terminalExec.ParentExecutionID == nil {
		return nil, fmt.Errorf("execution %s has no parent (not in a blueprint)", terminalExec.ID)
	}

	blueprintExec, err := models.FindNodeExecution(*terminalExec.ParentExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to find blueprint node execution: %w", err)
	}

	return blueprintExec, nil
}

func (w *ExecutionRouter) collectBlueprintOutputs(blueprintID *uuid.UUID, edges []models.Edge) (map[string][]any, error) {
	finishedExecs, err := w.findFinishedExecutionsInBlueprint(blueprintID)
	if err != nil {
		return nil, err
	}

	blueprintOutputs := make(map[string][]any)

	for _, exec := range finishedExecs {
		w.collectOutputsFromExecution(&exec, edges, blueprintOutputs)
	}

	return blueprintOutputs, nil
}

func (w *ExecutionRouter) findFinishedExecutionsInBlueprint(blueprintID *uuid.UUID) ([]models.WorkflowNodeExecution, error) {
	var executions []models.WorkflowNodeExecution
	err := database.Conn().
		Where("blueprint_id = ?", blueprintID).
		Where("state = ?", models.WorkflowNodeExecutionStateFinished).
		Find(&executions).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to find finished executions: %w", err)
	}

	return executions, nil
}

func (w *ExecutionRouter) collectOutputsFromExecution(
	exec *models.WorkflowNodeExecution,
	edges []models.Edge,
	blueprintOutputs map[string][]any,
) {
	for _, edge := range edges {
		if edge.SourceID != exec.NodeID {
			continue
		}

		if edge.TargetType != models.EdgeTargetTypeOutputBranch {
			continue
		}

		outputs := exec.Outputs.Data()
		branchData, exists := outputs[edge.Branch]
		if !exists {
			continue
		}

		exitBranchName := edge.TargetID
		blueprintOutputs[exitBranchName] = append(blueprintOutputs[exitBranchName], branchData...)

		w.log("Collected outputs from node %s to branch %s", exec.NodeID, exitBranchName)
	}
}

func (w *ExecutionRouter) finalizeBlueprintExecution(
	blueprintExec *models.WorkflowNodeExecution,
	blueprintOutputs map[string][]any,
) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := blueprintExec.PassInTransaction(tx, blueprintOutputs); err != nil {
			return fmt.Errorf("failed to pass blueprint execution: %w", err)
		}

		return blueprintExec.RouteInTransaction(tx)
	})
}

func (w *ExecutionRouter) log(format string, v ...any) {
	log.Printf("[ExecutionRouter] "+format, v...)
}
