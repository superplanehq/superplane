package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/components"
)

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Spec struct {
	URL     string   `json:"url"`
	Method  string   `json:"method"`
	Headers []Header `json:"headers"`
}

type HTTP struct{}

func (e *HTTP) Name() string {
	return "http"
}

func (e *HTTP) Label() string {
	return "HTTP"
}

func (e *HTTP) Description() string {
	return "Make HTTP requests"
}

func (e *HTTP) OutputBranches(configuration any) []components.OutputBranch {
	return []components.OutputBranch{components.DefaultOutputBranch}
}

func (e *HTTP) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:     "url",
			Label:    "URL",
			Type:     components.FieldTypeURL,
			Required: true,
		},
		{
			Name:     "method",
			Type:     components.FieldTypeSelect,
			Label:    "HTTP method",
			Required: true,
			Default:  "POST",
			Options: []components.FieldOption{
				{Label: "GET", Value: "GET"},
				{Label: "POST", Value: "POST"},
				{Label: "PUT", Value: "PUT"},
				{Label: "DELETE", Value: "DELETE"},
				{Label: "PATCH", Value: "PATCH"},
			},
		},
		{
			Name:     "headers",
			Label:    "Headers",
			Type:     components.FieldTypeList,
			Required: false,
			ListItem: &components.ListItemDefinition{
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

	body, err := e.getBody(ctx, spec)
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
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

	return ctx.ExecutionStateContext.Finish(map[string][]any{
		components.DefaultOutputBranch.Name: {response},
	})
}

func (e *HTTP) getBody(ctx components.ExecutionContext, spec Spec) (io.Reader, error) {
	if spec.Method == http.MethodGet {
		return nil, nil
	}

	bodyData, err := json.Marshal(ctx.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return bytes.NewReader(bodyData), nil
}
