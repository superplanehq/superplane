package merge

import (
    "fmt"

    "github.com/superplanehq/superplane/pkg/components"
    "github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("merge", &Merge{})
}

// Merge waits for all upstream inputs for a given root event
// and then forwards the aggregated inputs once, on the default channel.
type Merge struct{}

func (m *Merge) Name() string        { return "merge" }
func (m *Merge) Label() string       { return "Merge" }
func (m *Merge) Description() string { return "Merge multiple upstream inputs and forward" }
func (m *Merge) Icon() string        { return "arrow-right-from-line" }
func (m *Merge) Color() string       { return "gray" }

func (m *Merge) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (m *Merge) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{}
}

func (m *Merge) Actions() []components.Action { return []components.Action{} }

func (m *Merge) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("merge does not support actions")
}

func (m *Merge) Execute(ctx components.ExecutionContext) error {
    // Pass-through: downstream expects the aggregated input already.
    return ctx.ExecutionStateContext.Pass(map[string][]any{
        components.DefaultOutputChannel.Name: {ctx.Data},
    })
}

// Generic pre-execution policy implementation for merge
func (m *Merge) WantsPreExecution(nodeConfiguration any) bool { return true }

func (m *Merge) StateKey(rootEventID string, nodeConfiguration any) string { return rootEventID }

func (m *Merge) Expected(incoming []components.IncomingEdge, nodeConfiguration any) []string {
    keys := make([]string, 0, len(incoming))
    for _, e := range incoming {
        keys = append(keys, e.SourceNodeID+"::"+e.Channel)
    }
    return keys
}

func (m *Merge) Observe(sourceNodeID string, channel string, payload any, nodeConfiguration any) (string, any) {
    return sourceNodeID + "::" + channel, payload
}

func (m *Merge) Ready(expected []string, observed map[string]any, nodeConfiguration any) bool {
    if len(expected) == 0 {
        return true
    }
    for _, k := range expected {
        if _, ok := observed[k]; !ok {
            return false
        }
    }
    return true
}

func (m *Merge) Aggregate(expected []string, observed map[string]any, nodeConfiguration any) any {
    out := make([]any, 0, len(observed))
    for _, k := range expected {
        if v, ok := observed[k]; ok {
            out = append(out, v)
        }
    }
    return out
}
