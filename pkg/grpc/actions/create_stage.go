package actions

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"sort"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/encryptor"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateStage(ctx context.Context, encryptor encryptor.Encryptor, req *pb.CreateStageRequest) (*pb.CreateStageResponse, error) {
	err := ValidateUUIDs(req.CanvasId, req.RequesterId)
	if err != nil {
		return nil, err
	}

	canvas, err := models.FindCanvas(req.CanvasId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "canvas not found")
	}

	template, err := validateRunTemplate(ctx, encryptor, req.RunTemplate)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	connections, err := validateConnections(canvas, req.Connections)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	conditions, err := validateConditions(req.Conditions)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	tagUsage, err := validateTagUsageDefinition(req.Use, connections)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	err = canvas.CreateStage(req.Name, req.RequesterId, conditions, *template, connections, *tagUsage)
	if err != nil {
		if errors.Is(err, models.ErrNameAlreadyUsed) {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		return nil, err
	}

	stage, err := canvas.FindStageByName(req.Name)
	if err != nil {
		return nil, err
	}

	serialized, err := serializeStage(*stage, req.Connections)
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

func validateTagUsageDefinition(usage *pb.TagUsageDefinition, connections []models.StageConnection) (*models.StageTagUsageDefinition, error) {
	if usage == nil {
		return nil, fmt.Errorf("missing tag usage definition")
	}

	out := models.StageTagUsageDefinition{
		Tags: []models.StageTagDefinition{},
	}

	//
	// Check if all connections used are valid.
	//
	for _, from := range usage.From {
		if !slices.ContainsFunc(connections, func(connection models.StageConnection) bool {
			return connection.SourceName == from
		}) {
			return nil, fmt.Errorf("invalid tag: invalid from %s", from)
		}
	}

	out.From = usage.From
	if len(usage.Tags) == 0 {
		return nil, fmt.Errorf("tags must not be empty")
	}

	for _, t := range usage.Tags {
		if t.Name == "" || t.ValueFrom == "" {
			return nil, fmt.Errorf("invalid tag: no name or value defined")
		}

		out.Tags = append(out.Tags, models.StageTagDefinition{
			Name:      t.Name,
			ValueFrom: t.ValueFrom,
		})
	}

	return &out, nil
}

func validateRunTemplate(ctx context.Context, encryptor encryptor.Encryptor, in *pb.RunTemplate) (*models.RunTemplate, error) {
	if in == nil {
		return nil, fmt.Errorf("missing run template")
	}

	switch in.Type {
	case pb.RunTemplate_TYPE_SEMAPHORE:
		if in.Semaphore.OrganizationUrl == "" {
			return nil, fmt.Errorf("missing organization URL")
		}

		if in.Semaphore.ApiToken == "" {
			return nil, fmt.Errorf("missing API token")
		}

		if in.Semaphore.TaskId == "" {
			return nil, fmt.Errorf("only triggering tasks is supported for now")
		}

		token, err := encryptor.Encrypt(ctx, []byte(in.Semaphore.ApiToken), []byte(in.Semaphore.OrganizationUrl))
		if err != nil {
			return nil, fmt.Errorf("error encrypting API token: %v", err)
		}

		return &models.RunTemplate{
			Type: models.RunTemplateTypeSemaphore,
			Semaphore: &models.SemaphoreRunTemplate{
				OrganizationURL: in.Semaphore.OrganizationUrl,
				APIToken:        base64.StdEncoding.EncodeToString(token),
				ProjectID:       in.Semaphore.ProjectId,
				Branch:          in.Semaphore.Branch,
				PipelineFile:    in.Semaphore.PipelineFile,
				Parameters:      in.Semaphore.Parameters,
				TaskID:          in.Semaphore.TaskId,
			},
		}, nil

	default:
		return nil, errors.New("invalid run template type")
	}
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
		return "", errors.New("invalid type")
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

func serializeStage(stage models.Stage, connections []*pb.Connection) (*pb.Stage, error) {
	runTemplate, err := serializeRunTemplate(stage.RunTemplate.Data())
	if err != nil {
		return nil, err
	}

	conditions, err := serializeConditions(stage.Conditions)
	if err != nil {
		return nil, err
	}

	return &pb.Stage{
		Id:          stage.ID.String(),
		Name:        stage.Name,
		CanvasId:    stage.CanvasID.String(),
		CreatedAt:   timestamppb.New(*stage.CreatedAt),
		Conditions:  conditions,
		Connections: connections,
		Use:         serializeTagUsageDefinition(stage.Use.Data()),
		RunTemplate: runTemplate,
	}, nil
}

func serializeTagUsageDefinition(def models.StageTagUsageDefinition) *pb.TagUsageDefinition {
	return &pb.TagUsageDefinition{
		From: def.From,
		Tags: serializeTags(def.Tags),
	}
}

func serializeTags(tags []models.StageTagDefinition) []*pb.TagDefinition {
	out := []*pb.TagDefinition{}

	for _, t := range tags {
		out = append(out, &pb.TagDefinition{
			Name:      t.Name,
			ValueFrom: t.ValueFrom,
		})
	}

	return out
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

func serializeRunTemplate(runTemplate models.RunTemplate) (*pb.RunTemplate, error) {
	switch runTemplate.Type {
	case models.RunTemplateTypeSemaphore:
		return &pb.RunTemplate{
			Type: pb.RunTemplate_TYPE_SEMAPHORE,
			Semaphore: &pb.SemaphoreRunTemplate{
				OrganizationUrl: runTemplate.Semaphore.OrganizationURL,
				ProjectId:       runTemplate.Semaphore.ProjectID,
				Branch:          runTemplate.Semaphore.Branch,
				PipelineFile:    runTemplate.Semaphore.PipelineFile,
				Parameters:      runTemplate.Semaphore.Parameters,
				TaskId:          runTemplate.Semaphore.TaskID,
			},
		}, nil

	default:
		return nil, fmt.Errorf("invalid run template type: %s", runTemplate.Type)
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

		stage, err := serializeStage(stage, serialized)
		if err != nil {
			return nil, err
		}

		s = append(s, stage)
	}

	return s, nil
}
