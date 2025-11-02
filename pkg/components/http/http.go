package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
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

func (e *HTTP) OutputChannels(configuration any) []components.OutputChannel {
	return []components.OutputChannel{components.DefaultOutputChannel}
}

func (e *HTTP) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "url",
			Label:    "URL",
			Type:     components.FieldTypeString,
			Required: true,
		},
		{
			Name:     "method",
			Type:     components.FieldTypeSelect,
			Label:    "Method",
			Required: true,
			Default:  "POST",
			TypeOptions: &components.TypeOptions{
				Select: &components.SelectTypeOptions{
					Options: []components.FieldOption{
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
			Name:     "payload",
			Type:     components.FieldTypeObject,
			Label:    "Payload",
			Required: false,
		},
		{
			Name:     "headers",
			Label:    "Headers",
			Type:     components.FieldTypeList,
			Required: false,
			TypeOptions: &components.TypeOptions{
				List: &components.ListTypeOptions{
					ItemDefinition: &components.ListItemDefinition{
						Type: components.FieldTypeObject,
						Schema: []components.ConfigurationField{
							{
								Name:     "name",
								Type:     components.FieldTypeString,
								Label:    "Header Name",
								Required: true,
							},
							{
								Name:     "value",
								Type:     components.FieldTypeString,
								Label:    "Header value",
								Required: true,
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

func (e *HTTP) Execute(ctx components.ExecutionContext) error {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return err
	}

	configMap, ok := ctx.Configuration.(map[string]any)
	if !ok {
		return fmt.Errorf("failed to parse configuration")
	}
	_, payloadProvided := configMap["payload"]

	body, err := e.getBody(spec.Method, spec.Payload, payloadProvided)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(spec.Method, spec.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for _, header := range spec.Headers {
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

func (e *HTTP) getBody(method string, payload any, payloadProvided bool) (io.Reader, error) {
	if method == http.MethodGet {
		return nil, nil
	}

	if !payloadProvided {
		return nil, nil
	}

	if payload == nil {
		return bytes.NewReader([]byte("null")), nil
	}

	bodyData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return bytes.NewReader(bodyData), nil
}
