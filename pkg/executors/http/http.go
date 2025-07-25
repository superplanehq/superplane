package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
)

const MaxHTTPResponseSize = 8 * 1024

type HTTPExecutor struct{}

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

func init() {
	executors.Register(models.ExecutorSpecTypeHTTP, NewHTTPExecutor)
}

func NewHTTPExecutor(_ integrations.Integration, _ integrations.Resource) (executors.Executor, error) {
	return &HTTPExecutor{}, nil
}

func (e *HTTPExecutor) Name() string {
	return models.ExecutorSpecTypeHTTP
}

func (e *HTTPExecutor) HandleWebhook(data []byte) (executors.Response, error) {
	return nil, nil
}

func (e *HTTPExecutor) Execute(spec models.ExecutorSpec, parameters executors.ExecutionParameters) (executors.Response, error) {
	payload, err := e.buildPayload(spec.HTTP, parameters)
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, spec.HTTP.URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for k, v := range spec.HTTP.Headers {
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
		allowedCodes: spec.HTTP.ResponsePolicy.StatusCodes,
		body:         body,
	}, nil
}

func (e *HTTPExecutor) Check(id string) (executors.Response, error) {
	return nil, nil
}

func (e *HTTPExecutor) buildPayload(spec *models.HTTPExecutorSpec, parameters executors.ExecutionParameters) (map[string]string, error) {
	payload := map[string]string{
		"stageId":     parameters.StageID,
		"executionId": parameters.ExecutionID,
	}

	for key, value := range spec.Payload {
		payload[key] = value
	}

	return payload, nil
}
