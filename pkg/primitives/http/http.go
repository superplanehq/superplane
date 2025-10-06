package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/primitives"
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
	return []string{primitives.DefaultBranchName}
}

func (e *HTTP) Configuration() []primitives.ConfigurationField {
	return []primitives.ConfigurationField{
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

func (e *HTTP) Execute(ctx primitives.ExecutionContext) (*primitives.Result, error) {
	spec := Spec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return nil, err
	}

	body, err := e.getBody(ctx, spec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(spec.Method, spec.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range spec.Headers {
		req.Header.Set(key, value)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var response map[string]any
	if len(respBody) > 0 {
		err := json.Unmarshal(respBody, &response)
		if err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return &primitives.Result{
		Branches: map[string][]any{
			primitives.DefaultBranchName: {response},
		},
	}, nil
}

func (e *HTTP) getBody(ctx primitives.ExecutionContext, spec Spec) (io.Reader, error) {
	if spec.Method == http.MethodGet {
		return nil, nil
	}

	bodyData, err := json.Marshal(ctx.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	return bytes.NewReader(bodyData), nil
}
