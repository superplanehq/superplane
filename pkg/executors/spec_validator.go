package executors

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/integrations/semaphore"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
)

type SpecValidator struct {
	Encryptor crypto.Encryptor
}

func (v *SpecValidator) Validate(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec, integration *models.Integration, resource integrations.Resource) (string, *models.ExecutorSpec, error) {
	if in == nil {
		return "", nil, fmt.Errorf("missing executor spec")
	}

	switch in.Type {
	case pb.ExecutorSpec_TYPE_SEMAPHORE:
		return v.validateSemaphoreExecutorSpec(ctx, canvas, in, integration, resource)
	case pb.ExecutorSpec_TYPE_HTTP:
		return v.validateHTTPExecutorSpec(in)
	default:
		return "", nil, errors.New("invalid executor spec type")
	}
}

func (v *SpecValidator) validateHTTPExecutorSpec(in *pb.ExecutorSpec) (string, *models.ExecutorSpec, error) {
	if in.Http == nil {
		return "", nil, fmt.Errorf("invalid HTTP executor spec: missing HTTP executor spec")
	}

	if in.Http.Url == "" {
		return "", nil, fmt.Errorf("invalid HTTP executor spec: missing URL")
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
				return "", nil, fmt.Errorf("invalid HTTP executor spec: invalid status code: %d", code)
			}
		}

		responsePolicy = &models.HTTPResponsePolicy{
			StatusCodes: in.Http.ResponsePolicy.StatusCodes,
		}
	}

	return models.ExecutorSpecTypeHTTP, &models.ExecutorSpec{
		HTTP: &models.HTTPExecutorSpec{
			URL:            in.Http.Url,
			Headers:        headers,
			Payload:        payload,
			ResponsePolicy: responsePolicy,
		},
	}, nil
}

func (v *SpecValidator) validateSemaphoreExecutorSpec(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec, integration *models.Integration, resource integrations.Resource) (string, *models.ExecutorSpec, error) {
	if in.Semaphore == nil {
		return "", nil, fmt.Errorf("invalid semaphore executor spec: missing semaphore executor spec")
	}

	if in.Integration == nil {
		return "", nil, fmt.Errorf("invalid semaphore executor spec: missing integration")
	}

	i, err := integrations.NewIntegration(ctx, integration, v.Encryptor)
	if err != nil {
		return "", nil, fmt.Errorf("error building integration: %v", err)
	}

	taskId, err := v.findSemaphoreTaskId(i, resource, in.Semaphore)
	if err != nil {
		return "", nil, err
	}

	return models.ExecutorSpecTypeSemaphore, &models.ExecutorSpec{
		Semaphore: &models.SemaphoreExecutorSpec{
			TaskId:       taskId,
			Branch:       in.Semaphore.Branch,
			PipelineFile: in.Semaphore.PipelineFile,
			Parameters:   in.Semaphore.Parameters,
		},
	}, nil
}

func (v *SpecValidator) findSemaphoreTaskId(i integrations.Integration, project integrations.Resource, spec *pb.ExecutorSpec_Semaphore) (*string, error) {
	if spec.Task == "" {
		return nil, nil
	}

	//
	// If task is a UUID, we describe to validate that it exists.
	//
	_, err := uuid.Parse(spec.Task)
	if err == nil {
		task, err := i.Get(semaphore.ResourceTypeTask, spec.Task, project.Id())
		if err != nil {
			return nil, fmt.Errorf("task %s not found: %v", spec.Task, err)
		}

		taskId := task.Id()
		return &taskId, nil
	}

	//
	// If task is a string, we have to list tasks and find the one that matches.
	//
	tasks, err := i.List(semaphore.ResourceTypeTask, project.Id())
	if err != nil {
		return nil, fmt.Errorf("error listing tasks: %v", err)
	}

	for _, task := range tasks {
		if task.Name() == spec.Task {
			taskId := task.Id()
			return &taskId, nil
		}
	}

	return nil, fmt.Errorf("task %s not found", spec.Task)
}
