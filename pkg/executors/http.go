package executors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type HTTPExecutor struct {
	execution models.StageExecution
	jwtSigner *jwt.Signer
}

type HTTPResponse struct {
	statuses []int
	res      *http.Response
}

func (s *HTTPResponse) Finished() bool {
	return true
}

func (s *HTTPResponse) Successful() bool {
	return slices.Contains(s.statuses, s.res.StatusCode)
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

func (e *HTTPExecutor) BuildSpec(spec models.ExecutorSpec, inputs map[string]any, secrets map[string]string) (*models.ExecutorSpec, error) {
	if spec.Type != e.Name() {
		return nil, fmt.Errorf("wrong spec type")
	}

	URL, err := resolveExpression(spec.HTTP.URL, inputs, secrets)
	if err != nil {
		return nil, err
	}

	payload := make(map[string]string, len(spec.HTTP.Payload))
	for k, v := range spec.HTTP.Payload {
		value, err := resolveExpression(v, inputs, secrets)
		if err != nil {
			return nil, err
		}

		payload[k] = value.(string)
	}

	headers := make(map[string]string, len(spec.HTTP.Headers))
	for k, v := range spec.HTTP.Headers {
		value, err := resolveExpression(v, inputs, secrets)
		if err != nil {
			return nil, err
		}

		headers[k] = value.(string)
	}

	return &models.ExecutorSpec{
		Type: spec.Type,
		HTTP: &models.HTTPExecutorSpec{
			URL:     URL.(string),
			Payload: payload,
			Headers: headers,
		},
	}, nil
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

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return &HTTPResponse{res: res, statuses: spec.HTTP.SuccessPolicy.Statuses}, nil
}

func (e *HTTPExecutor) buildPayload(spec *models.HTTPExecutorSpec) (map[string]string, error) {
	payload := map[string]string{
		"SEMAPHORE_STAGE_ID":           e.execution.StageID.String(),
		"SEMAPHORE_STAGE_EXECUTION_ID": e.execution.ID.String(),
	}

	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	payload["SEMAPHORE_STAGE_EXECUTION_TOKEN"] = token
	for key, value := range spec.Payload {
		payload[key] = value
	}

	return payload, nil
}
