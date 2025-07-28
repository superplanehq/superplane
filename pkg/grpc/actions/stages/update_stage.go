package stages

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func UpdateStage(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, req *pb.UpdateStageRequest) (*pb.UpdateStageResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	err := actions.ValidateUUIDs(req.IdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(req.IdOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = canvas.FindStageByName(req.IdOrName)
	} else {
		stage, err = canvas.FindStageByID(req.IdOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	if req.Stage == nil || req.Stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage spec is required")
	}

	//
	// It is OK to create a stage without an integration.
	//
	var integration *models.Integration
	if req.Stage.Spec != nil && req.Stage.Spec.Executor != nil && req.Stage.Spec.Executor.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, req.Stage.Spec.Executor.Integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// If integration is defined, find the integration resource we are interested in.
	//
	var resource integrations.Resource
	if integration != nil {
		resource, err = actions.ValidateResource(ctx, registry, integration, req.Stage.Spec.Executor.Resource)
		if err != nil {
			return nil, err
		}
	}

	inputValidator := inputs.NewValidator(
		inputs.WithInputs(req.Stage.Spec.Inputs),
		inputs.WithOutputs(req.Stage.Spec.Outputs),
		inputs.WithInputMappings(req.Stage.Spec.InputMappings),
		inputs.WithConnections(req.Stage.Spec.Connections),
	)

	err = inputValidator.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvas, req.Stage.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(req.Stage.Spec.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets, err := validateSecrets(ctx, encryptor, canvas, req.Stage.Spec.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	executorSpec, err := req.Stage.Spec.Executor.Spec.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to marshal executor spec: %v", err)
	}

	stage, err = builders.NewStageBuilder(registry).
		WithContext(ctx).
		WithExistingStage(stage).
		WithEncryptor(encryptor).
		InCanvas(canvas).
		WithRequester(uuid.MustParse(userID)).
		WithConditions(conditions).
		WithConnections(connections).
		WithInputs(inputValidator.SerializeInputs()).
		WithInputMappings(inputValidator.SerializeInputMappings()).
		WithOutputs(inputValidator.SerializeOutputs()).
		WithSecrets(secrets).
		WithExecutorType(req.Stage.Spec.Executor.Type).
		WithExecutorSpec(executorSpec).
		ForResource(resource).
		ForIntegration(integration).
		Update()

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err = canvas.FindStageByID(stage.ID.String())
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStage(
		*stage,
		req.Stage.Spec.Connections,
		req.Stage.Spec.Inputs,
		req.Stage.Spec.Outputs,
		req.Stage.Spec.InputMappings,
	)

	if err != nil {
		return nil, err
	}

	response := &pb.UpdateStageResponse{
		Stage: serialized,
	}

	err = messages.NewStageCreatedMessage(stage).Publish()

	if err != nil {
		logging.ForStage(stage).Errorf("failed to publish stage created message: %v", err)
	}

	return response, nil
}
