package connectiongroup

import (
	"fmt"
	"time"

	"github.com/expr-lang/expr"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/crypto"
)

type GroupBy struct{}

type Spec struct {
	Fields          []Field
	Timeout         string
	TimeoutBehavior string
}

type Field struct {
	Name       string `json:"name"`
	Expression string `json:"expression"`
}

type Metadata struct {
	FieldSet   map[string]string `json:"field_set"`
	WaitingFor []string          `json:"waiting_for"`
}

func (m *Metadata) removeSourceFromWaitingList(source string) {
	newWaitingFor := []string{}

	for _, s := range m.WaitingFor {
		if s != source {
			newWaitingFor = append(newWaitingFor, s)
		}
	}

	m.WaitingFor = newWaitingFor
}

func (c *GroupBy) Name() string {
	return "group-by"
}

func (c *GroupBy) Label() string {
	return "Group By"
}

func (c *GroupBy) Description() string {
	return "Group events by fields"
}

func (c *GroupBy) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (c *GroupBy) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "fields",
			Label:    "Fields",
			Type:     components.FieldTypeList,
			Required: true,
			ListItem: &components.ListItemDefinition{
				Type: components.FieldTypeObject,
				Schema: []components.ConfigurationField{
					{
						Name:     "name",
						Label:    "Field Name",
						Type:     components.FieldTypeString,
						Required: true,
					},
					{
						Name:        "expression",
						Label:       "Field Expression",
						Type:        components.FieldTypeString,
						Description: "Expression to evaluate and fetch field value",
						Required:    true,
					},
				},
			},
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Description: "Number of seconds to wait for all connections to emit events with the same fields",
			Type:        components.FieldTypeNumber,
			Min:         intPtr(60),
			Max:         intPtr(86400),
		},
		{
			Name:        "timeoutBehavior",
			Label:       "Timeout Behavior",
			Description: "What to do when the timeout has been reached and missing connections still exist",
			Type:        components.FieldTypeSelect,
			Options: []components.FieldOption{
				{Label: "Emit", Value: "emit"},
				{Label: "Drop", Value: "drop"},
			},
		},
	}
}

func (c *GroupBy) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	waitingFor, err := initialWaitingFor(ctx.WorkflowContext)
	if err != nil {
		return err
	}

	fieldSet, err := calculateFieldSet(spec, ctx.Data)
	if err != nil {
		return err
	}

	metadata := Metadata{
		FieldSet:   fieldSet,
		WaitingFor: waitingFor,
	}

	//
	// Otherwise, fetch items in the queue until completion,
	// or schedule additional work if not possible.
	//
	for len(metadata.WaitingFor) > 0 {
		nextQueueItem, err := ctx.WorkflowContext.Dequeue()
		if err != nil {
			return fmt.Errorf("error fetching next queue item: %v", err)
		}

		//
		// If no items are left in the queue,
		// we break this loop, so we can schedule things
		// with the processing engine.
		//
		if nextQueueItem == nil {
			break
		}

		//
		// Calculate field set for the new event
		//
		f, err := calculateFieldSet(spec, nextQueueItem.Data)
		if err != nil {
			return fmt.Errorf("error calculating field for event from %s: %v", nextQueueItem.Source.ID, err)
		}

		//
		// If the field set is not equal to the current execution's field set,
		// just ignore it.
		//
		if !isEqual(f, fieldSet) {
			continue
		}

		metadata.removeSourceFromWaitingList(nextQueueItem.Source.ID)
	}

	//
	// Everything received, pass the execution.
	//
	if len(metadata.WaitingFor) == 0 {
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {fieldSet},
		})
	}

	//
	// We are still waiting for events from some nodes.
	// Save the metadata and schedule queue check.
	//
	ctx.MetadataContext.Set(metadata)
	return ctx.RequestContext.SubscribeTo("new-queue-item", "onQueueItem")
}

func (c *GroupBy) Actions() []components.Action {
	return []components.Action{
		{
			Name:         "onQueueItem",
			Description:  "Receive new queue item from processing engine",
			Parameters:   []components.ConfigurationField{},
			IsUserAction: false,
		},
	}
}

func (c *GroupBy) HandleAction(ctx components.ActionContext) error {
	switch ctx.Name {
	case "onQueueItem":
		return c.handleNewQueueItem(ctx)

	default:
		return fmt.Errorf("action %s not supported", ctx.Name)
	}
}

func (c *GroupBy) handleNewQueueItem(ctx components.ActionContext) error {
	//
	// Decode spec and metadata
	//
	spec := Spec{}
	err := mapstructure.Decode(ctx.NodeConfiguration, &spec)
	if err != nil {
		return err
	}

	//
	// Add new approval to metadata
	//
	var metadata Metadata
	err = mapstructure.Decode(ctx.MetadataContext.Get(), &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	//
	// Calculate and handle field set for queue item.
	//
	f, err := calculateFieldSet(spec, ctx.Data)
	if err != nil {
		return fmt.Errorf("error calculating field: %v", err)
	}

	if !isEqual(f, metadata.FieldSet) {
		return nil
	}

	sourceNode, err := ctx.WorkflowContext.SourceNode()
	if err != nil {
		return err
	}

	metadata.removeSourceFromWaitingList(sourceNode.ID)

	//
	// Nothing more to wait, complete the execution.
	//
	if len(metadata.WaitingFor) == 0 {
		return ctx.ExecutionStateContext.Pass(map[string][]any{
			components.DefaultOutputChannel.Name: {metadata.FieldSet},
		})
	}

	//
	// Still didn't receive everything we need.
	//
	return nil
}

// TODO: this could probably be passed through execution context too,
// so components don't have to rewrite this for themselves every time.
func evaluateFieldExpression(expression string, data any) (string, error) {
	env := map[string]any{"$": data}

	vm, err := expr.Compile(expression, []expr.Option{
		expr.Env(env),
		expr.AsAny(),
		expr.WithContext("ctx"),
		expr.Timezone(time.UTC.String()),
	}...)

	if err != nil {
		return "", fmt.Errorf("error compiling expression: %v", err)
	}

	output, err := expr.Run(vm, env)
	if err != nil {
		return "", fmt.Errorf("expression evaluation failed: %w", err)
	}

	v, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("expression must evaluate to string, got %T", output)
	}

	return v, nil
}

func intPtr(v int) *int {
	return &v
}

func initialWaitingFor(ctx components.WorkflowContext) ([]string, error) {
	source, err := ctx.SourceNode()
	if err != nil {
		return nil, err
	}

	previousNodes, err := ctx.PreviousNodes()
	if err != nil {
		return nil, err
	}

	waitingFor := []string{}
	for _, previousNode := range previousNodes {
		if previousNode.ID != source.ID {
			waitingFor = append(waitingFor, previousNode.ID)
		}
	}

	return waitingFor, nil
}

func calculateFieldSet(spec Spec, data any) (map[string]string, error) {
	fields := make(map[string]string, len(spec.Fields))

	for _, field := range spec.Fields {
		fieldValue, err := evaluateFieldExpression(field.Expression, data)
		if err != nil {
			return nil, fmt.Errorf("error evaluating field expression (%s) for %s: %v", field.Expression, field.Name, err)
		}

		fields[field.Name] = fieldValue
	}

	return fields, nil
}

func isEqual(a map[string]string, b map[string]string) bool {
	aHash, _ := crypto.SHA256ForMap(a)
	bHash, _ := crypto.SHA256ForMap(b)
	return aHash == bHash
}
