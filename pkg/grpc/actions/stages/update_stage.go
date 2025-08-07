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

func UpdateStage(
	ctx context.Context,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	canvasID string,
	idOrName string,
	stage *pb.Stage,
) (*pb.UpdateStageResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvasByIDOnly(canvasID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(idOrName)
	var existingStage *models.Stage
	if err != nil {
		existingStage, err = models.FindStageByName(canvasID, idOrName)
	} else {
		existingStage, err = models.FindStageByID(canvasID, idOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	if stage == nil || stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage spec is required")
	}

	if stage.Metadata != nil && stage.Metadata.Name != "" && stage.Metadata.Name != existingStage.Name {
		_, err := models.FindStageByName(canvasID, stage.Metadata.Name)
		if err == nil {
			return nil, status.Error(codes.InvalidArgument, "stage name already in use")
		}
		existingStage.Name = stage.Metadata.Name
	}

	if stage.Metadata != nil && stage.Metadata.Description != "" {
		existingStage.Description = stage.Metadata.Description
	}

	//
	// It is OK to create a stage without an integration.
	//
	var integration *models.Integration
	if stage.Spec != nil && stage.Spec.Executor != nil && stage.Spec.Executor.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, stage.Spec.Executor.Integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// If integration is defined, find the integration resource we are interested in.
	//
	var resource integrations.Resource
	if integration != nil {
		resource, err = actions.ValidateResource(ctx, registry, integration, stage.Spec.Executor.Resource)
		if err != nil {
			return nil, err
		}
	}

	inputValidator := inputs.NewValidator(
		inputs.WithInputs(stage.Spec.Inputs),
		inputs.WithOutputs(stage.Spec.Outputs),
		inputs.WithInputMappings(stage.Spec.InputMappings),
		inputs.WithConnections(stage.Spec.Connections),
	)

	err = inputValidator.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvas.ID.String(), stage.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(stage.Spec.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets, err := validateSecrets(ctx, encryptor, canvas, stage.Spec.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	executorSpec, err := stage.Spec.Executor.Spec.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to marshal executor spec: %v", err)
	}

	newStage, err := builders.NewStageBuilder(registry).
		WithContext(ctx).
		WithExistingStage(existingStage).
		WithName(existingStage.Name).
		WithDescription(existingStage.Description).
		WithEncryptor(encryptor).
		InCanvas(canvas.ID).
		WithRequester(uuid.MustParse(userID)).
		WithConditions(conditions).
		WithConnections(connections).
		WithInputs(inputValidator.SerializeInputs()).
		WithInputMappings(inputValidator.SerializeInputMappings()).
		WithOutputs(inputValidator.SerializeOutputs()).
		WithSecrets(secrets).
		WithExecutorType(stage.Spec.Executor.Type).
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

	serialized, err := serializeStage(
		*newStage,
		stage.Spec.Connections,
		stage.Spec.Inputs,
		stage.Spec.Outputs,
		stage.Spec.InputMappings,
	)

	if err != nil {
		return nil, err
	}

	response := &pb.UpdateStageResponse{
		Stage: serialized,
	}

	err = messages.NewStageCreatedMessage(newStage).Publish()

	if err != nil {
		logging.ForStage(newStage).Errorf("failed to publish stage created message: %v", err)
	}

	return response, nil
}
