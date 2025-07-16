package stages

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/builders"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/executors"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/inputs"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateStage(ctx context.Context, encryptor crypto.Encryptor, specValidator executors.SpecValidator, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	if req.Stage == nil {
		return nil, status.Error(codes.InvalidArgument, "stage is required")
	}

	if req.Stage.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.metadata is required")
	}

	if req.Stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.spec is required")
	}

	err := actions.ValidateUUIDs(req.CanvasIdOrName)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
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

	secrets, err := validateSecrets(req.Stage.Spec.Secrets)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	specValidationResponse, err := specValidator.Validate(ctx, canvas, req.Stage.Spec.Executor)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	stage, err := builders.NewStageBuilder().
		WithContext(ctx).
		WithEncryptor(encryptor).
		InCanvas(canvas).
		WithName(req.Stage.Metadata.Name).
		WithRequester(uuid.MustParse(userID)).
		WithConditions(conditions).
		WithConnections(connections).
		WithInputs(inputValidator.SerializeInputs()).
		WithInputMappings(inputValidator.SerializeInputMappings()).
		WithOutputs(inputValidator.SerializeOutputs()).
		WithSecrets(secrets).
		WithExecutorType(specValidationResponse.ExecutorType).
		WithExecutorSpec(specValidationResponse.ExecutorSpec).
		WithExecutorResource(specValidationResponse.ExecutorResource).
		Create()

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

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

	response := &pb.CreateStageResponse{
		Stage: serialized,
	}

	err = messages.NewStageCreatedMessage(stage).Publish()

	if err != nil {
		logging.ForStage(stage).Errorf("failed to publish stage created message: %v", err)
	}

	return response, nil
}

func validateSecrets(in []*pb.ValueDefinition) ([]models.ValueDefinition, error) {
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

		out = append(out, models.ValueDefinition{
			Name:  s.Name,
			Value: nil,
			ValueFrom: &models.ValueDefinitionFrom{
				Secret: &models.ValueDefinitionFromSecret{
					Name: s.ValueFrom.Secret.Name,
					Key:  s.ValueFrom.Secret.Key,
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
	stageExecutor, err := stage.GetExecutor()
	if err != nil {
		return nil, err
	}

	executor, err := serializeExecutor(stageExecutor)
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
			Id:        stage.ID.String(),
			Name:      stage.Name,
			CanvasId:  stage.CanvasID.String(),
			CreatedAt: timestamppb.New(*stage.CreatedAt),
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
				Name: in.Secret.Name,
				Key:  in.Secret.Key,
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

func serializeExecutor(executor *models.StageExecutor) (*pb.ExecutorSpec, error) {
	executorSpec := executor.Spec.Data()

	switch executor.Type {
	case models.ExecutorSpecTypeHTTP:
		return &pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_HTTP,
			Http: &pb.ExecutorSpec_HTTP{
				Url:     executorSpec.HTTP.URL,
				Headers: executorSpec.HTTP.Headers,
				Payload: executorSpec.HTTP.Payload,
				ResponsePolicy: &pb.ExecutorSpec_HTTPResponsePolicy{
					StatusCodes: executorSpec.HTTP.ResponsePolicy.StatusCodes,
				},
			},
		}, nil
	case models.ExecutorSpecTypeSemaphore:
		resource, err := executor.GetResource()
		if err != nil {
			return nil, err
		}

		spec := &pb.ExecutorSpec_Semaphore{
			Project:      resource.ResourceName,
			Branch:       executorSpec.Semaphore.Branch,
			PipelineFile: executorSpec.Semaphore.PipelineFile,
			Parameters:   executorSpec.Semaphore.Parameters,
		}

		if executorSpec.Semaphore.TaskId != nil {
			spec.Task = *executorSpec.Semaphore.TaskId
		}

		return &pb.ExecutorSpec{
			Type:      pb.ExecutorSpec_TYPE_SEMAPHORE,
			Semaphore: spec,
		}, nil

	default:
		return nil, fmt.Errorf("invalid executor spec type: %s", executor.Type)
	}
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
