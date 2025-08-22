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

func UpdateStage(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, orgID, canvasID, idOrName string, newStage *pb.Stage) (*pb.UpdateStageResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvasByID(canvasID, uuid.MustParse(orgID))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	err = actions.ValidateUUIDs(idOrName)
	var stage *models.Stage
	if err != nil {
		stage, err = models.FindStageByName(canvasID, idOrName)
	} else {
		stage, err = models.FindStageByID(canvasID, idOrName)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.InvalidArgument, "stage not found")
		}

		return nil, err
	}

	if newStage == nil || newStage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage spec is required")
	}

	if newStage.Metadata != nil && newStage.Metadata.Name != "" && newStage.Metadata.Name != stage.Name {
		_, err := models.FindStageByName(canvasID, newStage.Metadata.Name)
		if err == nil {
			return nil, status.Error(codes.InvalidArgument, "stage name already in use")
		}
		stage.Name = newStage.Metadata.Name
	}

	if newStage.Metadata != nil && newStage.Metadata.Description != "" {
		stage.Description = newStage.Metadata.Description
	}

	//
	// It is OK to create a stage without an integration.
	//
	var integration *models.Integration
	if newStage.Spec != nil && newStage.Spec.Executor != nil && newStage.Spec.Executor.Integration != nil {
		integration, err = actions.ValidateIntegration(canvas, newStage.Spec.Executor.Integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// If integration is defined, find the integration resource we are interested in.
	//
	var resource integrations.Resource
	if integration != nil {
		resource, err = actions.ValidateResource(ctx, registry, integration, newStage.Spec.Executor.Resource)
		if err != nil {
			return nil, err
		}
	}

	inputValidator := inputs.NewValidator(
		inputs.WithInputs(newStage.Spec.Inputs),
		inputs.WithOutputs(newStage.Spec.Outputs),
		inputs.WithInputMappings(newStage.Spec.InputMappings),
		inputs.WithConnections(newStage.Spec.Connections),
	)

	err = inputValidator.Validate()
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	connections, err := actions.ValidateConnections(canvasID, newStage.Spec.Connections)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(newStage.Spec.Conditions)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	secrets, err := validateSecrets(ctx, encryptor, canvas, newStage.Spec.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	executorSpec, err := newStage.Spec.Executor.Spec.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to marshal executor spec: %v", err)
	}

	stage, err = builders.NewStageBuilder(registry).
		WithContext(ctx).
		WithExistingStage(stage).
		WithName(stage.Name).
		WithDescription(stage.Description).
		WithEncryptor(encryptor).
		InCanvas(uuid.MustParse(canvasID)).
		WithRequester(uuid.MustParse(userID)).
		WithConditions(conditions).
		WithConnections(connections).
		WithInputs(inputValidator.SerializeInputs()).
		WithInputMappings(inputValidator.SerializeInputMappings()).
		WithOutputs(inputValidator.SerializeOutputs()).
		WithSecrets(secrets).
		WithExecutorType(newStage.Spec.Executor.Type).
		WithExecutorSpec(executorSpec).
		WithExecutorName(newStage.Spec.Executor.Name).
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
		*stage,
		newStage.Spec.Connections,
		newStage.Spec.Inputs,
		newStage.Spec.Outputs,
		newStage.Spec.InputMappings,
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
