package executors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

const MaxHTTPResponseSize = 8 * 1024

type HTTPExecutor struct {
	execution models.StageExecution
	jwtSigner *jwt.Signer
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

func NewHTTPExecutor(execution models.StageExecution, jwtSigner *jwt.Signer) (*HTTPExecutor, error) {
	return &HTTPExecutor{
		execution: execution,
		jwtSigner: jwtSigner,
	}, nil
}

func (e *HTTPExecutor) Name() string {
	return models.ExecutorSpecTypeHTTP
}

func (e *HTTPExecutor) Execute(spec models.ExecutorSpec) (Response, error) {
	payload, err := e.buildPayload(spec.HTTP)
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

func (e *HTTPExecutor) Check(spec models.ExecutorSpec, id string) (Response, error) {
	return nil, nil
}

func (e *HTTPExecutor) buildPayload(spec *models.HTTPExecutorSpec) (map[string]string, error) {
	payload := map[string]string{
		"stageId":     e.execution.StageID.String(),
		"executionId": e.execution.ID.String(),
	}

	for key, value := range spec.Payload {
		payload[key] = value
	}

	return payload, nil
}
