package extensions

const (
	InvocationBlockTypeComponent = "components"

	InvocationOperationSetup   = "setup"
	InvocationOperationExecute = "execute"
	InvocationOperationCancel  = "cancel"
)

type InvocationTarget struct {
	BlockType string `json:"blockType"`
	BlockName string `json:"blockName"`
	Operation string `json:"operation"`
}

type InvocationIntegrationContext struct {
	ID            string         `json:"id,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	Metadata      any            `json:"metadata,omitempty"`
}

type InvocationContext struct {
	Configuration any                           `json:"configuration,omitempty"`
	Integration   *InvocationIntegrationContext `json:"integration,omitempty"`
	Metadata      any                           `json:"metadata,omitempty"`
}

type NormalizedInvocationContext struct {
	Configuration any                          `json:"configuration"`
	Integration   InvocationIntegrationContext `json:"integration"`
	Metadata      any                          `json:"metadata"`
}

type SetupInvocation struct{}

type ExecuteInvocation struct {
	Data any `json:"data,omitempty"`
}

type CancelInvocation struct {
	Data any `json:"data,omitempty"`
}

type SetupInvocationPayload struct {
	Target     InvocationTarget   `json:"target"`
	Context    *InvocationContext `json:"context,omitempty"`
	Invocation *SetupInvocation   `json:"invocation,omitempty"`
}

type ExecuteInvocationPayload struct {
	Target     InvocationTarget   `json:"target"`
	Context    *InvocationContext `json:"context,omitempty"`
	Invocation *ExecuteInvocation `json:"invocation,omitempty"`
}

type CancelInvocationPayload struct {
	Target     InvocationTarget   `json:"target"`
	Context    *InvocationContext `json:"context,omitempty"`
	Invocation *CancelInvocation  `json:"invocation,omitempty"`
}

type SetupInvocationEnvelope struct {
	Target     InvocationTarget            `json:"target"`
	Context    NormalizedInvocationContext `json:"context"`
	Invocation SetupInvocation             `json:"invocation"`
}

type ExecuteInvocationEnvelope struct {
	Target     InvocationTarget            `json:"target"`
	Context    NormalizedInvocationContext `json:"context"`
	Invocation ExecuteInvocation           `json:"invocation"`
}

type CancelInvocationEnvelope struct {
	Target     InvocationTarget            `json:"target"`
	Context    NormalizedInvocationContext `json:"context"`
	Invocation CancelInvocation            `json:"invocation"`
}

type InvocationScheduledAction struct {
	ActionName string         `json:"actionName"`
	Parameters map[string]any `json:"parameters"`
	IntervalMs int64          `json:"intervalMs"`
}

type InvocationEvent struct {
	PayloadType string `json:"payloadType"`
	Payload     any    `json:"payload"`
}

type InvocationExecutionEmission struct {
	Channel     string `json:"channel"`
	PayloadType string `json:"payloadType"`
	Payloads    []any  `json:"payloads"`
}

type InvocationExecutionStateEffects struct {
	Finished  bool                          `json:"finished"`
	Passed    bool                          `json:"passed"`
	Failed    *InvocationExecutionFailure   `json:"failed"`
	KV        map[string]string             `json:"kv"`
	Emissions []InvocationExecutionEmission `json:"emissions"`
}

type InvocationExecutionFailure struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type InvocationIntegrationSecret struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type InvocationIntegrationSubscription struct {
	ID            string `json:"id"`
	Configuration any    `json:"configuration"`
	Messages      []any  `json:"messages"`
}

type InvocationIntegrationScheduledAction struct {
	ActionName string `json:"actionName"`
	Parameters any    `json:"parameters"`
	IntervalMs int64  `json:"intervalMs"`
}

type InvocationIntegrationEffects struct {
	ID                        string                                 `json:"id"`
	Ready                     bool                                   `json:"ready"`
	Error                     string                                 `json:"error,omitempty"`
	Metadata                  any                                    `json:"metadata"`
	BrowserAction             any                                    `json:"browserAction"`
	RequestedWebhooks         []any                                  `json:"requestedWebhooks"`
	ScheduledResyncIntervalMs *int64                                 `json:"scheduledResyncIntervalMs"`
	ScheduledActions          []InvocationIntegrationScheduledAction `json:"scheduledActions"`
	Secrets                   []InvocationIntegrationSecret          `json:"secrets"`
	Subscriptions             []InvocationIntegrationSubscription    `json:"subscriptions"`
}

type InvocationWebhookEffects struct {
	URL     string `json:"url"`
	BaseURL string `json:"baseURL"`
	Secret  any    `json:"secret"`
}

type InvocationEffects struct {
	Metadata any `json:"metadata"`
	Requests struct {
		ScheduledActions []InvocationScheduledAction `json:"scheduledActions"`
	} `json:"requests"`
	Events         []InvocationEvent               `json:"events"`
	ExecutionState InvocationExecutionStateEffects `json:"executionState"`
	Integration    InvocationIntegrationEffects    `json:"integration"`
	Webhook        InvocationWebhookEffects        `json:"webhook"`
}

//
// Output messages
//

type SetupInvocationOutput struct {
	Target  InvocationTarget  `json:"target"`
	Effects InvocationEffects `json:"effects"`
}

type ExecuteInvocationOutput struct {
	Target  InvocationTarget  `json:"target"`
	Effects InvocationEffects `json:"effects"`
}

type CancelInvocationOutput struct {
	Target  InvocationTarget  `json:"target"`
	Effects InvocationEffects `json:"effects"`
}
