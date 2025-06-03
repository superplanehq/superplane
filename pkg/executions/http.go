package executions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
)

type HTTPExecutor struct {
	execution models.StageExecution
	template  *models.HTTPExecutorSpec
	encryptor encryptor.Encryptor
	jwtSigner *jwt.Signer
}

type HTTPResource struct {
	statuses []int
	res      *http.Response
}

func (r *HTTPResource) Async() bool {
	return false
}

func (r *HTTPResource) AsyncId() string {
	return ""
}

func (r *HTTPResource) Check() (Status, error) {
	return &HTTPStatus{res: r.res, statuses: r.statuses}, nil
}

type HTTPStatus struct {
	statuses []int
	res      *http.Response
}

func (s *HTTPStatus) Finished() bool {
	return true
}

func (s *HTTPStatus) Successful() bool {
	return slices.Contains(s.statuses, s.res.StatusCode)
}

func NewHTTPExecutor(execution models.StageExecution, template *models.HTTPExecutorSpec, encryptor encryptor.Encryptor, jwtSigner *jwt.Signer) (*HTTPExecutor, error) {
	return &HTTPExecutor{
		execution: execution,
		template:  template,
		encryptor: encryptor,
		jwtSigner: jwtSigner,
	}, nil
}

func (e *HTTPExecutor) Execute() (Resource, error) {
	payload, err := e.buildPayload()
	if err != nil {
		return nil, fmt.Errorf("error building parameters: %v", err)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, e.template.URL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	for k, v := range e.template.Headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return &HTTPResource{res: res, statuses: e.template.SuccessPolicy.Statuses}, nil
}

func (e *HTTPExecutor) buildPayload() (map[string]string, error) {
	payload := map[string]string{
		"SEMAPHORE_STAGE_ID":           e.execution.StageID.String(),
		"SEMAPHORE_STAGE_EXECUTION_ID": e.execution.ID.String(),
	}

	token, err := e.jwtSigner.Generate(e.execution.ID.String(), 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("error generating tags token: %v", err)
	}

	payload["SEMAPHORE_STAGE_EXECUTION_TOKEN"] = token
	for key, value := range e.template.Payload {
		payload[key] = value
	}

	return payload, nil
}

// HTTP executor is sync, so no need for this
func (e *HTTPExecutor) AsyncCheck(_ string) (Status, error) {
	return nil, nil
}
