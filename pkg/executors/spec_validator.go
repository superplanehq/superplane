package executors

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"gorm.io/datatypes"
)

type SpecValidator struct {
	Encryptor crypto.Encryptor
}

func (v *SpecValidator) Validate(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec) (*models.StageExecutor, *models.Resource, error) {
	if in == nil {
		return nil, nil, fmt.Errorf("missing executor spec")
	}

	switch in.Type {
	case pb.ExecutorSpec_TYPE_SEMAPHORE:
		return v.validateSemaphoreExecutorSpec(ctx, canvas, in)
	case pb.ExecutorSpec_TYPE_HTTP:
		return v.validateHTTPExecutorSpec(in)
	default:
		return nil, nil, errors.New("invalid executor spec type")
	}
}

func (v *SpecValidator) validateHTTPExecutorSpec(in *pb.ExecutorSpec) (*models.StageExecutor, *models.Resource, error) {
	if in.Http == nil {
		return nil, nil, fmt.Errorf("invalid HTTP executor spec: missing HTTP executor spec")
	}

	if in.Http.Url == "" {
		return nil, nil, fmt.Errorf("invalid HTTP executor spec: missing URL")
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
				return nil, nil, fmt.Errorf("invalid HTTP executor spec: invalid status code: %d", code)
			}
		}

		responsePolicy = &models.HTTPResponsePolicy{
			StatusCodes: in.Http.ResponsePolicy.StatusCodes,
		}
	}

	return &models.StageExecutor{
		Type: models.ExecutorSpecTypeHTTP,
		Spec: datatypes.NewJSONType(models.ExecutorSpec{
			HTTP: &models.HTTPExecutorSpec{
				URL:            in.Http.Url,
				Headers:        headers,
				Payload:        payload,
				ResponsePolicy: responsePolicy,
			},
		}),
	}, nil, nil
}

func (v *SpecValidator) validateSemaphoreExecutorSpec(ctx context.Context, canvas *models.Canvas, in *pb.ExecutorSpec) (*models.StageExecutor, *models.Resource, error) {
	if in.Semaphore == nil {
		return nil, nil, fmt.Errorf("invalid semaphore executor spec: missing semaphore executor spec")
	}

	if in.Integration == nil {
		return nil, nil, fmt.Errorf("invalid semaphore executor spec: missing integration")
	}

	// TODO: support for organization level canvas
	integration, err := models.FindIntegrationByName(authorization.DomainCanvas, canvas.ID, in.Integration.Name)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid semaphore executor spec: integration not found")
	}

	if integration.Type != models.IntegrationTypeSemaphore {
		return nil, nil, fmt.Errorf("invalid semaphore executor spec: integration is not of type semaphore")
	}

	i, err := integrations.NewIntegration(ctx, integration, v.Encryptor)
	if err != nil {
		return nil, nil, fmt.Errorf("error building integration: %v", err)
	}

	resource, err := i.Get(integrations.ResourceTypeProject, in.Semaphore.ProjectId)
	if err != nil {
		return nil, nil, fmt.Errorf("%s %s not found: %v", integrations.ResourceTypeProject, in.Semaphore.ProjectId, err)
	}

	return &models.StageExecutor{
			Type: models.ExecutorSpecTypeSemaphore,
			Spec: datatypes.NewJSONType(models.ExecutorSpec{
				Semaphore: &models.SemaphoreExecutorSpec{
					ProjectID:    in.Semaphore.ProjectId,
					Branch:       in.Semaphore.Branch,
					PipelineFile: in.Semaphore.PipelineFile,
					Parameters:   in.Semaphore.Parameters,
					TaskID:       in.Semaphore.TaskId,
				},
			}),
		}, &models.Resource{
			ExternalID:    resource.Id(),
			Name:          resource.Name(),
			IntegrationID: integration.ID,
			Type:          integrations.ResourceTypeProject,
		}, nil
}
