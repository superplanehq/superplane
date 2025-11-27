package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
)

func init() {
	registry.RegisterComponent("http", &HTTP{})
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Spec struct {
	URL     string   `json:"url"`
	Method  string   `json:"method"`
	Payload any      `json:"payload"`
	Headers []Header `json:"headers"`
}

type HTTP struct{}

func (e *HTTP) Name() string {
	return "http"
}

func (e *HTTP) Label() string {
	return "HTTP Request"
}

func (e *HTTP) Description() string {
	return "Make HTTP requests"
}

func (e *HTTP) Icon() string {
	return "globe"
}

func (e *HTTP) Color() string {
	return "blue"
}

func (e *HTTP) Setup(ctx components.SetupContext) error {
	return nil
}

func (e *HTTP) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (e *HTTP) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "url",
			Label:    "URL",
			Type:     configuration.FieldTypeString,
			Required: true,
		},
		{
			Name:     "method",
			Type:     configuration.FieldTypeSelect,
			Label:    "Method",
			Required: true,
			Default:  "POST",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "GET", Value: "GET"},
						{Label: "POST", Value: "POST"},
						{Label: "PUT", Value: "PUT"},
						{Label: "DELETE", Value: "DELETE"},
						{Label: "PATCH", Value: "PATCH"},
					},
				},
			},
		},
		{
			Name:        "payload",
			Type:        configuration.FieldTypeObject,
			Label:       "Payload",
			Description: "JSON payload to send. String values support dynamic interpolation using expr language (e.g., $.ref, $.data.status)",
			Required:    false,
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "method", Values: []string{"POST", "PUT", "PATCH"}},
			},
		},
		{
			Name:        "headers",
			Label:       "Headers",
			Type:        configuration.FieldTypeList,
			Description: "HTTP headers. Both name and value support dynamic interpolation (e.g., $.ref)",
			Required:    false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "name",
								Type:        configuration.FieldTypeString,
								Label:       "Header Name",
								Description: "Supports dynamic interpolation (e.g., $.headerName)",
								Required:    true,
							},
							{
								Name:        "value",
								Type:        configuration.FieldTypeString,
								Label:       "Header value",
								Description: "Supports dynamic interpolation (e.g., $.headerValue)",
								Required:    true,
							},
						},
					},
				},
			},
		},
	}
}

func (e *HTTP) Actions() []components.Action {
	return []components.Action{}
}

func (e *HTTP) HandleAction(ctx components.ActionContext) error {
	return fmt.Errorf("http does not support actions")
}

func (e *HTTP) ProcessQueueItem(ctx components.ProcessQueueContext) (*models.WorkflowNodeExecution, error) {
	return ctx.DefaultProcessing()
}

func (e *HTTP) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	// Process payload with dynamic data interpolation
	processedPayload, err := ProcessPayload(spec.Payload, ctx.Data)
	if err != nil {
		return fmt.Errorf("failed to process payload: %w", err)
	}

	// Process headers with dynamic data interpolation
	processedHeaders, err := ProcessHeaders(spec.Headers, ctx.Data)
	if err != nil {
		return fmt.Errorf("failed to process headers: %w", err)
	}

	body, err := e.getBody(spec.Method, processedPayload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(spec.Method, spec.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for _, header := range processedHeaders {
		req.Header.Set(header.Name, header.Value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var bodyData any
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &bodyData)
		if err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	// Build response with status, headers, and body
	response := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Header,
		"body":    bodyData,
	}

	return ctx.ExecutionStateContext.Pass(map[string][]any{
		components.DefaultOutputChannel.Name: {response},
	})
}

func (e *HTTP) getBody(method string, payload any) (io.Reader, error) {
	if method == http.MethodGet || payload == nil {
		return nil, nil
	}

	bodyData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return bytes.NewReader(bodyData), nil
}
