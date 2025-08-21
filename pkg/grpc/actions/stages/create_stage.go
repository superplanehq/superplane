package stages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/secrets"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateStage(ctx context.Context, encryptor crypto.Encryptor, registry *registry.Registry, orgID, canvasID string, stage *pb.Stage) (*pb.CreateStageResponse, error) {
	canvas, err := models.FindCanvasByID(canvasID, uuid.MustParse(orgID))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if stage == nil {
		return nil, status.Error(codes.InvalidArgument, "stage is required")
	}

	if stage.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.metadata is required")
	}

	if stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.spec is required")
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

	connections, err := actions.ValidateConnections(canvasID, stage.Spec.Connections)
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

	executorSpec, err := stage.Spec.Executor.Spec.MarshalJSON()
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to marshal executor spec: %v", err)
	}

	newStage, err := builders.NewStageBuilder(registry).
		WithContext(ctx).
		WithEncryptor(encryptor).
		InCanvas(canvas.ID).
		WithName(stage.Metadata.Name).
		WithDescription(stage.Metadata.Description).
		WithRequester(uuid.MustParse(userID)).
		WithConditions(conditions).
		WithConnections(connections).
		WithInputs(inputValidator.SerializeInputs()).
		WithInputMappings(inputValidator.SerializeInputMappings()).
		WithOutputs(inputValidator.SerializeOutputs()).
		WithSecrets(secrets).
		WithExecutorType(stage.Spec.Executor.Type).
		WithExecutorSpec(executorSpec).
		WithExecutorLabel(stage.Spec.Executor.Label).
		ForIntegration(integration).
		ForResource(resource).
		Create()

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

	response := &pb.CreateStageResponse{
		Stage: serialized,
	}

	err = messages.NewStageCreatedMessage(newStage).Publish()
	if err != nil {
		logging.ForStage(newStage).Errorf("failed to publish stage created message: %v", err)
	}

	return response, nil
}

func validateSecrets(ctx context.Context, encryptor crypto.Encryptor, canvas *models.Canvas, in []*pb.ValueDefinition) ([]models.ValueDefinition, error) {
	out := []models.ValueDefinition{}
	for _, s := range in {
		if s.Name == "" {
			return nil, fmt.Errorf("empty name")
		}

		if s.ValueFrom == nil || s.ValueFrom.Secret == nil {
			return nil, fmt.Errorf("missing secret")
		}

		if s.ValueFrom.Secret.Name == "" || s.ValueFrom.Secret.Key == "" {
			return nil, fmt.Errorf("missing secret name or key")
		}

		domainType, domainID, err := actions.GetDomainForSecret(models.DomainTypeCanvas, &canvas.ID, s.ValueFrom.Secret.DomainType)
		if err != nil {
			return nil, err
		}

		name := s.ValueFrom.Secret.Name
		provider, err := secrets.NewProvider(encryptor, name, domainType, *domainID)
		if err != nil {
			return nil, err
		}

		values, err := provider.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("error loading values for secret %s: %v", name, err)
		}

		key := s.ValueFrom.Secret.Key
		_, ok := values[key]
		if !ok {
			return nil, fmt.Errorf("key %s not found in secret %s", key, name)
		}

		out = append(out, models.ValueDefinition{
			Name:  s.Name,
			Value: nil,
			ValueFrom: &models.ValueDefinitionFrom{
				Secret: &models.ValueDefinitionFromSecret{
					DomainType: domainType,
					Name:       s.ValueFrom.Secret.Name,
					Key:        s.ValueFrom.Secret.Key,
				},
			},
		})
	}

	return out, nil
}

func validateConditions(conditions []*pb.Condition) ([]models.StageCondition, error) {
	cs := []models.StageCondition{}

	for _, condition := range conditions {
		c, err := validateCondition(condition)
		if err != nil {
			return nil, fmt.Errorf("invalid condition: %v", err)
		}

		cs = append(cs, *c)
	}

	return cs, nil
}

func validateCondition(condition *pb.Condition) (*models.StageCondition, error) {
	switch condition.Type {
	case pb.Condition_CONDITION_TYPE_APPROVAL:
		if condition.Approval == nil {
			return nil, fmt.Errorf("missing approval settings")
		}

		if condition.Approval.Count == 0 {
			return nil, fmt.Errorf("invalid approval condition: count must be greater than 0")
		}

		return &models.StageCondition{
			Type: models.StageConditionTypeApproval,
			Approval: &models.ApprovalCondition{
				Count: int(condition.Approval.Count),
			},
		}, nil

	case pb.Condition_CONDITION_TYPE_TIME_WINDOW:
		if condition.TimeWindow == nil {
			return nil, fmt.Errorf("missing time window settings")
		}

		c := condition.TimeWindow
		t, err := models.NewTimeWindowCondition(c.Start, c.End, c.WeekDays)
		if err != nil {
			return nil, fmt.Errorf("invalid time window condition: %v", err)
		}

		return &models.StageCondition{
			Type:       models.StageConditionTypeTimeWindow,
			TimeWindow: t,
		}, nil

	default:
		return nil, fmt.Errorf("invalid condition type: %s", condition.Type)
	}
}

func serializeStage(
	stage models.Stage,
	connections []*pb.Connection,
	inputs []*pb.InputDefinition,
	outputs []*pb.OutputDefinition,
	inputMappings []*pb.InputMapping,
) (*pb.Stage, error) {
	executor, err := serializeExecutor(stage)
	if err != nil {
		return nil, err
	}

	conditions, err := serializeConditions(stage.Conditions)
	if err != nil {
		return nil, err
	}

	secrets := []*pb.ValueDefinition{}
	for _, valueDef := range stage.Secrets {
		secrets = append(secrets, serializeValueDefinition(valueDef))
	}

	return &pb.Stage{
		Metadata: &pb.Stage_Metadata{
			Id:          stage.ID.String(),
			Name:        stage.Name,
			Description: stage.Description,
			CanvasId:    stage.CanvasID.String(),
			CreatedAt:   timestamppb.New(*stage.CreatedAt),
		},
		Spec: &pb.Stage_Spec{
			Conditions:    conditions,
			Connections:   connections,
			Executor:      executor,
			Inputs:        inputs,
			Outputs:       outputs,
			InputMappings: inputMappings,
			Secrets:       secrets,
		},
	}, nil
}

func serializeInputs(in []models.InputDefinition) []*pb.InputDefinition {
	out := []*pb.InputDefinition{}
	for _, def := range in {
		out = append(out, &pb.InputDefinition{
			Name:        def.Name,
			Description: def.Description,
		})
	}

	return out
}

func serializeOutputs(in []models.OutputDefinition) []*pb.OutputDefinition {
	out := []*pb.OutputDefinition{}
	for _, def := range in {
		out = append(out, &pb.OutputDefinition{
			Name:        def.Name,
			Description: def.Description,
			Required:    def.Required,
		})
	}

	return out
}

func serializeInputMappings(in []models.InputMapping) []*pb.InputMapping {
	out := []*pb.InputMapping{}
	for _, m := range in {
		mapping := &pb.InputMapping{
			Values: []*pb.ValueDefinition{},
		}

		for _, valueDef := range m.Values {
			mapping.Values = append(mapping.Values, serializeValueDefinition(valueDef))
		}

		if m.When != nil && m.When.TriggeredBy != nil {
			mapping.When = &pb.InputMapping_When{
				TriggeredBy: &pb.InputMapping_WhenTriggeredBy{
					Connection: m.When.TriggeredBy.Connection,
				},
			}
		}

		out = append(out, mapping)
	}

	return out
}

func serializeValueDefinition(in models.ValueDefinition) *pb.ValueDefinition {
	v := &pb.ValueDefinition{
		Name: in.Name,
	}

	if in.Value != nil {
		v.Value = *in.Value
	}

	if in.ValueFrom != nil {
		v.ValueFrom = serializeValueFrom(*in.ValueFrom)
	}

	return v
}

func serializeValueFrom(in models.ValueDefinitionFrom) *pb.ValueFrom {
	if in.EventData != nil {
		return &pb.ValueFrom{
			EventData: &pb.ValueFromEventData{
				Connection: in.EventData.Connection,
				Expression: in.EventData.Expression,
			},
		}
	}

	if in.LastExecution != nil {
		results := []pb.Execution_Result{}
		for _, r := range in.LastExecution.Results {
			results = append(results, actions.ExecutionResultToProto(r))
		}

		return &pb.ValueFrom{
			LastExecution: &pb.ValueFromLastExecution{
				Results: results,
			},
		}
	}

	if in.Secret != nil {
		return &pb.ValueFrom{
			Secret: &pb.ValueFromSecret{
				DomainType: actions.DomainTypeToProto(in.Secret.DomainType),
				Name:       in.Secret.Name,
				Key:        in.Secret.Key,
			},
		}
	}

	return nil
}

func serializeConditions(conditions []models.StageCondition) ([]*pb.Condition, error) {
	cs := []*pb.Condition{}

	for _, condition := range conditions {
		c, err := serializeCondition(condition)
		if err != nil {
			return nil, fmt.Errorf("invalid condition: %v", err)
		}

		cs = append(cs, c)
	}

	return cs, nil
}

func serializeCondition(condition models.StageCondition) (*pb.Condition, error) {
	switch condition.Type {
	case models.StageConditionTypeApproval:
		return &pb.Condition{
			Type: pb.Condition_CONDITION_TYPE_APPROVAL,
			Approval: &pb.ConditionApproval{
				Count: uint32(condition.Approval.Count),
			},
		}, nil

	case models.StageConditionTypeTimeWindow:
		return &pb.Condition{
			Type: pb.Condition_CONDITION_TYPE_TIME_WINDOW,
			TimeWindow: &pb.ConditionTimeWindow{
				Start:    condition.TimeWindow.Start,
				End:      condition.TimeWindow.End,
				WeekDays: condition.TimeWindow.WeekDays,
			},
		}, nil

	default:
		return nil, fmt.Errorf("invalid condition type: %s", condition.Type)
	}
}

func serializeExecutor(stage models.Stage) (*pb.Executor, error) {
	var executorSpec map[string]any
	err := json.Unmarshal(stage.ExecutorSpec, &executorSpec)
	if err != nil {
		return nil, err
	}

	spec, err := structpb.NewStruct(executorSpec)
	if err != nil {
		return nil, err
	}

	if stage.ResourceID == nil {
		return &pb.Executor{
			Type:  stage.ExecutorType,
			Spec:  spec,
			Label: stage.ExecutorLabel,
		}, nil
	}

	integrationResource, err := stage.GetIntegrationResource()
	if err != nil {
		return nil, err
	}

	return &pb.Executor{
		Type:  stage.ExecutorType,
		Spec:  spec,
		Label: stage.ExecutorLabel,
		Integration: &integrationpb.IntegrationRef{
			Name:       integrationResource.IntegrationName,
			DomainType: actions.DomainTypeToProto(integrationResource.DomainType),
		},
		Resource: &integrationpb.ResourceRef{
			Type: integrationResource.Type,
			Name: integrationResource.Name,
		},
	}, nil
}

func serializeStages(stages []models.Stage) ([]*pb.Stage, error) {
	s := []*pb.Stage{}
	for _, stage := range stages {
		connections, err := models.ListConnections(stage.ID, models.ConnectionTargetTypeStage)
		if err != nil {
			return nil, err
		}

		serialized, err := actions.SerializeConnections(connections)
		if err != nil {
			return nil, err
		}

		stage, err := serializeStage(
			stage,
			serialized,
			serializeInputs(stage.Inputs),
			serializeOutputs(stage.Outputs),
			serializeInputMappings(stage.InputMappings),
		)

		if err != nil {
			return nil, err
		}

		s = append(s, stage)
	}

	return s, nil
}
