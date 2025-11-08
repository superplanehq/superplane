package merge

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("merge", &Merge{})
}

// Merge is a component that passes its input downstream on
// the default channel. The queue/worker layer is responsible
// for aggregating inputs from multiple parents.
type Merge struct{}

func (m *Merge) Name() string        { return "merge" }
func (m *Merge) Label() string       { return "Merge" }
func (m *Merge) Description() string { return "Merge multiple upstream inputs and forward" }
func (m *Merge) Icon() string        { return "arrow-right-from-line" }
func (m *Merge) Color() string       { return "gray" }

func (m *Merge) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (m *Merge) Configuration() []configuration.Field {
	return []configuration.Field{}
}

func (m *Merge) Actions() []components.Action { return []components.Action{} }

func (m *Merge) Setup(ctx components.SetupContext) error {
	return nil
}

func (m *Merge) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("merge does not support actions")
}

func (m *Merge) Execute(ctx components.ExecutionContext) error {
	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {ctx.Data},
	})
}

func (m *Merge) ProcessQueueItem(ctx components.ProcessQueueContext) error {
	merge_group := ctx.RootEventID

	fmt.Println("processing merge for group:", merge_group)

	execID, err := m.findOrCreateExecution(ctx, merge_group)
	if err != nil {
		fmt.Println("error finding or creating execution:", err)
		return err
	}

	fmt.Println("using execution ID:", execID)

	if err := ctx.DequeueItem(); err != nil {
		return err
	}

	incoming, err := ctx.CountIncomingEdges()
	if err != nil {
		return err
	}

	md, err := m.addEventToMetadata(ctx, execID)
	if err != nil {
		return err
	}

	fmt.Println("collected:", md.EventIDs)
	fmt.Println("incoming:", incoming)

	if len(md.EventIDs) >= incoming {
		return m.FinishExecution(ctx, execID, merge_group, md)
	}

	return nil
}

func (m *Merge) findOrCreateExecution(ctx components.ProcessQueueContext, mergeGroup string) (uuid.UUID, error) {
	execID, found, err := ctx.FindExecutionIDByKV("merge_group", mergeGroup)
	if err != nil {
		return uuid.Nil, err
	}

	if found {
		return execID, nil
	}

	execID, err = ctx.CreateExecution()
	if err != nil {
		return uuid.Nil, err
	}

	err = ctx.SetExecutionKV(execID, "merge_group", mergeGroup)
	if err != nil {
		return uuid.Nil, err
	}

	md := &ExecutionMetadata{
		GroupKey: mergeGroup,
		EventIDs: []string{},
	}

	err = ctx.SetExecutionMetadata(execID, md)
	if err != nil {
		return uuid.Nil, err
	}

	return execID, nil
}

func (m *Merge) addEventToMetadata(ctx components.ProcessQueueContext, execID uuid.UUID) (*ExecutionMetadata, error) {
	md := &ExecutionMetadata{}

	rawMeta, err := ctx.GetExecutionMetadata(execID)
	if err != nil {
		return nil, err
	}

	err = mapstructure.Decode(rawMeta, md)
	if err != nil {
		return nil, err
	}

	md.EventIDs = append(md.EventIDs, ctx.EventID)

	err = ctx.SetExecutionMetadata(execID, md)
	if err != nil {
		return nil, err
	}

	return md, nil
}

func (m *Merge) FinishExecution(ctx components.ProcessQueueContext, execID uuid.UUID, merge_group string, md *ExecutionMetadata) error {
	output := map[string]any{
		"merge_group":     merge_group,
		"event_ids_count": md.EventIDs,
	}

	return ctx.FinishExecution(execID, map[string][]any{
		components.DefaultOutputChannel.Name: {output},
	})
}
