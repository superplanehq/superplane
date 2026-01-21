package loop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

const ComponentName = "loop"

const (
	ModeCount   = "count"
	ModeForEach = "for_each"
	ModeUntil   = "until"
)

const (
	PayloadTypeIteration = "loop.iteration"
	PayloadTypeOutput    = "loop.output"
)

const (
	DefaultIterations    = 10
	DefaultMaxIterations = 10
)

func init() {
	registry.RegisterComponent(ComponentName, &Loop{})
}

type Loop struct{}

type Spec struct {
	Mode            string `json:"mode"`
	Iterations      any    `json:"iterations"`
	ItemsExpression string `json:"itemsExpression"`
	UntilExpression string `json:"untilExpression"`
	MaxIterations   any    `json:"maxIterations"`
}

type loopState struct {
	Mode            string `json:"mode"`
	Iterations      int    `json:"iterations,omitempty"`
	Items           []any  `json:"items,omitempty"`
	UntilExpression string `json:"untilExpression,omitempty"`
	MaxIterations   int    `json:"maxIterations,omitempty"`
	Input           any    `json:"input,omitempty"`
	Completed       int    `json:"completed"`
	Results         []any  `json:"results,omitempty"`
}

func (l *Loop) Name() string {
	return ComponentName
}

func (l *Loop) Label() string {
	return "Loop"
}

func (l *Loop) Description() string {
	return "Repeat nested steps within a loop container"
}

func (l *Loop) Icon() string {
	return "repeat"
}

func (l *Loop) Color() string {
	return "sky"
}

func (l *Loop) OutputChannels(_ any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (l *Loop) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "mode",
			Label:    "Loop Mode",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  ModeCount,
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Count", Value: ModeCount},
						{Label: "For Each", Value: ModeForEach},
						{Label: "Until", Value: ModeUntil},
					},
				},
			},
		},
		{
			Name:        "iterations",
			Label:       "Iterations",
			Type:        configuration.FieldTypeNumber,
			Description: "How many times to run the loop",
			Default:     DefaultIterations,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeCount}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeCount}},
			},
		},
		{
			Name:        "itemsExpression",
			Label:       "Items Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Expression that evaluates to a list of items",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeForEach}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeForEach}},
			},
		},
		{
			Name:        "untilExpression",
			Label:       "Until Expression",
			Type:        configuration.FieldTypeExpression,
			Description: "Stop when this expression evaluates to true",
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeUntil}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeUntil}},
			},
		},
		{
			Name:        "maxIterations",
			Label:       "Max Iterations",
			Type:        configuration.FieldTypeNumber,
			Description: "Safety limit for until loops",
			Default:     DefaultMaxIterations,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "mode", Values: []string{ModeUntil}},
			},
			RequiredConditions: []configuration.RequiredCondition{
				{Field: "mode", Values: []string{ModeUntil}},
			},
		},
	}
}

func (l *Loop) Execute(ctx core.ExecutionContext) error {
	if ctx.NodeMetadata == nil {
		return fmt.Errorf("loop requires node metadata")
	}

	if ctx.RootEventID == uuid.Nil {
		return fmt.Errorf("loop requires root event id")
	}

	spec := Spec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if strings.TrimSpace(spec.Mode) == "" {
		spec.Mode = ModeCount
	}

	workflowID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return fmt.Errorf("invalid workflow id: %w", err)
	}

	workflow, err := models.FindWorkflowWithoutOrgScope(workflowID)
	if err != nil {
		return fmt.Errorf("failed to find workflow: %w", err)
	}

	isInternal := isChildOfLoop(workflow.Nodes, ctx.NodeID, ctx.SourceNodeID)

	metadata, err := normalizeMetadata(ctx.NodeMetadata.Get())
	if err != nil {
		return err
	}

	state, hasState, err := getLoopState(metadata, ctx.RootEventID)
	if err != nil {
		return err
	}

	if isInternal {
		if !hasState {
			return fmt.Errorf("loop state not found for root event %s", ctx.RootEventID)
		}
		return l.handleInternalEvent(ctx, metadata, state)
	}

	return l.handleExternalEvent(ctx, metadata, spec)
}

func (l *Loop) handleExternalEvent(ctx core.ExecutionContext, metadata map[string]any, spec Spec) error {
	state, totalIterations, err := l.initializeState(ctx, spec)
	if err != nil {
		return err
	}

	if totalIterations == 0 {
		clearLoopState(metadata, ctx.RootEventID)
		if err := ctx.NodeMetadata.Set(metadata); err != nil {
			return err
		}
		payload := buildOutputPayload(state, nil, false)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeOutput, []any{payload})
	}

	if err := setLoopState(metadata, ctx.RootEventID, state); err != nil {
		return err
	}
	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return err
	}

	payload, err := buildIterationPayload(state, 0, nil)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeIteration, []any{payload})
}

func (l *Loop) handleInternalEvent(ctx core.ExecutionContext, metadata map[string]any, state loopState) error {
	currentIndex := state.Completed
	lastOutput := ctx.Data

	state.Results = append(state.Results, lastOutput)
	state.Completed++

	stop := false
	maxReached := false

	switch state.Mode {
	case ModeCount:
		if state.Completed >= state.Iterations {
			stop = true
		}
	case ModeForEach:
		if state.Completed >= len(state.Items) {
			stop = true
		}
	case ModeUntil:
		loopEnv := loopContext(state, currentIndex, lastOutput)
		matches, err := evaluateUntilExpression(ctx, state.UntilExpression, loopEnv)
		if err != nil {
			return err
		}
		if matches {
			stop = true
		}
		if !stop && state.Completed >= state.MaxIterations {
			stop = true
			maxReached = true
		}
	default:
		return fmt.Errorf("unsupported loop mode %s", state.Mode)
	}

	if stop {
		clearLoopState(metadata, ctx.RootEventID)
		if err := ctx.NodeMetadata.Set(metadata); err != nil {
			return err
		}
		payload := buildOutputPayload(state, lastOutput, maxReached)
		return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeOutput, []any{payload})
	}

	if err := setLoopState(metadata, ctx.RootEventID, state); err != nil {
		return err
	}
	if err := ctx.NodeMetadata.Set(metadata); err != nil {
		return err
	}

	payload, err := buildIterationPayload(state, state.Completed, lastOutput)
	if err != nil {
		return err
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, PayloadTypeIteration, []any{payload})
}

func (l *Loop) initializeState(ctx core.ExecutionContext, spec Spec) (loopState, int, error) {
	state := loopState{
		Mode:  spec.Mode,
		Input: ctx.Data,
	}

	switch spec.Mode {
	case ModeCount:
		iterations, err := parseIntegerValue(spec.Iterations)
		if err != nil {
			return loopState{}, 0, fmt.Errorf("invalid iterations: %w", err)
		}
		if iterations < 0 {
			return loopState{}, 0, fmt.Errorf("iterations must be >= 0")
		}
		state.Iterations = iterations
		return state, iterations, nil
	case ModeForEach:
		if strings.TrimSpace(spec.ItemsExpression) == "" {
			return loopState{}, 0, fmt.Errorf("itemsExpression is required for for_each mode")
		}
		items, err := evaluateItemsExpression(ctx, spec.ItemsExpression)
		if err != nil {
			return loopState{}, 0, err
		}
		state.Items = items
		return state, len(items), nil
	case ModeUntil:
		if strings.TrimSpace(spec.UntilExpression) == "" {
			return loopState{}, 0, fmt.Errorf("untilExpression is required for until mode")
		}
		maxIterations, err := parseIntegerValue(spec.MaxIterations)
		if err != nil {
			return loopState{}, 0, fmt.Errorf("invalid maxIterations: %w", err)
		}
		if maxIterations <= 0 {
			return loopState{}, 0, fmt.Errorf("maxIterations must be > 0")
		}
		state.UntilExpression = spec.UntilExpression
		state.MaxIterations = maxIterations
		return state, maxIterations, nil
	default:
		return loopState{}, 0, fmt.Errorf("unsupported loop mode %s", spec.Mode)
	}
}

func loopContext(state loopState, index int, last any) map[string]any {
	ctx := map[string]any{
		"mode":      state.Mode,
		"iteration": index + 1,
		"index":     index,
		"input":     state.Input,
		"last":      last,
		"results":   state.Results,
	}

	switch state.Mode {
	case ModeCount:
		ctx["total"] = state.Iterations
	case ModeForEach:
		ctx["total"] = len(state.Items)
		if index >= 0 && index < len(state.Items) {
			ctx["item"] = state.Items[index]
		}
	case ModeUntil:
		ctx["maxIterations"] = state.MaxIterations
	}

	return ctx
}

func buildIterationPayload(state loopState, index int, last any) (map[string]any, error) {
	payload := map[string]any{
		"mode":      state.Mode,
		"iteration": index + 1,
		"index":     index,
		"input":     state.Input,
		"last":      last,
	}

	switch state.Mode {
	case ModeCount:
		payload["total"] = state.Iterations
	case ModeForEach:
		if index < 0 || index >= len(state.Items) {
			return nil, fmt.Errorf("iteration index %d out of range", index)
		}
		payload["item"] = state.Items[index]
		payload["total"] = len(state.Items)
	case ModeUntil:
		payload["maxIterations"] = state.MaxIterations
	default:
		return nil, fmt.Errorf("unsupported loop mode %s", state.Mode)
	}

	return payload, nil
}

func buildOutputPayload(state loopState, last any, maxReached bool) map[string]any {
	payload := map[string]any{
		"mode":       state.Mode,
		"iterations": state.Completed,
		"results":    state.Results,
		"last":       last,
		"input":      state.Input,
	}

	switch state.Mode {
	case ModeCount:
		payload["total"] = state.Iterations
	case ModeForEach:
		payload["items"] = state.Items
		payload["total"] = len(state.Items)
	case ModeUntil:
		payload["maxIterations"] = state.MaxIterations
		payload["maxReached"] = maxReached
	}

	return payload
}

func evaluateItemsExpression(ctx core.ExecutionContext, expression string) ([]any, error) {
	env, err := expressionEnv(ctx, expression)
	if err != nil {
		return nil, err
	}

	vm, err := expr.Compile(expression, []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)
	if err != nil {
		return nil, err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return nil, fmt.Errorf("items expression evaluation failed: %w", err)
	}

	return toAnySlice(output)
}

func evaluateUntilExpression(ctx core.ExecutionContext, expression string, loopEnv map[string]any) (bool, error) {
	env, err := expressionEnv(ctx, expression)
	if err != nil {
		return false, err
	}
	if env == nil {
		env = map[string]any{}
	}
	env["loop"] = loopEnv

	vm, err := expr.Compile(expression, []expr.Option{
		expr.Env(env),
		expr.AsBool(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)
	if err != nil {
		return false, err
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return false, fmt.Errorf("until expression evaluation failed: %w", err)
	}

	matches, ok := output.(bool)
	if !ok {
		return false, fmt.Errorf("until expression must evaluate to boolean, got %T", output)
	}

	return matches, nil
}

func expressionEnv(ctx core.ExecutionContext, expression string) (map[string]any, error) {
	if ctx.ExpressionEnv != nil {
		return ctx.ExpressionEnv(expression)
	}

	return buildExpressionEnv(ctx.Data, ctx.SourceNodeID), nil
}

func buildExpressionEnv(input any, sourceNodeID string) map[string]any {
	if sourceNodeID == "" {
		return map[string]any{"$": input}
	}

	if inputMap, ok := input.(map[string]any); ok {
		envData := make(map[string]any, len(inputMap)+1)
		for key, value := range inputMap {
			envData[key] = value
		}
		if _, exists := envData[sourceNodeID]; !exists {
			envData[sourceNodeID] = input
		}
		return map[string]any{"$": envData}
	}

	if inputMap, ok := input.(map[string]string); ok {
		envData := make(map[string]any, len(inputMap)+1)
		for key, value := range inputMap {
			envData[key] = value
		}
		if _, exists := envData[sourceNodeID]; !exists {
			envData[sourceNodeID] = input
		}
		return map[string]any{"$": envData}
	}

	return map[string]any{"$": map[string]any{sourceNodeID: input}}
}

func isChildOfLoop(nodes []models.Node, loopID string, nodeID string) bool {
	if loopID == "" || nodeID == "" {
		return false
	}

	for _, node := range nodes {
		if node.ID != nodeID {
			continue
		}
		return parentNodeID(node.Metadata) == loopID
	}

	return false
}

func parentNodeID(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}

	value, ok := metadata["parentNodeId"]
	if !ok {
		return ""
	}

	parentID, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(parentID)
}

func normalizeMetadata(raw any) (map[string]any, error) {
	if raw == nil {
		return map[string]any{}, nil
	}

	if metadata, ok := raw.(map[string]any); ok {
		return metadata, nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	metadata := map[string]any{}
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	return metadata, nil
}

func getLoopState(metadata map[string]any, rootEventID uuid.UUID) (loopState, bool, error) {
	state := loopState{}

	raw, ok := metadata["loopState"]
	if !ok {
		return state, false, nil
	}

	states, ok := raw.(map[string]any)
	if !ok {
		return state, false, nil
	}

	rootKey := rootEventID.String()
	stateRaw, ok := states[rootKey]
	if !ok {
		return state, false, nil
	}

	stateMap, ok := stateRaw.(map[string]any)
	if !ok {
		return state, false, nil
	}

	if err := mapstructure.Decode(stateMap, &state); err != nil {
		return loopState{}, false, fmt.Errorf("invalid loop state: %w", err)
	}

	return state, true, nil
}

func setLoopState(metadata map[string]any, rootEventID uuid.UUID, state loopState) error {
	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to encode loop state: %w", err)
	}

	stateMap := map[string]any{}
	if err := json.Unmarshal(payload, &stateMap); err != nil {
		return fmt.Errorf("failed to encode loop state: %w", err)
	}

	states := map[string]any{}
	if existing, ok := metadata["loopState"].(map[string]any); ok {
		states = existing
	}

	states[rootEventID.String()] = stateMap
	metadata["loopState"] = states
	return nil
}

func clearLoopState(metadata map[string]any, rootEventID uuid.UUID) {
	states, ok := metadata["loopState"].(map[string]any)
	if !ok {
		return
	}

	delete(states, rootEventID.String())
	if len(states) == 0 {
		delete(metadata, "loopState")
		return
	}

	metadata["loopState"] = states
}

func parseIntegerValue(value any) (int, error) {
	switch v := value.(type) {
	case nil:
		return 0, fmt.Errorf("value is required")
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed, nil
		}
		return 0, fmt.Errorf("value is not a valid integer: %s", v)
	default:
		return 0, fmt.Errorf("value must be an integer, got %T", value)
	}
}

func toAnySlice(value any) ([]any, error) {
	if value == nil {
		return []any{}, nil
	}

	val := reflect.ValueOf(value)
	if val.Kind() != reflect.Slice && val.Kind() != reflect.Array {
		return nil, fmt.Errorf("items expression must evaluate to a list, got %T", value)
	}

	result := make([]any, val.Len())
	for i := 0; i < val.Len(); i++ {
		result[i] = val.Index(i).Interface()
	}

	return result, nil
}

func (l *Loop) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (l *Loop) Actions() []core.Action {
	return []core.Action{}
}

func (l *Loop) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("loop does not support actions")
}

func (l *Loop) Setup(ctx core.SetupContext) error {
	return nil
}

func (l *Loop) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (l *Loop) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
