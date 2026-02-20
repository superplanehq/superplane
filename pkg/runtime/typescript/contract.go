package typescript

import "fmt"

const (
	OperationComponentExecute         = "component.execute"
	OperationComponentSetup           = "component.setup"
	OperationIntegrationSync          = "integration.sync"
	OperationIntegrationHandleAction  = "integration.handleAction"
	OperationIntegrationListResources = "integration.listResources"
	OperationIntegrationHandleRequest = "integration.handleRequest"
	OutcomePass                       = "pass"
	OutcomeFail                       = "fail"
	OutcomeNoop                       = "noop"
)

type ComponentExecutionRequest struct {
	Operation string                  `json:"operation"`
	Component string                  `json:"component"`
	Context   ComponentExecutionInput `json:"context"`
}

type ComponentExecutionInput struct {
	ExecutionID              string         `json:"executionId"`
	WorkflowID               string         `json:"workflowId"`
	OrganizationID           string         `json:"organizationId"`
	NodeID                   string         `json:"nodeId"`
	SourceNodeID             string         `json:"sourceNodeId"`
	Configuration            any            `json:"configuration"`
	IntegrationConfiguration map[string]any `json:"integrationConfiguration,omitempty"`
	Data                     any            `json:"data"`
	Metadata                 map[string]any `json:"metadata,omitempty"`
	NodeMetadata             map[string]any `json:"nodeMetadata,omitempty"`
}

type ComponentExecutionResponse struct {
	Outcome      string                 `json:"outcome"`
	ErrorReason  string                 `json:"errorReason,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Outputs      []ComponentOutput      `json:"outputs,omitempty"`
	Metadata     map[string]any         `json:"metadata,omitempty"`
	NodeMetadata map[string]any         `json:"nodeMetadata,omitempty"`
	KVs          []ComponentExecutionKV `json:"kvs,omitempty"`
}

type ComponentOutput struct {
	Channel     string `json:"channel"`
	PayloadType string `json:"payloadType"`
	Payload     any    `json:"payload"`
}

type ComponentExecutionKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type IntegrationRuntimeRequest struct {
	Operation   string                    `json:"operation"`
	Integration string                    `json:"integration"`
	Context     IntegrationRuntimeContext `json:"context"`
}

type IntegrationRuntimeContext struct {
	Configuration      any                     `json:"configuration"`
	Metadata           map[string]any          `json:"metadata,omitempty"`
	OrganizationID     string                  `json:"organizationId,omitempty"`
	BaseURL            string                  `json:"baseUrl,omitempty"`
	WebhooksBaseURL    string                  `json:"webhooksBaseUrl,omitempty"`
	ActionName         string                  `json:"actionName,omitempty"`
	ActionParameters   any                     `json:"actionParameters,omitempty"`
	ResourceType       string                  `json:"resourceType,omitempty"`
	ResourceParameters map[string]string       `json:"resourceParameters,omitempty"`
	Request            *IntegrationHTTPRequest `json:"request,omitempty"`
}

type IntegrationRuntimeResponse struct {
	Outcome          string                   `json:"outcome"`
	ErrorReason      string                   `json:"errorReason,omitempty"`
	Error            string                   `json:"error,omitempty"`
	Metadata         map[string]any           `json:"metadata,omitempty"`
	State            string                   `json:"state,omitempty"`
	StateDescription string                   `json:"stateDescription,omitempty"`
	Resources        []IntegrationResource    `json:"resources,omitempty"`
	HTTP             *IntegrationHTTPResponse `json:"http,omitempty"`
}

type IntegrationResource struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

type IntegrationHTTPRequest struct {
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Query   string              `json:"query,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`
	Body    []byte              `json:"body,omitempty"`
}

type IntegrationHTTPResponse struct {
	StatusCode int                 `json:"statusCode"`
	Headers    map[string][]string `json:"headers,omitempty"`
	Body       []byte              `json:"body,omitempty"`
}

type EmittedPayload struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Data      any    `json:"data"`
}

func (r *ComponentExecutionResponse) Validate() error {
	switch r.Outcome {
	case OutcomePass, OutcomeFail, OutcomeNoop:
		return nil
	default:
		return fmt.Errorf("invalid outcome %q", r.Outcome)
	}
}

func (r *IntegrationRuntimeResponse) Validate() error {
	switch r.Outcome {
	case OutcomePass, OutcomeFail, OutcomeNoop:
		return nil
	default:
		return fmt.Errorf("invalid outcome %q", r.Outcome)
	}
}
