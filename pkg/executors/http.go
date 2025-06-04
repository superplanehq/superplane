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
	statusCodes []uint32
	res         *http.Response
}

func (s *HTTPResponse) Finished() bool {
	return true
}

func (s *HTTPResponse) Successful() bool {
	return slices.Contains(s.statusCodes, uint32(s.res.StatusCode))
}

func (s *HTTPResponse) Id() string {
	return ""
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

// TODO: lots of duplication in this function since all these functions,
// regardless of executor type, will do is iterate over all "resolvable" fields,
// and resolve them. If we had a function that worked on map[string]any,
// we can remove this from the interface, and then Execute()
// can just convert the map[string]any back into the structure it is expecting.
func (e *HTTPExecutor) BuildSpec(spec models.ExecutorSpec, inputs map[string]any, secrets map[string]string) (*models.ExecutorSpec, error) {
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
			ResponsePolicy: &models.HTTPResponsePolicy{
				StatusCodes: spec.HTTP.ResponsePolicy.StatusCodes,
			},
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

	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return &HTTPResponse{res: res, statusCodes: spec.HTTP.ResponsePolicy.StatusCodes}, nil
}

func (e *HTTPExecutor) Check(spec models.ExecutorSpec, id string) (Response, error) {
	return nil, nil
}

func (e *HTTPExecutor) buildPayload(spec *models.HTTPExecutorSpec) (map[string]string, error) {
	payload := map[string]string{
		"stageId":          e.execution.StageID.String(),
		"stageExecutionId": e.execution.ID.String(),
	}

	// TODO: not sure we need this
	// We could somehow define outputs from the response body
	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	payload["stageExecutionToken"] = token

	for key, value := range spec.Payload {
		payload[key] = value
	}

	return payload, nil
}
