package executors

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
)

type SpecValidator struct {
	Encryptor crypto.Encryptor
}

type SpecValidationResponse struct {
	Integration      *models.Integration
	ExecutorType     string
	ExecutorSpec     *models.ExecutorSpec
	ExecutorResource *models.Resource
}

func (v *SpecValidator) Validate(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec) (*SpecValidationResponse, error) {
	if in == nil {
		return nil, fmt.Errorf("missing executor spec")
	}

	switch in.Type {
	case pb.ExecutorSpec_TYPE_SEMAPHORE:
		return v.validateSemaphoreExecutorSpec(ctx, canvas, in)
	case pb.ExecutorSpec_TYPE_HTTP:
		return v.validateHTTPExecutorSpec(in)
	default:
		return nil, errors.New("invalid executor spec type")
	}
}

func (v *SpecValidator) validateHTTPExecutorSpec(in *pb.ExecutorSpec) (*SpecValidationResponse, error) {
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

	return &SpecValidationResponse{
		ExecutorType: models.ExecutorSpecTypeHTTP,
		ExecutorSpec: &models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL:            in.Http.Url,
				Headers:        headers,
				Payload:        payload,
				ResponsePolicy: responsePolicy,
			},
		},
	}, nil
}

func (v *SpecValidator) validateSemaphoreExecutorSpec(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec) (*SpecValidationResponse, error) {
	if in.Semaphore == nil {
		return nil, fmt.Errorf("invalid semaphore executor spec: missing semaphore executor spec")
	}

	if in.Integration == nil {
		return nil, fmt.Errorf("invalid semaphore executor spec: missing integration")
	}

	// TODO: support for organization level canvas
	integration, err := models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, in.Integration.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid semaphore executor spec: integration not found")
	}

	if integration.Type != models.IntegrationTypeSemaphore {
		return nil, fmt.Errorf("invalid semaphore executor spec: integration is not of type semaphore")
	}

	i, err := integrations.NewIntegration(ctx, integration, v.Encryptor)
	if err != nil {
		return nil, fmt.Errorf("error building integration: %v", err)
	}

	project, err := i.Get(integrations.ResourceTypeProject, in.Semaphore.Project)
	if err != nil {
		return nil, fmt.Errorf("project %s not found: %v", in.Semaphore.Project, err)
	}

	taskId, err := v.findSemaphoreTaskId(i, project, in.Semaphore)
	if err != nil {
		return nil, err
	}

	return &SpecValidationResponse{
		Integration:  integration,
		ExecutorType: models.ExecutorSpecTypeSemaphore,
		ExecutorSpec: &models.ExecutorSpec{
			Semaphore: &models.SemaphoreExecutorSpec{
				TaskId:       taskId,
				Branch:       in.Semaphore.Branch,
				PipelineFile: in.Semaphore.PipelineFile,
				Parameters:   in.Semaphore.Parameters,
			},
		},
		ExecutorResource: &models.Resource{
			ExternalID:    project.Id(),
			ResourceName:  project.Name(),
			IntegrationID: integration.ID,
			ResourceType:  integrations.ResourceTypeProject,
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
		task, err := i.Get(integrations.ResourceTypeTask, spec.Task, project.Id())
		if err != nil {
			return nil, fmt.Errorf("task %s not found: %v", spec.Task, err)
		}

		taskId := task.Id()
		return &taskId, nil
	}

	//
	// If task is a string, we have to list tasks and find the one that matches.
	//
	tasks, err := i.List(integrations.ResourceTypeTask, project.Id())
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
