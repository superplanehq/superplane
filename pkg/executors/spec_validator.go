package executors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

type SpecValidator struct {
}

func (v *SpecValidator) Validate(in *pb.ExecutorSpec) (*models.ExecutorSpec, error) {
	if in == nil {
		return nil, fmt.Errorf("missing executor spec")
	}

	switch in.Type {
	case pb.ExecutorSpec_TYPE_SEMAPHORE:
		return v.validateSemaphoreExecutorSpec(in)
	case pb.ExecutorSpec_TYPE_HTTP:
		return v.validateHTTPExecutorSpec(in)
	default:
		return nil, errors.New("invalid executor spec type")
	}
}

func (v *SpecValidator) validateHTTPExecutorSpec(in *pb.ExecutorSpec) (*models.ExecutorSpec, error) {
	if in.Http == nil {
		return nil, fmt.Errorf("invalid HTTP executor spec: missing HTTP executor spec")
	}

	if in.Http.Url == "" {
		return nil, fmt.Errorf("invalid HTTP executor spec: missing URL")
	}

	headers := in.Http.Headers
	if headers == nil {
		headers = map[string]string{}
	}

	payload := in.Http.Payload
	if payload == nil {
		payload = map[string]string{}
	}

	var responsePolicy *models.HTTPResponsePolicy
	if in.Http.ResponsePolicy == nil || len(in.Http.ResponsePolicy.StatusCodes) == 0 {
		responsePolicy = &models.HTTPResponsePolicy{
			StatusCodes: []uint32{http.StatusOK},
		}
	} else {
		for _, code := range in.Http.ResponsePolicy.StatusCodes {
			if code < http.StatusOK || code > http.StatusNetworkAuthenticationRequired {
				return nil, fmt.Errorf("invalid HTTP executor spec: invalid status code: %d", code)
			}
		}

		responsePolicy = &models.HTTPResponsePolicy{
			StatusCodes: in.Http.ResponsePolicy.StatusCodes,
		}
	}

	return &models.ExecutorSpec{
		Type: models.ExecutorSpecTypeHTTP,
		HTTP: &models.HTTPExecutorSpec{
			URL:            in.Http.Url,
			Headers:        headers,
			Payload:        payload,
			ResponsePolicy: responsePolicy,
		},
	}, nil
}

func (v *SpecValidator) validateSemaphoreExecutorSpec(in *pb.ExecutorSpec) (*models.ExecutorSpec, error) {
	if in.Semaphore == nil {
		return nil, fmt.Errorf("invalid semaphore executor spec: missing semaphore executor spec")
	}

	if in.Semaphore.OrganizationUrl == "" {
		return nil, fmt.Errorf("invalid semaphore executor spec: missing organization URL")
	}

	if in.Semaphore.ApiToken == "" {
		return nil, fmt.Errorf("invalid semaphore executor spec: missing API token")
	}

	if in.Semaphore.TaskId == "" {
		return nil, fmt.Errorf("invalid semaphore executor spec: only triggering tasks is supported for now")
	}

	return &models.ExecutorSpec{
		Type: models.ExecutorSpecTypeSemaphore,
		Semaphore: &models.SemaphoreExecutorSpec{
			OrganizationURL: in.Semaphore.OrganizationUrl,
			APIToken:        in.Semaphore.ApiToken,
			ProjectID:       in.Semaphore.ProjectId,
			Branch:          in.Semaphore.Branch,
			PipelineFile:    in.Semaphore.PipelineFile,
			Parameters:      in.Semaphore.Parameters,
			TaskID:          in.Semaphore.TaskId,
		},
	}, nil
}
