package extensions

type InvocationTarget struct {
	BlockType string `json:"blockType"`
	BlockName string `json:"blockName"`
	Operation string `json:"operation"`
}

type InvocationPayload struct {
	Target        InvocationTarget       `json:"target"`
	Configuration any                    `json:"configuration,omitempty"`
	Input         any                    `json:"input,omitempty"`
	Current       any                    `json:"current,omitempty"`
	Requested     any                    `json:"requested,omitempty"`
	Parameters    map[string]any         `json:"parameters,omitempty"`
	ActionName    string                 `json:"actionName,omitempty"`
	Headers       map[string][]string    `json:"headers,omitempty"`
	Body          any                    `json:"body,omitempty"`
	Message       any                    `json:"message,omitempty"`
	Integration   *InvocationIntegration `json:"integration,omitempty"`
	Webhook       *InvocationWebhook     `json:"webhook,omitempty"`
	Metadata      any                    `json:"metadata,omitempty"`
}

type InvocationIntegration struct {
	ID            string         `json:"id,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	Metadata      any            `json:"metadata,omitempty"`
}

type InvocationWebhook struct {
	ID            string `json:"id,omitempty"`
	URL           string `json:"url,omitempty"`
	Secret        any    `json:"secret,omitempty"`
	Metadata      any    `json:"metadata,omitempty"`
	Configuration any    `json:"configuration,omitempty"`
}
