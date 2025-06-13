package stages

import (
	"context"
	"errors"
	"fmt"
	"sort"

	uuid "github.com/google/uuid"
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

func CreateStage(ctx context.Context, specValidator executors.SpecValidator, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	if req.Stage == nil {
		return nil, status.Error(codes.InvalidArgument, "stage is required")
	}

	if req.Stage.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.metadata is required")
	}

	if req.Stage.Spec == nil {
		return nil, status.Error(codes.InvalidArgument, "stage.spec is required")
	}

	err := actions.ValidateUUIDs(req.CanvasIdOrName, req.RequesterId)
	var canvas *models.Canvas
	if err != nil {
		canvas, err = models.FindCanvasByName(req.CanvasIdOrName)
	} else {
		canvas, err = models.FindCanvasByID(req.CanvasIdOrName)
	}

	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "canvas not found")
	}

	spec, err := specValidator.Validate(req.Stage.Spec.Executor)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

	connections, err := validateConnections(canvas, req.Stage.Spec.Connections)
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

	err = canvas.CreateStage(
		req.Stage.Metadata.Name,
		req.RequesterId,
		conditions,
		*spec,
		connections,
		inputValidator.SerializeInputs(),
		inputValidator.SerializeInputMappings(),
		inputValidator.SerializeOutputs(),
		secrets,
	)

	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err := canvas.FindStageByName(req.Stage.Metadata.Name)
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

func validateConnections(canvas *models.Canvas, connections []*pb.Connection) ([]models.StageConnection, error) {
	cs := []models.StageConnection{}

	if len(connections) == 0 {
		return nil, fmt.Errorf("connections must not be empty")
	}

	for _, connection := range connections {
		sourceID, err := findConnectionSourceID(canvas, connection)
		if err != nil {
			return nil, fmt.Errorf("invalid connection: %v", err)
		}

		filters, err := validateFilters(connection.Filters)
		if err != nil {
			return nil, err
		}

		cs = append(cs, models.StageConnection{
			SourceID:       *sourceID,
			SourceName:     connection.Name,
			SourceType:     protoToConnectionType(connection.Type),
			FilterOperator: protoToFilterOperator(connection.FilterOperator),
			Filters:        filters,
		})
	}

	return cs, nil
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

func validateFilters(in []*pb.Connection_Filter) ([]models.StageConnectionFilter, error) {
	filters := []models.StageConnectionFilter{}
	for i, f := range in {
		filter, err := validateFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter [%d]: %v", i, err)
		}

		filters = append(filters, *filter)
	}

	return filters, nil
}

func validateFilter(filter *pb.Connection_Filter) (*models.StageConnectionFilter, error) {
	switch filter.Type {
	case pb.Connection_FILTER_TYPE_DATA:
		return validateDataFilter(filter.Data)
	case pb.Connection_FILTER_TYPE_HEADER:
		return validateHeaderFilter(filter.Header)
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filter.Type)
	}
}

func validateDataFilter(filter *pb.Connection_DataFilter) (*models.StageConnectionFilter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.StageConnectionFilter{
		Type: models.FilterTypeData,
		Data: &models.DataFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func validateHeaderFilter(filter *pb.Connection_HeaderFilter) (*models.StageConnectionFilter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.StageConnectionFilter{
		Type: models.FilterTypeHeader,
		Header: &models.HeaderFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func protoToFilterOperator(in pb.Connection_FilterOperator) string {
	switch in {
	case pb.Connection_FILTER_OPERATOR_OR:
		return models.FilterOperatorOr
	default:
		return models.FilterOperatorAnd
	}
}

func filterOperatorToProto(in string) pb.Connection_FilterOperator {
	switch in {
	case models.FilterOperatorOr:
		return pb.Connection_FILTER_OPERATOR_OR
	default:
		return pb.Connection_FILTER_OPERATOR_AND
	}
}

func serializeFilters(in []models.StageConnectionFilter) ([]*pb.Connection_Filter, error) {
	filters := []*pb.Connection_Filter{}

	for _, f := range in {
		filter, err := serializeFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func serializeFilter(in models.StageConnectionFilter) (*pb.Connection_Filter, error) {
	switch in.Type {
	case models.FilterTypeData:
		return &pb.Connection_Filter{
			Type: pb.Connection_FILTER_TYPE_DATA,
			Data: &pb.Connection_DataFilter{
				Expression: in.Data.Expression,
			},
		}, nil
	case models.FilterTypeHeader:
		return &pb.Connection_Filter{
			Type: pb.Connection_FILTER_TYPE_HEADER,
			Header: &pb.Connection_HeaderFilter{
				Expression: in.Header.Expression,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid filter type: %s", in.Type)
	}
}

func serializeConnections(stages []models.Stage, sources []models.EventSource, in []models.StageConnection) ([]*pb.Connection, error) {
	connections := []*pb.Connection{}

	for _, c := range in {
		name, err := findConnectionName(stages, sources, c)
		if err != nil {
			return nil, fmt.Errorf("invalid connection: %v", err)
		}

		filters, err := serializeFilters(c.Filters)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %v", err)
		}

		connections = append(connections, &pb.Connection{
			Type:           connectionTypeToProto(c.SourceType),
			Name:           name,
			FilterOperator: filterOperatorToProto(c.FilterOperator),
			Filters:        filters,
		})
	}

	//
	// Sort them by name so we have some predictability here.
	//
	sort.SliceStable(connections, func(i, j int) bool {
		return connections[i].Name < connections[j].Name
	})

	return connections, nil
}

func connectionTypeToProto(t string) pb.Connection_Type {
	switch t {
	case models.SourceTypeStage:
		return pb.Connection_TYPE_STAGE
	case models.SourceTypeEventSource:
		return pb.Connection_TYPE_EVENT_SOURCE
	default:
		return pb.Connection_TYPE_UNKNOWN
	}
}

func findConnectionName(stages []models.Stage, sources []models.EventSource, connection models.StageConnection) (string, error) {
	switch connection.SourceType {
	case models.SourceTypeStage:
		for _, stage := range stages {
			if stage.ID == connection.SourceID {
				return stage.Name, nil
			}
		}

		return "", fmt.Errorf("stage %s not found", connection.SourceID)

	case models.SourceTypeEventSource:
		for _, s := range sources {
			if s.ID == connection.SourceID {
				return s.Name, nil
			}
		}

		return "", fmt.Errorf("event source %s not found", connection.ID)

	default:
		return "", errors.New("invalid type " + connection.SourceType)
	}
}

func findConnectionSourceID(canvas *models.Canvas, connection *pb.Connection) (*uuid.UUID, error) {
	switch connection.Type {
	case pb.Connection_TYPE_STAGE:
		stage, err := canvas.FindStageByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("stage %s not found", connection.Name)
		}

		return &stage.ID, nil

	case pb.Connection_TYPE_EVENT_SOURCE:
		eventSource, err := canvas.FindEventSourceByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("event source %s not found", connection.Name)
		}

		return &eventSource.ID, nil

	default:
		return nil, errors.New("invalid type")
	}
}

func protoToConnectionType(t pb.Connection_Type) string {
	switch t {
	case pb.Connection_TYPE_STAGE:
		return models.SourceTypeStage
	case pb.Connection_TYPE_EVENT_SOURCE:
		return models.SourceTypeEventSource
	default:
		return ""
	}
}

func serializeStage(
	stage models.Stage,
	connections []*pb.Connection,
	inputs []*pb.InputDefinition,
	outputs []*pb.OutputDefinition,
	inputMappings []*pb.InputMapping,
) (*pb.Stage, error) {
	executor, err := serializeExecutorSpec(stage.ExecutorSpec.Data())
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

func serializeExecutorSpec(executor models.ExecutorSpec) (*pb.ExecutorSpec, error) {
	switch executor.Type {
	case models.ExecutorSpecTypeHTTP:
		return &pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_HTTP,
			Http: &pb.ExecutorSpec_HTTP{
				Url:     executor.HTTP.URL,
				Headers: executor.HTTP.Headers,
				Payload: executor.HTTP.Payload,
				ResponsePolicy: &pb.ExecutorSpec_HTTPResponsePolicy{
					StatusCodes: executor.HTTP.ResponsePolicy.StatusCodes,
				},
			},
		}, nil
	case models.ExecutorSpecTypeSemaphore:
		return &pb.ExecutorSpec{
			Type: pb.ExecutorSpec_TYPE_SEMAPHORE,
			Semaphore: &pb.ExecutorSpec_Semaphore{
				OrganizationUrl: executor.Semaphore.OrganizationURL,
				ApiToken:        executor.Semaphore.APIToken,
				ProjectId:       executor.Semaphore.ProjectID,
				Branch:          executor.Semaphore.Branch,
				PipelineFile:    executor.Semaphore.PipelineFile,
				Parameters:      executor.Semaphore.Parameters,
				TaskId:          executor.Semaphore.TaskID,
			},
		}, nil

	default:
		return nil, fmt.Errorf("invalid executor spec type: %s", executor.Type)
	}
}

func serializeStages(stages []models.Stage, sources []models.EventSource) ([]*pb.Stage, error) {
	s := []*pb.Stage{}
	for _, stage := range stages {
		connections, err := models.ListConnectionsForStage(stage.ID)
		if err != nil {
			return nil, err
		}

		serialized, err := serializeConnections(stages, sources, connections)
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
