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
)

const MaxHTTPResponseSize = 8 * 1024

type HTTPExecutor struct{}

func NewHTTPExecutor() executors.Executor {
	return &HTTPExecutor{}
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

func (e *HTTPExecutor) HandleWebhook(data []byte) (executors.Response, error) {
	return nil, nil
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
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	reader := io.LimitReader(res.Body, MaxHTTPResponseSize)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return &HTTPResponse{
		statusCode:   res.StatusCode,
		allowedCodes: spec.ResponsePolicy.StatusCodes,
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

func (r *HTTPResponse) Finished() bool {
	return true
}

func (r *HTTPResponse) Successful() bool {
	return slices.Contains(r.allowedCodes, uint32(r.statusCode))
}

func (r *HTTPResponse) Id() string {
	return ""
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
