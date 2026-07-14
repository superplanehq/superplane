package contexts

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/expressionvalidation"
	"github.com/superplanehq/superplane/pkg/exprruntime"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

var expressionRegex = regexp.MustCompile(`\{\{(.*?)\}\}`)

type NodeConfigurationBuilder struct {
	tx                  *gorm.DB
	workflowID          uuid.UUID
	nodeID              string
	previousExecutionID *uuid.UUID
	incomingEventID     *uuid.UUID
	rootEventID         *uuid.UUID
	rootPayload         any
	input               any
	expressionVariables map[string]any
	configurationFields []configuration.Field
}

func NewNodeConfigurationBuilder(tx *gorm.DB, workflowID uuid.UUID) *NodeConfigurationBuilder {
	return &NodeConfigurationBuilder{
		tx:         tx,
		workflowID: workflowID,
	}
}

func (b *NodeConfigurationBuilder) WithNodeID(nodeID string) *NodeConfigurationBuilder {
	b.nodeID = nodeID
	return b
}

func (b *NodeConfigurationBuilder) WithRootEvent(rootEventID *uuid.UUID) *NodeConfigurationBuilder {
	b.rootEventID = rootEventID
	return b
}

// WithRootPayload stores payload normalized for expression evaluation (see normalizeExpressionValue).
func (b *NodeConfigurationBuilder) WithRootPayload(payload any) *NodeConfigurationBuilder {
	b.rootPayload = normalizeExpressionValue(payload)
	return b
}

func (b *NodeConfigurationBuilder) WithPreviousExecution(previousExecutionID *uuid.UUID) *NodeConfigurationBuilder {
	b.previousExecutionID = previousExecutionID
	return b
}

func (b *NodeConfigurationBuilder) WithIncomingEventID(incomingEventID *uuid.UUID) *NodeConfigurationBuilder {
	b.incomingEventID = incomingEventID
	return b
}

// WithInput stores input normalized for expression evaluation (see normalizeExpressionValue).
func (b *NodeConfigurationBuilder) WithInput(input any) *NodeConfigurationBuilder {
	b.input = normalizeExpressionValue(input)
	return b
}

func (b *NodeConfigurationBuilder) WithExpressionVariables(variables map[string]any) *NodeConfigurationBuilder {
	b.expressionVariables = variables
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

func WithoutRunTitleConfiguration(configuration map[string]any) map[string]any {
	if _, ok := configuration["customName"]; !ok {
		return configuration
	}

	result := make(map[string]any, len(configuration)-1)
	for key, value := range configuration {
		if key == "customName" {
			continue
		}
		result[key] = value
	}

	return result
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

	if _, ok := value.(string); ok && !fieldAllowsExpressionResolution(field) {
		return value, nil
	}

	return b.resolveValue(value)
}

// fieldAllowsExpressionResolution reports whether {{ }} placeholders in this
// field should be evaluated. Text fields can opt out via
// TypeOptions.Text.AllowExpressions=false; placeholders are then left as
// literal text (e.g. runner scripts).
func fieldAllowsExpressionResolution(field configuration.Field) bool {
	if field.Type != configuration.FieldTypeText || field.TypeOptions == nil || field.TypeOptions.Text == nil {
		return true
	}

	if field.TypeOptions.Text.AllowExpressions == nil {
		return true
	}

	return *field.TypeOptions.Text.AllowExpressions
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
		return b.ResolveTemplateExpressions(v)

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

func (b *NodeConfigurationBuilder) ResolveTemplateExpressions(expression string) (any, error) {
	matches := expressionRegex.FindAllStringIndex(expression, -1)
	if len(matches) == 0 {
		return expression, nil
	}

	var err error

	result := expressionRegex.ReplaceAllStringFunc(expression, func(match string) string {
		matches := expressionRegex.FindStringSubmatch(match)
		if len(matches) != 2 {
			return match
		}

		value, e := b.ResolveExpression(matches[1])
		if e != nil {
			err = e
			return ""
		}

		return formatTemplateValue(value)
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (b *NodeConfigurationBuilder) ResolveExpression(expression string) (any, error) {
	return b.ResolveExpressionWithExtraVariables(expression, b.expressionVariables)
}

// ResolveExpressionWithExtraVariables evaluates an expression with extra
// variables merged into the eval environment. Provided keys cannot override
// built-ins; we reject any attempt to shadow reserved names so that `$`,
// `memory`, `config`, `root`, and `previous` stay deterministic.
func (b *NodeConfigurationBuilder) ResolveExpressionWithExtraVariables(expression string, variables map[string]any) (any, error) {
	referencedNodes, err := expressionvalidation.ParseReferencedNodes(expression)
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

	for key, value := range variables {
		if isReservedExpressionIdentifier(key) {
			return "", fmt.Errorf("variable %q is reserved", key)
		}
		env[key] = value
	}

	exprOptions := []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
		exprruntime.DateFunctionOption(),
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
	var executionByNodeID map[string]models.CanvasNodeExecution
	if b.previousExecutionID != nil {
		executionsInChain, err := b.listExecutionsInChain()
		if err != nil {
			return nil, err
		}

		//
		// listExecutionsInChain merges the current execution's linear lineage
		// (walked via previous_execution_id) with every run-wide execution of
		// upstream nodes. When the graph has a feedback cycle (e.g. a loop whose
		// body points back at the loop), an upstream node has one execution per
		// iteration, so the same name maps to many executions in the run. We must
		// bind each name to the execution in the *current* lineage; otherwise
		// expressions like a loop's until-condition would read a stale
		// iteration's output. So a linear-chain execution always wins over a
		// run-wide upstream match for the same node.
		//
		linearExecutions, err := b.listLinearExecutionsInChain()
		if err != nil {
			return nil, err
		}
		linearExecutionIDs := make(map[uuid.UUID]struct{}, len(linearExecutions))
		for _, execution := range linearExecutions {
			linearExecutionIDs[execution.ID] = struct{}{}
		}

		executionByNodeID = make(map[string]models.CanvasNodeExecution, len(executionsInChain))
		for _, execution := range executionsInChain {
			executionChainNodeIDs = append(executionChainNodeIDs, execution.NodeID)

			if existing, ok := executionByNodeID[execution.NodeID]; ok {
				_, existingIsLinear := linearExecutionIDs[existing.ID]
				_, candidateIsLinear := linearExecutionIDs[execution.ID]
				if existingIsLinear && !candidateIsLinear {
					continue
				}
			}
			executionByNodeID[execution.NodeID] = execution
		}
	}

	// Also include the root event's node (triggers don't create executions)
	if rootEvent != nil && rootEvent.NodeID != "" {
		executionChainNodeIDs = append(executionChainNodeIDs, rootEvent.NodeID)
	}

	nodeRefs, err := b.resolveNodeRefs(referencedNodes, executionChainNodeIDs)
	if err != nil {
		return nil, err
	}

	for _, nodeRef := range nodeRefs.unresolved {
		messageChain[nodeRef] = nil
	}

	chainRefs := populateFromInputOrRoot(messageChain, inputMap, rootEvent, nodeRefs.byRef)

	if len(chainRefs) == 0 {
		b.injectConfigIntoMessageChain(messageChain, nodeRefs.byRef, executionByNodeID)
		return messageChain, nil
	}

	if b.previousExecutionID == nil {
		for nodeRef := range chainRefs {
			messageChain[nodeRef] = nil
		}

		b.injectConfigIntoMessageChain(messageChain, nodeRefs.byRef, executionByNodeID)
		return messageChain, nil
	}

	err = b.populateFromExecutions(messageChain, chainRefs, executionByNodeID)
	if err != nil {
		return nil, err
	}

	b.injectConfigIntoMessageChain(messageChain, nodeRefs.byRef, executionByNodeID)
	return messageChain, nil
}

// BuildExecutionMessageChain returns upstream node payloads keyed by canvas node name.
// This is the $ object passed to runner JavaScript tasks.
func (b *NodeConfigurationBuilder) BuildExecutionMessageChain() (map[string]any, error) {
	nodeIDToName, err := b.nodeIDToNameMap()
	if err != nil {
		return nil, err
	}

	messageChain := map[string]any{}
	inputMap := extractInputMap(b.input)

	rootEvent, err := b.fetchRootEvent()
	if err != nil {
		return nil, err
	}
	if rootEvent != nil {
		if name, ok := nodeIDToName[rootEvent.NodeID]; ok && name != "" {
			payload := normalizeExpressionValue(rootEvent.Data.Data())
			if rootEvent.ExecutionID != nil {
				execution, execErr := models.FindNodeExecutionInTransaction(b.tx, b.workflowID, *rootEvent.ExecutionID)
				if execErr == nil && execution != nil {
					payload = injectConfig(payload, execution.Configuration.Data())
				}
			}
			messageChain[name] = payload
		}
	}

	executionsInChain, err := b.listExecutionsInChain()
	if err != nil {
		return nil, err
	}
	if len(executionsInChain) > 0 {
		linearExecutions, err := b.listLinearExecutionsInChain()
		if err != nil {
			return nil, err
		}

		outputs, err := newExecutionOutputLookup(
			b.tx,
			linearExecutions,
			executionIDsFromExecutions(executionsInChain),
			b.incomingEventID,
		)
		if err != nil {
			return nil, err
		}

		for i := len(executionsInChain) - 1; i >= 0; i-- {
			execution := executionsInChain[i]
			name, ok := nodeIDToName[execution.NodeID]
			if !ok || name == "" {
				continue
			}

			event, found, err := outputs.outputEvent(execution.ID)
			if err != nil {
				return nil, err
			}
			if !found {
				continue
			}

			messageChain[name] = injectConfig(normalizeExpressionValue(event.Data.Data()), execution.Configuration.Data())
		}
	}

	for nodeID, value := range inputMap {
		name, ok := nodeIDToName[nodeID]
		if !ok || name == "" {
			continue
		}
		messageChain[name] = value
	}

	return messageChain, nil
}

func (b *NodeConfigurationBuilder) nodeIDToNameMap() (map[string]string, error) {
	nodes, err := models.FindCanvasNodesInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(nodes))
	for _, node := range nodes {
		if node.Name == "" {
			continue
		}
		out[node.NodeID] = node.Name
	}
	return out, nil
}

func (b *NodeConfigurationBuilder) injectConfigIntoMessageChain(
	messageChain map[string]any,
	refToNodeID map[string]string,
	executionByNodeID map[string]models.CanvasNodeExecution,
) {
	if len(executionByNodeID) == 0 {
		return
	}

	for nodeRef, nodeID := range refToNodeID {
		execution, ok := executionByNodeID[nodeID]
		if !ok {
			continue
		}

		payload, ok := messageChain[nodeRef]
		if !ok {
			continue
		}

		messageChain[nodeRef] = injectConfig(payload, execution.Configuration.Data())
	}
}

func extractInputMap(input any) map[string]any {
	if inputMap, ok := input.(map[string]any); ok {
		return inputMap
	}

	return map[string]any{}
}

func normalizeExpressionValue(value any) any {
	normalized, _ := normalizeExpressionValueWithChanged(value)
	return normalized
}

func normalizeExpressionValueWithChanged(value any) (any, bool) {
	switch v := value.(type) {
	case json.Number:
		return normalizeJSONNumber(v), true
	case map[string]any:
		return normalizeExpressionMap(v)
	case map[string]string:
		result := make(map[string]any, len(v))
		for key, item := range v {
			result[key] = item
		}
		return result, true
	case []any:
		return normalizeExpressionSlice(v)
	default:
		return v, false
	}
}

func normalizeExpressionMap(value map[string]any) (any, bool) {
	var out map[string]any
	changed := false
	for key, item := range value {
		normalized, itemChanged := normalizeExpressionValueWithChanged(item)
		if !itemChanged {
			continue
		}
		changed = true
		if out == nil {
			out = make(map[string]any, len(value))
			for k, v := range value {
				out[k] = v
			}
		}
		out[key] = normalized
	}
	if !changed {
		return value, false
	}
	return out, true
}

func normalizeExpressionSlice(value []any) (any, bool) {
	var out []any
	changed := false
	for i, item := range value {
		normalized, itemChanged := normalizeExpressionValueWithChanged(item)
		if !itemChanged {
			continue
		}
		changed = true
		if out == nil {
			out = make([]any, len(value))
			copy(out, value)
		}
		out[i] = normalized
	}
	if !changed {
		return value, false
	}
	return out, true
}

func normalizeJSONNumber(value json.Number) any {
	// Use float64 for all parseable numeric tokens so expression envs match
	// standard json.Unmarshal (which never produces int/int). Preferring Int64
	// would make expr perform integer division for int / int operands.
	if number, err := value.Float64(); err == nil {
		return number
	}

	return value
}

func formatTemplateValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

type resolvedNodeRefs struct {
	byRef      map[string]string
	unresolved []string
}

func (b *NodeConfigurationBuilder) resolveNodeRefs(nodeRefs []string, executionChainNodeIDs []string) (resolvedNodeRefs, error) {
	nodes, err := models.FindCanvasNodesInTransaction(b.tx, b.workflowID)
	if err != nil {
		return resolvedNodeRefs{}, err
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

	resolved := resolvedNodeRefs{byRef: make(map[string]string, len(nodeRefs))}
	for _, nodeRef := range nodeRefs {
		if nodeID, ok := nameToNodeID[nodeRef]; ok {
			resolved.byRef[nodeRef] = nodeID
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
				resolved.byRef[nodeRef] = closestNodeID
				continue
			}

			return resolvedNodeRefs{}, fmt.Errorf("node name %s is not unique and none of the matching nodes are in the execution chain", nodeRef)
		}

		resolved.unresolved = append(resolved.unresolved, nodeRef)
	}

	return resolved, nil
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
		if b.rootPayload != nil {
			return b.rootPayload, nil
		}
		return nil, fmt.Errorf("no root event found")
	}

	payload := normalizeExpressionValue(rootEvent.Data.Data())

	if rootEvent.ExecutionID != nil {
		execution, err := models.FindNodeExecutionInTransaction(b.tx, b.workflowID, *rootEvent.ExecutionID)
		if err == nil && execution != nil {
			return injectConfig(payload, execution.Configuration.Data()), nil
		}
	}

	return payload, nil
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
				messageChain[nodeRef] = normalizeExpressionValue(rootEvent.Data.Data())
			}
			continue
		}

		chainRefs[nodeRef] = nodeID
	}

	return chainRefs
}

// injectConfig merges config into payload. payload must already be normalized; configData is normalized here.
func injectConfig(payload any, configData map[string]any) any {
	if len(configData) == 0 {
		return payload
	}

	payloadMap, ok := payload.(map[string]any)
	if !ok {
		return payload
	}

	if _, exists := payloadMap["config"]; exists {
		return payload
	}

	withConfig := make(map[string]any, len(payloadMap)+1)
	for key, value := range payloadMap {
		withConfig[key] = value
	}
	withConfig["config"] = normalizeExpressionValue(configData)
	return withConfig
}

func (b *NodeConfigurationBuilder) populateFromExecutions(
	messageChain map[string]any,
	chainRefs map[string]string,
	executionByNode map[string]models.CanvasNodeExecution,
) error {
	if len(executionByNode) == 0 {
		executionsInChain, err := b.listExecutionsInChain()
		if err != nil {
			return err
		}

		executionByNode = make(map[string]models.CanvasNodeExecution, len(executionsInChain))
		for _, execution := range executionsInChain {
			executionByNode[execution.NodeID] = execution
		}
	}

	executionIDs := make([]uuid.UUID, 0, len(chainRefs))
	executionIDByRef := make(map[string]uuid.UUID, len(chainRefs))
	for nodeRef, nodeID := range chainRefs {
		execution, ok := executionByNode[nodeID]
		if !ok {
			messageChain[nodeRef] = nil
			continue
		}
		executionIDs = append(executionIDs, execution.ID)
		executionIDByRef[nodeRef] = execution.ID
	}

	chainExecutions, err := b.listLinearExecutionsInChain()
	if err != nil {
		return err
	}

	referencedExecutionIDs := make([]uuid.UUID, 0, len(executionIDByRef))
	for _, executionID := range executionIDByRef {
		referencedExecutionIDs = append(referencedExecutionIDs, executionID)
	}

	outputs, err := newExecutionOutputLookup(
		b.tx,
		chainExecutions,
		unionExecutionIDs(referencedExecutionIDs, executionIDsFromExecutions(chainExecutions)),
		b.incomingEventID,
	)
	if err != nil {
		return err
	}

	for nodeRef, executionID := range executionIDByRef {
		event, ok, err := outputs.outputEvent(executionID)
		if err != nil {
			return fmt.Errorf("node %s: %w", nodeRef, err)
		}
		if !ok {
			messageChain[nodeRef] = nil
			continue
		}

		messageChain[nodeRef] = normalizeExpressionValue(event.Data.Data())
	}

	return nil
}

var reservedExpressionIdentifiers = map[string]struct{}{
	"$":        {},
	"memory":   {},
	"config":   {},
	"root":     {},
	"previous": {},
	"ctx":      {},
}

func isReservedExpressionIdentifier(name string) bool {
	_, ok := reservedExpressionIdentifiers[name]
	return ok
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
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0, fmt.Errorf("depth must be an integer")
		}
		if parsed < 1 {
			return 0, fmt.Errorf("depth must be >= 1")
		}
		return int(parsed), nil
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
	var currentOutput currentOutputRef
	if hasInput {
		step++
		currentOutput = b.currentInputOutputRef()
		if step >= depth && inputPayload != nil {
			return b.injectConfigFromPreviousExecution(inputPayload), nil
		}
	}

	if !hasInput {
		incomingPayload, ok, err := b.incomingEventPayload()
		if err != nil {
			return nil, err
		}
		if ok {
			step++
			currentOutput = currentOutputRef{eventID: b.incomingEventID}
			if step >= depth {
				return incomingPayload, nil
			}
		}
	}

	payloads, err := b.latestDirectUpstreamOutputPayloads()
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		if currentOutput.matches(payload.event) {
			continue
		}
		step++
		if step >= depth {
			return payload.data, nil
		}
	}

	step, payload, err := b.resolveFromExecutions(depth, step, step > 0)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		return payload, nil
	}

	return b.resolveFromRoot(depth, step)
}

func (b *NodeConfigurationBuilder) incomingEventPayload() (any, bool, error) {
	if b.incomingEventID == nil {
		return nil, false, nil
	}

	event, err := models.FindCanvasEventInTransaction(b.tx, *b.incomingEventID)
	if err != nil {
		return nil, false, err
	}

	return b.payloadFromEvent(*event), true, nil
}

type canvasEventPayload struct {
	event models.CanvasEvent
	data  any
}

type currentOutputRef struct {
	eventID     *uuid.UUID
	executionID *uuid.UUID
}

func (b *NodeConfigurationBuilder) currentInputOutputRef() currentOutputRef {
	if b.incomingEventID != nil {
		return currentOutputRef{eventID: b.incomingEventID}
	}

	return currentOutputRef{executionID: b.previousExecutionID}
}

func (r currentOutputRef) matches(event models.CanvasEvent) bool {
	if r.eventID != nil {
		return event.ID == *r.eventID
	}

	return r.executionID != nil && event.ExecutionID != nil && *event.ExecutionID == *r.executionID
}

func (b *NodeConfigurationBuilder) latestDirectUpstreamOutputPayloads() ([]canvasEventPayload, error) {
	if b.nodeID == "" || b.rootEventID == nil {
		return nil, nil
	}

	directExecutions, err := b.listDirectUpstreamExecutions()
	if err != nil {
		return nil, err
	}
	if len(directExecutions) == 0 {
		return nil, nil
	}

	linearExecutions, err := b.listLinearExecutionsInChain()
	if err != nil {
		return nil, err
	}
	executions := selectCurrentExecutionsByNode(directExecutions, linearExecutions)

	outputs, err := newExecutionOutputLookup(
		b.tx,
		linearExecutions,
		executionIDsFromExecutions(executions),
		b.incomingEventID,
	)
	if err != nil {
		return nil, err
	}

	payloads := make([]canvasEventPayload, 0, len(executions))
	for _, execution := range executions {
		event, ok, err := outputs.outputEvent(execution.ID)
		if err != nil {
			return nil, fmt.Errorf("node %s: %w", execution.NodeID, err)
		}
		if !ok {
			continue
		}

		payloads = append(payloads, canvasEventPayload{
			event: event,
			data:  injectConfig(normalizeExpressionValue(event.Data.Data()), execution.Configuration.Data()),
		})
	}
	sort.Slice(payloads, func(i, j int) bool {
		return canvasEventAfter(payloads[i].event, payloads[j].event)
	})

	return payloads, nil
}

func selectCurrentExecutionsByNode(
	executions []models.CanvasNodeExecution,
	linearExecutions []models.CanvasNodeExecution,
) []models.CanvasNodeExecution {
	linearByNode := make(map[string]models.CanvasNodeExecution, len(linearExecutions))
	for _, execution := range linearExecutions {
		if _, exists := linearByNode[execution.NodeID]; exists {
			continue
		}
		linearByNode[execution.NodeID] = execution
	}

	selectedByNode := make(map[string]models.CanvasNodeExecution, len(executions))
	for _, execution := range executions {
		if linear, ok := linearByNode[execution.NodeID]; ok {
			selectedByNode[execution.NodeID] = linear
			continue
		}

		selected, exists := selectedByNode[execution.NodeID]
		if !exists || executionCreatedAfter(execution, selected) {
			selectedByNode[execution.NodeID] = execution
		}
	}

	selected := make([]models.CanvasNodeExecution, 0, len(selectedByNode))
	for _, execution := range selectedByNode {
		selected = append(selected, execution)
	}
	return selected
}

func executionCreatedAfter(left models.CanvasNodeExecution, right models.CanvasNodeExecution) bool {
	if left.CreatedAt == nil && right.CreatedAt == nil {
		return left.ID.String() > right.ID.String()
	}
	if left.CreatedAt == nil {
		return false
	}
	if right.CreatedAt == nil {
		return true
	}
	if left.CreatedAt.Equal(*right.CreatedAt) {
		return left.ID.String() > right.ID.String()
	}
	return left.CreatedAt.After(*right.CreatedAt)
}

func canvasEventAfter(left models.CanvasEvent, right models.CanvasEvent) bool {
	if left.CreatedAt == nil && right.CreatedAt == nil {
		return left.ID.String() > right.ID.String()
	}
	if left.CreatedAt == nil {
		return false
	}
	if right.CreatedAt == nil {
		return true
	}
	if left.CreatedAt.Equal(*right.CreatedAt) {
		return left.ID.String() > right.ID.String()
	}
	return left.CreatedAt.After(*right.CreatedAt)
}

func (b *NodeConfigurationBuilder) payloadFromEvent(event models.CanvasEvent) any {
	payload := normalizeExpressionValue(event.Data.Data())
	if event.ExecutionID == nil {
		return payload
	}

	execution, err := models.FindNodeExecutionInTransaction(b.tx, b.workflowID, *event.ExecutionID)
	if err != nil || execution == nil {
		return payload
	}

	return injectConfig(payload, execution.Configuration.Data())
}

func (b *NodeConfigurationBuilder) injectConfigFromPreviousExecution(payload any) any {
	if b.previousExecutionID == nil {
		return payload
	}

	execution, err := models.FindNodeExecutionInTransaction(b.tx, b.workflowID, *b.previousExecutionID)
	if err != nil || execution == nil {
		return payload
	}

	return injectConfig(payload, execution.Configuration.Data())
}

func (b *NodeConfigurationBuilder) resolveFromExecutions(depth int, step int, hasCurrentPayload bool) (int, any, error) {
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
	if hasCurrentPayload {
		startIndex = 1
	}

	outputs, err := newExecutionOutputLookup(
		b.tx,
		executionsInChain,
		executionIDsFromExecutions(executionsInChain),
		b.incomingEventID,
	)
	if err != nil {
		return step, nil, err
	}

	for _, execution := range executionsInChain[startIndex:] {
		step++
		if step < depth {
			continue
		}

		event, ok, err := outputs.outputEvent(execution.ID)
		if err != nil {
			return step, nil, err
		}
		if !ok {
			continue
		}

		if payload := event.Data.Data(); payload != nil {
			return step, injectConfig(normalizeExpressionValue(payload), execution.Configuration.Data()), nil
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
		payload = normalizeExpressionValue(payload)
		if rootEvent.ExecutionID != nil {
			execution, err := models.FindNodeExecutionInTransaction(b.tx, b.workflowID, *rootEvent.ExecutionID)
			if err == nil && execution != nil {
				return injectConfig(payload, execution.Configuration.Data()), nil
			}
		}

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
				values = append(values, normalizeExpressionValue(record.Values.Data()))
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
				return normalizeExpressionValue(record.Values.Data()), nil
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

	_, liveEdges, err := models.FindLiveCanvasSpecInTransaction(b.tx, canvas.ID)
	if err != nil {
		return nil, err
	}
	if len(liveEdges) == 0 {
		return nil, nil
	}

	incoming := make(map[string][]string, len(liveEdges))
	for _, edge := range liveEdges {
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

func (b *NodeConfigurationBuilder) listDirectSourceNodeIDs() ([]string, error) {
	if b.nodeID == "" {
		return nil, nil
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(b.tx, b.workflowID)
	if err != nil {
		return nil, err
	}

	_, liveEdges, err := models.FindLiveCanvasSpecInTransaction(b.tx, canvas.ID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	ids := []string{}
	for _, edge := range liveEdges {
		if edge.TargetID != b.nodeID {
			continue
		}
		if _, exists := seen[edge.SourceID]; exists {
			continue
		}
		seen[edge.SourceID] = struct{}{}
		ids = append(ids, edge.SourceID)
	}

	return ids, nil
}

func (b *NodeConfigurationBuilder) listDirectUpstreamExecutions() ([]models.CanvasNodeExecution, error) {
	if b.rootEventID == nil {
		return nil, nil
	}

	sourceIDs, err := b.listDirectSourceNodeIDs()
	if err != nil {
		return nil, err
	}
	if len(sourceIDs) == 0 {
		return nil, nil
	}

	var executions []models.CanvasNodeExecution
	err = b.tx.
		Where("workflow_id = ? AND root_event_id = ? AND node_id IN ?", b.workflowID, *b.rootEventID, sourceIDs).
		Find(&executions).
		Error
	if err != nil {
		return nil, err
	}

	return executions, nil
}
