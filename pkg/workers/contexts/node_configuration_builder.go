package contexts

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/parser"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var expressionRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)
var previousDepthRegex = regexp.MustCompile(`\bprevious\s*\(([^)]*)\)`)

type NodeConfigurationBuilder struct {
	tx                  *gorm.DB
	workflowID          uuid.UUID
	nodeID              string
	previousExecutionID *uuid.UUID
	rootEventID         *uuid.UUID
	input               any
	parentBlueprintNode *models.CanvasNode
	configurationFields []configuration.Field
}

func NewNodeConfigurationBuilder(tx *gorm.DB, workflowID uuid.UUID) *NodeConfigurationBuilder {
	return &NodeConfigurationBuilder{
		tx:         tx,
		workflowID: workflowID,
	}
}

func (b *NodeConfigurationBuilder) ForBlueprintNode(parentBlueprintNode *models.CanvasNode) *NodeConfigurationBuilder {
	b.parentBlueprintNode = parentBlueprintNode
	return b
}

func (b *NodeConfigurationBuilder) WithNodeID(nodeID string) *NodeConfigurationBuilder {
	b.nodeID = nodeID
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

func (b *NodeConfigurationBuilder) WithConfigurationFields(fields []configuration.Field) *NodeConfigurationBuilder {
	b.configurationFields = fields
	return b
}

func (b *NodeConfigurationBuilder) Build(configuration map[string]any) (map[string]any, error) {
	if len(b.configurationFields) > 0 {
		return b.resolveWithSchema(configuration, b.configurationFields)
	}

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

func (b *NodeConfigurationBuilder) resolveWithSchema(config map[string]any, fields []configuration.Field) (map[string]any, error) {
	result := make(map[string]any, len(config))
	fieldsByName := make(map[string]configuration.Field, len(fields))
	for _, field := range fields {
		fieldsByName[field.Name] = field
	}

	for key, value := range config {
		field, ok := fieldsByName[key]
		if !ok {
			resolved, err := b.resolveValue(value)
			if err != nil {
				return nil, fmt.Errorf("error resolving field %s: %w", key, err)
			}
			result[key] = resolved
			continue
		}

		resolved, err := b.resolveFieldValue(value, field)
		if err != nil {
			return nil, fmt.Errorf("error resolving field %s: %w", key, err)
		}
		result[key] = resolved
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) resolveFieldValue(value any, field configuration.Field) (any, error) {
	if field.DisallowExpression {
		return value, nil
	}

	if field.TypeOptions != nil {
		if field.TypeOptions.Object != nil && len(field.TypeOptions.Object.Schema) > 0 {
			if obj, ok := asAnyMap(value); ok {
				return b.resolveWithSchema(obj, field.TypeOptions.Object.Schema)
			}
		}

		if field.TypeOptions.List != nil && field.TypeOptions.List.ItemDefinition != nil {
			if list, ok := value.([]any); ok {
				return b.resolveListItems(list, field.TypeOptions.List.ItemDefinition)
			}
		}
	}

	return b.resolveValue(value)
}

func (b *NodeConfigurationBuilder) resolveListItems(list []any, itemDef *configuration.ListItemDefinition) ([]any, error) {
	result := make([]any, len(list))
	for i, item := range list {
		if itemDef.Type == configuration.FieldTypeObject && len(itemDef.Schema) > 0 {
			if itemMap, ok := asAnyMap(item); ok {
				resolved, err := b.resolveWithSchema(itemMap, itemDef.Schema)
				if err != nil {
					return nil, fmt.Errorf("list item %d: %w", i, err)
				}
				result[i] = resolved
				continue
			}
		}

		resolved, err := b.resolveValue(item)
		if err != nil {
			return nil, fmt.Errorf("list item %d: %w", i, err)
		}
		result[i] = resolved
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

func asAnyMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[string]string:
		anyMap := make(map[string]any, len(typed))
		for key, value := range typed {
			anyMap[key] = value
		}
		return anyMap, true
	default:
		return nil, false
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

func (b *NodeConfigurationBuilder) BuildMessageChainForExpression(expression string) (map[string]any, error) {
	referencedNodes, err := parseReferencedNodes(expression)
	if err != nil {
		return nil, err
	}

	return b.buildMessageChain(referencedNodes)
}

func (b *NodeConfigurationBuilder) BuildExpressionEnv(expression string) (map[string]any, error) {
	messageChain, err := b.BuildMessageChainForExpression(expression)
	if err != nil {
		return nil, err
	}

	env := map[string]any{
		"$":      messageChain,
		"memory": b.buildMemoryExpressionNamespace(),
	}

	if strings.Contains(expression, "root(") {
		rootPayload, err := b.resolveRootPayload()
		if err != nil {
			return nil, err
		}
		env["__root"] = rootPayload
	}

	depths, err := parsePreviousDepths(expression)
	if err != nil {
		return nil, err
	}
	if len(depths) > 0 {
		previousByDepth := make(map[string]any, len(depths))
		for _, depth := range depths {
			payload, err := b.resolvePreviousPayload(depth)
			if err != nil {
				return nil, err
			}
			previousByDepth[strconv.Itoa(depth)] = payload
		}
		env["__previousByDepth"] = previousByDepth
	}

	return env, nil
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

	env := map[string]any{
		"$":      messageChain,
		"memory": b.buildMemoryExpressionNamespace(),
	}

	if b.parentBlueprintNode != nil {
		env["config"] = b.parentBlueprintNode.Configuration.Data()
	}

	exprOptions := []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
		expr.Function("root", func(params ...any) (any, error) {
			if len(params) != 0 {
				return nil, fmt.Errorf("root() takes no arguments")
			}

			return b.resolveRootPayload()
		}),
		expr.Function("previous", func(params ...any) (any, error) {
			depth := 1
			if len(params) > 1 {
				return nil, fmt.Errorf("previous() accepts zero or one argument")
			}
			if len(params) == 1 {
				parsedDepth, err := parseDepth(params[0])
				if err != nil {
					return nil, err
				}
				depth = parsedDepth
			}

			return b.resolvePreviousPayload(depth)
		}),
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

	// Fetch root event early - needed for both chain resolution and data population
	rootEvent, err := b.fetchRootEvent()
	if err != nil {
		return nil, err
	}

	// Get execution chain first to help resolve ambiguous names
	var executionChainNodeIDs []string
	if b.previousExecutionID != nil {
		executionsInChain, err := b.listExecutionsInChain()
		if err != nil {
			return nil, err
		}
		for _, execution := range executionsInChain {
			executionChainNodeIDs = append(executionChainNodeIDs, execution.NodeID)
		}
	}

	// Also include the root event's node (triggers don't create executions)
	if rootEvent != nil && rootEvent.NodeID != "" {
		executionChainNodeIDs = append(executionChainNodeIDs, rootEvent.NodeID)
	}

	refToNodeID, err := b.resolveNodeRefs(referencedNodes, executionChainNodeIDs)
	if err != nil {
		return nil, err
	}

	chainRefs := populateFromInputOrRoot(messageChain, inputMap, rootEvent, refToNodeID)

	if len(chainRefs) == 0 {
		return messageChain, nil
	}

	if b.previousExecutionID == nil {
		return nil, fmt.Errorf("node name %s not found in execution chain", firstChainRef(chainRefs))
	}

	err = b.populateFromExecutions(messageChain, chainRefs)
	if err != nil {
		return nil, err
	}

	return messageChain, nil
}

func firstChainRef(chainRefs map[string]string) string {
	for nodeRef := range chainRefs {
		return nodeRef
	}
	return ""
}

func extractInputMap(input any) map[string]any {
	inputMap := map[string]any{}
	if input, ok := input.(map[string]any); ok {
		return input
	}

	return inputMap
}

func (b *NodeConfigurationBuilder) resolveNodeRefs(nodeRefs []string, executionChainNodeIDs []string) (map[string]string, error) {
	nodes, err := models.FindCanvasNodesInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}

	nameToNodeID := make(map[string]string, len(nodes))
	ambiguousNames := make(map[string][]string) // name -> list of node IDs with that name
	for _, node := range nodes {
		if node.Name == "" {
			continue
		}

		if nodeIDs, ok := ambiguousNames[node.Name]; ok {
			ambiguousNames[node.Name] = append(nodeIDs, node.NodeID)
			continue
		}

		if existing, ok := nameToNodeID[node.Name]; ok {
			if existing != node.NodeID {
				delete(nameToNodeID, node.Name)
				ambiguousNames[node.Name] = []string{existing, node.NodeID}
			}
			continue
		}

		nameToNodeID[node.Name] = node.NodeID
	}

	// Build a map of node ID to its position in the execution chain (lower = closer)
	executionChainOrder := make(map[string]int, len(executionChainNodeIDs))
	for i, nodeID := range executionChainNodeIDs {
		executionChainOrder[nodeID] = i
	}

	refToNodeID := make(map[string]string, len(nodeRefs))
	for _, nodeRef := range nodeRefs {
		if nodeID, ok := nameToNodeID[nodeRef]; ok {
			refToNodeID[nodeRef] = nodeID
			continue
		}

		if nodeIDs, ok := ambiguousNames[nodeRef]; ok {
			// Find the closest node in the execution chain
			closestNodeID := ""
			closestOrder := -1
			for _, nodeID := range nodeIDs {
				if order, inChain := executionChainOrder[nodeID]; inChain {
					if closestOrder == -1 || order < closestOrder {
						closestOrder = order
						closestNodeID = nodeID
					}
				}
			}

			if closestNodeID != "" {
				refToNodeID[nodeRef] = closestNodeID
				continue
			}

			return nil, fmt.Errorf("node name %s is not unique and none of the matching nodes are in the execution chain", nodeRef)
		}

		return nil, fmt.Errorf("node name %s not found in execution chain", nodeRef)
	}

	return refToNodeID, nil
}

func (b *NodeConfigurationBuilder) fetchRootEvent() (*models.CanvasEvent, error) {
	if b.rootEventID == nil {
		return nil, nil
	}

	rootEvent, err := models.FindCanvasEventInTransaction(b.tx, *b.rootEventID)
	if err != nil {
		return nil, err
	}

	return rootEvent, nil
}

func (b *NodeConfigurationBuilder) resolveRootPayload() (any, error) {
	rootEvent, err := b.fetchRootEvent()
	if err != nil {
		return nil, err
	}
	if rootEvent == nil {
		return nil, fmt.Errorf("no root event found")
	}

	return rootEvent.Data.Data(), nil
}

func populateFromInputOrRoot(messageChain map[string]any, inputMap map[string]any, rootEvent *models.CanvasEvent, refToNodeID map[string]string) map[string]string {
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

	executionByNode := make(map[string]models.CanvasNodeExecution, len(executionsInChain))
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

	events, err := models.ListCanvasEventsForExecutionsInTransaction(b.tx, executionIDs)
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

func latestEventByExecution(events []models.CanvasEvent, executionIDs []uuid.UUID) map[uuid.UUID]models.CanvasEvent {
	latestByExecution := make(map[uuid.UUID]models.CanvasEvent, len(executionIDs))
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

func parsePreviousDepths(expression string) ([]int, error) {
	matches := previousDepthRegex.FindAllStringSubmatch(expression, -1)
	if len(matches) == 0 {
		return nil, nil
	}

	seen := map[int]struct{}{}
	for _, match := range matches {
		raw := strings.TrimSpace(match[1])
		if raw == "" {
			seen[1] = struct{}{}
			continue
		}

		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("depth must be an integer")
		}
		if parsed < 1 {
			return nil, fmt.Errorf("depth must be >= 1")
		}
		seen[parsed] = struct{}{}
	}

	depths := make([]int, 0, len(seen))
	for depth := range seen {
		depths = append(depths, depth)
	}
	sort.Ints(depths)
	return depths, nil
}

func parseDepth(param any) (int, error) {
	switch value := param.(type) {
	case int:
		if value < 1 {
			return 0, fmt.Errorf("depth must be >= 1")
		}
		return value, nil
	case int64:
		if value < 1 {
			return 0, fmt.Errorf("depth must be >= 1")
		}
		return int(value), nil
	case float64:
		parsed := int(value)
		if value != float64(parsed) {
			return 0, fmt.Errorf("depth must be an integer")
		}
		if parsed < 1 {
			return 0, fmt.Errorf("depth must be >= 1")
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("depth must be an integer")
	}
}

func (b *NodeConfigurationBuilder) resolvePreviousPayload(depth int) (any, error) {
	if depth < 1 {
		return nil, fmt.Errorf("depth must be >= 1")
	}

	inputPayload, hasInput, err := b.singleInputPayload()
	if err != nil {
		return nil, err
	}
	step := 0
	if hasInput {
		step++
		if step >= depth && inputPayload != nil {
			return inputPayload, nil
		}
	}

	step, payload, err := b.resolveFromExecutions(depth, step, hasInput)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		return payload, nil
	}

	return b.resolveFromRoot(depth, step)
}

func (b *NodeConfigurationBuilder) resolveFromExecutions(depth int, step int, hasInput bool) (int, any, error) {
	if b.previousExecutionID == nil {
		return step, nil, nil
	}

	executionsInChain, err := b.listLinearExecutionsInChain()
	if err != nil {
		return step, nil, err
	}
	if len(executionsInChain) == 0 {
		return step, nil, nil
	}

	startIndex := 0
	if hasInput {
		startIndex = 1
	}

	executionIDs := make([]uuid.UUID, 0, len(executionsInChain))
	for _, execution := range executionsInChain {
		executionIDs = append(executionIDs, execution.ID)
	}

	events, err := models.ListCanvasEventsForExecutionsInTransaction(b.tx, executionIDs)
	if err != nil {
		return step, nil, err
	}

	latestByExecution := latestEventByExecution(events, executionIDs)
	for _, execution := range executionsInChain[startIndex:] {
		step++
		if step < depth {
			continue
		}

		event, exists := latestByExecution[execution.ID]
		if !exists {
			continue
		}

		if payload := event.Data.Data(); payload != nil {
			return step, payload, nil
		}
	}

	return step, nil, nil
}

func (b *NodeConfigurationBuilder) resolveFromRoot(depth int, step int) (any, error) {
	rootEvent, err := b.fetchRootEvent()
	if err != nil {
		return nil, err
	}
	if rootEvent == nil {
		return nil, nil
	}

	step++
	if step < depth {
		return nil, nil
	}

	if payload := rootEvent.Data.Data(); payload != nil {
		return payload, nil
	}

	return nil, nil
}

func (b *NodeConfigurationBuilder) buildMemoryExpressionNamespace() map[string]any {
	return map[string]any{
		"find": func(params ...any) (any, error) {
			namespace, matches, err := parseMemoryFindParams(params)
			if err != nil {
				return nil, err
			}

			records, err := models.ListCanvasMemoriesByNamespaceAndMatchesInTransaction(b.tx, b.workflowID, namespace, matches)
			if err != nil {
				return nil, err
			}

			values := make([]any, 0, len(records))
			for _, record := range records {
				values = append(values, record.Values.Data())
			}

			return values, nil
		},
		"findFirst": func(params ...any) (any, error) {
			namespace, matches, err := parseMemoryFindParams(params)
			if err != nil {
				return nil, err
			}

			record, err := models.FindFirstCanvasMemoryByNamespaceAndMatchesInTransaction(b.tx, b.workflowID, namespace, matches)
			if err != nil {
				return nil, err
			}
			if record != nil {
				return record.Values.Data(), nil
			}

			return nil, nil
		},
	}
}

func parseMemoryFindParams(params []any) (string, map[string]any, error) {
	if len(params) == 0 || len(params) > 2 {
		return "", nil, fmt.Errorf("memory.find() and memory.findFirst() require a namespace and matches")
	}

	namespace, ok := params[0].(string)
	if !ok {
		return "", nil, fmt.Errorf("memory namespace must be a string")
	}

	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return "", nil, fmt.Errorf("memory namespace is required")
	}

	if len(params) == 1 {
		return namespace, nil, fmt.Errorf("at least one match expression is required")
	}

	matches, ok := params[1].(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("match expression must be an object")
	}

	return namespace, matches, nil
}

func (b *NodeConfigurationBuilder) singleInputPayload() (any, bool, error) {
	inputMap := extractInputMap(b.input)
	if len(inputMap) == 0 {
		return nil, false, nil
	}
	if len(inputMap) > 1 {
		return nil, false, fmt.Errorf("previous() is not available when multiple inputs are present")
	}

	for _, value := range inputMap {
		return value, true, nil
	}

	return nil, false, nil
}

func (b *NodeConfigurationBuilder) listExecutionsInChain() ([]models.CanvasNodeExecution, error) {
	executions, err := b.listLinearExecutionsInChain()
	if err != nil {
		return nil, err
	}

	if b.nodeID == "" || b.rootEventID == nil {
		return executions, nil
	}

	upstreamNodeIDs, err := b.listUpstreamNodeIDs()
	if err != nil {
		return nil, err
	}
	if len(upstreamNodeIDs) == 0 {
		return executions, nil
	}

	var upstreamExecutions []models.CanvasNodeExecution
	err = b.tx.
		Where("workflow_id = ? AND root_event_id = ? AND node_id IN ?", b.workflowID, *b.rootEventID, upstreamNodeIDs).
		Find(&upstreamExecutions).
		Error
	if err != nil {
		return nil, err
	}

	executionByID := make(map[uuid.UUID]models.CanvasNodeExecution, len(executions)+len(upstreamExecutions))
	for _, execution := range executions {
		executionByID[execution.ID] = execution
	}
	for _, execution := range upstreamExecutions {
		executionByID[execution.ID] = execution
	}

	combined := make([]models.CanvasNodeExecution, 0, len(executionByID))
	for _, execution := range executionByID {
		combined = append(combined, execution)
	}
	sort.Slice(combined, func(i, j int) bool {
		left := combined[i].CreatedAt
		right := combined[j].CreatedAt
		if left == nil && right == nil {
			return combined[i].ID.String() > combined[j].ID.String()
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.Equal(*right) {
			return combined[i].ID.String() > combined[j].ID.String()
		}
		return left.After(*right)
	})

	return combined, nil
}

func (b *NodeConfigurationBuilder) listLinearExecutionsInChain() ([]models.CanvasNodeExecution, error) {
	if b.previousExecutionID == nil {
		return nil, nil
	}

	var executions []models.CanvasNodeExecution

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

func (b *NodeConfigurationBuilder) listUpstreamNodeIDs() ([]string, error) {
	if b.nodeID == "" {
		return nil, nil
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}
	if len(canvas.Edges) == 0 {
		return nil, nil
	}

	incoming := make(map[string][]string, len(canvas.Edges))
	for _, edge := range canvas.Edges {
		incoming[edge.TargetID] = append(incoming[edge.TargetID], edge.SourceID)
	}

	seen := map[string]struct{}{}
	queue := []string{b.nodeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, source := range incoming[current] {
			if _, exists := seen[source]; exists {
				continue
			}
			seen[source] = struct{}{}
			queue = append(queue, source)
		}
	}

	ids := make([]string, 0, len(seen))
	for nodeID := range seen {
		ids = append(ids, nodeID)
	}
	sort.Strings(ids)
	return ids, nil
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
