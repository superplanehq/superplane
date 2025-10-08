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

type Spec struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

type HTTP struct{}

func (e *HTTP) Name() string {
	return "http"
}

func (e *HTTP) Description() string {
	return "Send HTTP request. The HTTP response is sent to the default output branch"
}

func (e *HTTP) Outputs(configuration any) []string {
	return []string{components.DefaultBranchName}
}

func (e *HTTP) Configuration() []components.ConfigurationField {
	return []components.ConfigurationField{
		{
			Name:        "url",
			Type:        "string",
			Description: "URL to send the HTTP request to",
			Required:    true,
		},
		{
			Name:        "method",
			Type:        "string",
			Description: "HTTP method (GET, POST, PUT, DELETE, etc.)",
			Required:    true,
		},
		{
			Name:        "headers",
			Type:        "map",
			Description: "HTTP headers to include in the request",
			Required:    false,
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
	for key, value := range spec.Headers {
		req.Header.Set(key, value)
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

	return ctx.State.Finish(map[string][]any{
		components.DefaultBranchName: {response},
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
