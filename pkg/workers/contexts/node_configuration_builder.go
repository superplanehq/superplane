package contexts

import (
	"fmt"
	"regexp"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var expressionRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

type NodeConfigurationBuilder struct {
	tx                  *gorm.DB
	workflowID          uuid.UUID
	previousExecutionID *uuid.UUID
	rootEventID         *uuid.UUID
	input               any
	parentBlueprintNode *models.WorkflowNode
}

func NewNodeConfigurationBuilder(tx *gorm.DB, workflowID uuid.UUID) *NodeConfigurationBuilder {
	return &NodeConfigurationBuilder{
		tx:         tx,
		workflowID: workflowID,
	}
}

func (b *NodeConfigurationBuilder) ForBlueprintNode(parentBlueprintNode *models.WorkflowNode) *NodeConfigurationBuilder {
	b.parentBlueprintNode = parentBlueprintNode
	return b
}

func (b *NodeConfigurationBuilder) WithRootEvent(rootEventID *uuid.UUID) *NodeConfigurationBuilder {
	b.rootEventID = rootEventID
	return b
}

func (b *NodeConfigurationBuilder) WithPreviousExecution(previousExecutionID *uuid.UUID) *NodeConfigurationBuilder {
	b.previousExecutionID = previousExecutionID
	return b
}

func (b *NodeConfigurationBuilder) WithInput(input any) *NodeConfigurationBuilder {
	b.input = input
	return b
}

func (b *NodeConfigurationBuilder) Build(configuration map[string]any) (map[string]any, error) {
	resolved, err := b.resolve(configuration)
	if err != nil {
		return nil, err
	}

	return resolved, nil
}

func (b *NodeConfigurationBuilder) resolve(configuration map[string]any) (map[string]any, error) {
	result := make(map[string]any, len(configuration))

	for k, v := range configuration {
		resolved, err := b.resolveValue(v)
		if err != nil {
			return nil, fmt.Errorf("error resolving field %s: %w", k, err)
		}
		result[k] = resolved
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) resolveValue(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return b.ResolveExpression(v)

	case map[string]any:
		return b.resolve(v)

	case map[string]string:
		anyMap := make(map[string]any, len(v))
		for key, value := range v {
			anyMap[key] = value
		}

		return b.resolve(anyMap)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			resolved, err := b.resolveValue(item)
			if err != nil {
				return nil, err
			}
			result[i] = resolved
		}
		return result, nil

	default:
		return v, nil
	}
}

func (b *NodeConfigurationBuilder) ResolveExpression(expression string) (any, error) {
	if !expressionRegex.MatchString(expression) {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := b.resolveExpression(matches[1])
		if e != nil {
			err = e
			return ""
		}

		return fmt.Sprintf("%v", value)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) resolveExpression(expression string) (any, error) {
	referencedNodes, err := parseReferencedNodes(expression)
	if err != nil {
		return "", err
	}

	messageChain, err := b.buildMessageChain(referencedNodes)
	if err != nil {
		return "", err
	}

	env := map[string]any{"$": messageChain}

	if b.parentBlueprintNode != nil {
		env["config"] = b.parentBlueprintNode.Configuration.Data()
	}

	exprOptions := []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}

	vm, err := expr.Compile(expression, exprOptions...)
	if err != nil {
		return "", err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return "", fmt.Errorf("expression evaluation failed: %w", err)
	}

	return output, nil
}

func (b *NodeConfigurationBuilder) buildMessageChain(referencedNodes []string) (map[string]any, error) {
	messageChain := map[string]any{}
	inputMap := extractInputMap(b.input)
	for key, value := range inputMap {
		messageChain[key] = value
	}

	if len(referencedNodes) == 0 {
		return messageChain, nil
	}

	unresolved := unresolvedNodeRefs(referencedNodes, messageChain)
	if len(unresolved) == 0 {
		return messageChain, nil
	}

	refToNodeID, err := b.resolveNodeRefs(unresolved)
	if err != nil {
		return nil, err
	}

	rootEvent, err := b.fetchRootEvent()
	if err != nil {
		return nil, err
	}

	chainRefs := populateFromInputOrRoot(messageChain, inputMap, rootEvent, refToNodeID)

	if len(chainRefs) == 0 {
		return messageChain, nil
	}

	if b.previousExecutionID == nil {
		return nil, fmt.Errorf("node %s not found in execution chain", unresolved[0])
	}

	err = b.populateFromExecutions(messageChain, chainRefs)
	if err != nil {
		return nil, err
	}

	return messageChain, nil
}

func extractInputMap(input any) map[string]any {
	inputMap := map[string]any{}
	if input, ok := input.(map[string]any); ok {
		return input
	}

	return inputMap
}

func unresolvedNodeRefs(referencedNodes []string, messageChain map[string]any) []string {
	unresolved := make([]string, 0, len(referencedNodes))
	for _, nodeRef := range referencedNodes {
		if _, ok := messageChain[nodeRef]; !ok {
			unresolved = append(unresolved, nodeRef)
		}
	}

	return unresolved
}

func (b *NodeConfigurationBuilder) resolveNodeRefs(nodeRefs []string) (map[string]string, error) {
	nodes, err := models.FindWorkflowNodesInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}

	nodeIDs := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeIDs[node.NodeID] = struct{}{}
	}

	refToNodeID := make(map[string]string, len(nodeRefs))
	for _, nodeRef := range nodeRefs {
		if _, ok := nodeIDs[nodeRef]; ok {
			refToNodeID[nodeRef] = nodeRef
			continue
		}

		return nil, fmt.Errorf("node %s not found in execution chain", nodeRef)
	}

	return refToNodeID, nil
}

func (b *NodeConfigurationBuilder) fetchRootEvent() (*models.WorkflowEvent, error) {
	if b.rootEventID == nil {
		return nil, nil
	}

	rootEvent, err := models.FindWorkflowEventInTransaction(b.tx, *b.rootEventID)
	if err != nil {
		return nil, err
	}

	return rootEvent, nil
}

func populateFromInputOrRoot(messageChain map[string]any, inputMap map[string]any, rootEvent *models.WorkflowEvent, refToNodeID map[string]string) map[string]string {
	chainRefs := make(map[string]string, len(refToNodeID))
	for nodeRef, nodeID := range refToNodeID {
		if value, ok := inputMap[nodeID]; ok {
			if _, exists := messageChain[nodeRef]; !exists {
				messageChain[nodeRef] = value
			}
			continue
		}

		if rootEvent != nil && rootEvent.NodeID == nodeID {
			if _, exists := messageChain[nodeRef]; !exists {
				messageChain[nodeRef] = rootEvent.Data.Data()
			}
			continue
		}

		chainRefs[nodeRef] = nodeID
	}

	return chainRefs
}

func (b *NodeConfigurationBuilder) populateFromExecutions(messageChain map[string]any, chainRefs map[string]string) error {
	executionsInChain, err := b.listExecutionsInChain()
	if err != nil {
		return err
	}

	executionByNode := make(map[string]models.WorkflowNodeExecution, len(executionsInChain))
	for _, execution := range executionsInChain {
		executionByNode[execution.NodeID] = execution
	}

	executionIDs := make([]uuid.UUID, 0, len(chainRefs))
	executionIDByRef := make(map[string]uuid.UUID, len(chainRefs))
	for nodeRef, nodeID := range chainRefs {
		execution, ok := executionByNode[nodeID]
		if !ok {
			return fmt.Errorf("node %s not found in execution chain", nodeRef)
		}
		executionIDs = append(executionIDs, execution.ID)
		executionIDByRef[nodeRef] = execution.ID
	}

	events, err := models.ListWorkflowEventsForExecutionsInTransaction(b.tx, executionIDs)
	if err != nil {
		return err
	}

	latestByExecution := latestEventByExecution(events, executionIDs)
	for nodeRef, executionID := range executionIDByRef {
		event, ok := latestByExecution[executionID]
		if !ok {
			return fmt.Errorf("node %s has no outputs", nodeRef)
		}

		messageChain[nodeRef] = event.Data.Data()
	}

	return nil
}

func latestEventByExecution(events []models.WorkflowEvent, executionIDs []uuid.UUID) map[uuid.UUID]models.WorkflowEvent {
	latestByExecution := make(map[uuid.UUID]models.WorkflowEvent, len(executionIDs))
	for _, event := range events {
		if event.ExecutionID == nil {
			continue
		}

		latest, ok := latestByExecution[*event.ExecutionID]
		if !ok || event.CreatedAt == nil {
			latestByExecution[*event.ExecutionID] = event
			continue
		}

		if latest.CreatedAt == nil || event.CreatedAt.After(*latest.CreatedAt) {
			latestByExecution[*event.ExecutionID] = event
		}
	}

	return latestByExecution
}

func (b *NodeConfigurationBuilder) listExecutionsInChain() ([]models.WorkflowNodeExecution, error) {
	var executions []models.WorkflowNodeExecution

	err := b.tx.Raw(`
		WITH RECURSIVE execution_chain AS (
			SELECT
				id,
				workflow_id,
				node_id,
				root_event_id,
				event_id,
				previous_execution_id,
				parent_execution_id,
				state,
				result,
				result_reason,
				result_message,
				metadata,
				configuration,
				created_at,
				updated_at
			FROM workflow_node_executions
			WHERE id = ? AND workflow_id = ?

			UNION ALL

			-- Recursive case: Get the previous execution
			SELECT
				wne.id,
				wne.workflow_id,
				wne.node_id,
				wne.root_event_id,
				wne.event_id,
				wne.previous_execution_id,
				wne.parent_execution_id,
				wne.state,
				wne.result,
				wne.result_reason,
				wne.result_message,
				wne.metadata,
				wne.configuration,
				wne.created_at,
				wne.updated_at
			FROM workflow_node_executions wne
			INNER JOIN execution_chain ec ON wne.id = ec.previous_execution_id
			WHERE wne.workflow_id = ?
		)
		SELECT *
		FROM execution_chain
		ORDER BY created_at DESC;
	`, b.previousExecutionID, b.workflowID, b.workflowID).Scan(&executions).Error

	if err != nil {
		return nil, err
	}

	return executions, nil
}

func parseReferencedNodes(expression string) ([]string, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}

	collector := &nodeReferenceCollector{
		seen: make(map[string]struct{}),
	}

	ast.Walk(&tree.Node, collector)

	return collector.identifiers, nil
}

type nodeReferenceCollector struct {
	identifiers []string
	seen        map[string]struct{}
}

func (c *nodeReferenceCollector) Visit(node *ast.Node) {
	member, ok := (*node).(*ast.MemberNode)
	if !ok {
		return
	}

	root, ok := member.Node.(*ast.IdentifierNode)
	if !ok || root.Value != "$" {
		return
	}

	switch property := member.Property.(type) {
	case *ast.StringNode:
		c.add(property.Value)
	case *ast.IdentifierNode:
		c.add(property.Value)
	}
}

func (c *nodeReferenceCollector) add(value string) {
	if value == "" {
		return
	}

	if _, ok := c.seen[value]; ok {
		return
	}

	c.seen[value] = struct{}{}
	c.identifiers = append(c.identifiers, value)
}
