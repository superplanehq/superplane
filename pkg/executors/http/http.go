package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/manifest"
)

const MaxHTTPResponseSize = 8 * 1024

type HTTPExecutor struct {
	httpClient *http.Client
}

func NewHTTPExecutor(httpClient *http.Client) executors.Executor {
	return &HTTPExecutor{httpClient: httpClient}
}

type HTTPSpec struct {
	URL            string              `json:"url"`
	Payload        map[string]string   `json:"payload"`
	Headers        map[string]string   `json:"headers"`
	ResponsePolicy *HTTPResponsePolicy `json:"responsePolicy"`
}

type HTTPResponsePolicy struct {
	StatusCodes []uint32 `json:"statusCodes"`
}

func (e *HTTPExecutor) Validate(ctx context.Context, specData []byte) error {
	var spec HTTPSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	if spec.URL == "" {
		return fmt.Errorf("invalid HTTP spec: missing URL")
	}

	if spec.ResponsePolicy != nil && len(spec.ResponsePolicy.StatusCodes) > 0 {
		for _, code := range spec.ResponsePolicy.StatusCodes {
			if code < http.StatusOK || code > http.StatusNetworkAuthenticationRequired {
				return fmt.Errorf("invalid HTTP spec: invalid status code: %d", code)
			}
		}
	}

	return nil
}

func (e *HTTPExecutor) Execute(specData []byte, parameters executors.ExecutionParameters) (executors.Response, error) {
	var spec HTTPSpec
	err := json.Unmarshal(specData, &spec)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling spec data: %v", err)
	}

	payload, err := e.buildPayload(spec, parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, spec.URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := e.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	reader := io.LimitReader(res.Body, MaxHTTPResponseSize)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var allowedCodes []uint32
	if spec.ResponsePolicy != nil && spec.ResponsePolicy.StatusCodes != nil {
		allowedCodes = spec.ResponsePolicy.StatusCodes
	}

	if !slices.Contains(allowedCodes, uint32(res.StatusCode)) {
		return nil, fmt.Errorf("invalid HTTP response: status code %d not in allowed codes %v", res.StatusCode, allowedCodes)
	}

	return &HTTPResponse{
		statusCode:   res.StatusCode,
		allowedCodes: allowedCodes,
		body:         body,
	}, nil
}

func (e *HTTPExecutor) buildPayload(spec HTTPSpec, parameters executors.ExecutionParameters) (map[string]string, error) {
	payload := map[string]string{
		"stageId":     parameters.StageID,
		"executionId": parameters.ExecutionID,
	}

	for key, value := range spec.Payload {
		payload[key] = value
	}

	return payload, nil
}

type HTTPResponse struct {
	statusCode   int
	body         []byte
	allowedCodes []uint32
}

func (r *HTTPResponse) Successful() bool {
	return slices.Contains(r.allowedCodes, uint32(r.statusCode))
}

func (r *HTTPResponse) Outputs() map[string]any {
	var response map[string]any
	err := json.Unmarshal(r.body, &response)
	if err != nil {
		return map[string]any{}
	}

	if v, ok := response["outputs"]; ok {
		if outputs, ok := v.(map[string]any); ok {
			return outputs
		}
	}

	return nil
}

func (e *HTTPExecutor) Manifest() *manifest.TypeManifest {
	return &manifest.TypeManifest{
		Type:        "http",
		DisplayName: "HTTP",
		Description: "Execute HTTP POST requests to external APIs",
		Category:    "executor",
		Icon:        "http",
		Fields: []manifest.FieldManifest{
			{
				Name:        "url",
				DisplayName: "URL",
				Type:        manifest.FieldTypeString,
				Required:    true,
				Description: "The HTTP endpoint URL to send the POST request to",
				Placeholder: "https://api.example.com/webhook",
			},
			{
				Name:        "payload",
				DisplayName: "Payload",
				Type:        manifest.FieldTypeMap,
				Required:    false,
				Description: "Key-value pairs to include in the request body (in addition to stageId and executionId)",
				Placeholder: "Add custom payload fields",
			},
			{
				Name:        "headers",
				DisplayName: "Headers",
				Type:        manifest.FieldTypeMap,
				Required:    false,
				Description: "Custom HTTP headers to include in the request",
				Placeholder: "Add custom headers",
			},
			{
				Name:        "responsePolicy",
				DisplayName: "Response Policy",
				Type:        manifest.FieldTypeObject,
				Required:    false,
				Description: "Define which HTTP status codes should be considered successful",
				Fields: []manifest.FieldManifest{
					{
						Name:        "statusCodes",
						DisplayName: "Status Codes",
						Type:        manifest.FieldTypeArray,
						ItemType:    manifest.FieldTypeNumber,
						Required:    false,
						Description: "List of HTTP status codes that indicate success",
						Placeholder: "e.g., 200, 201, 204",
					},
				},
			},
		},
	}
}
