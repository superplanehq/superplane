package actions

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"time"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	pbAuth "github.com/superplanehq/superplane/pkg/protos/authorization"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	integrationpb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		_, err := uuid.Parse(id)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid UUID: %s", id)
		}
	}

	return nil
}

func ExecutionResultToProto(result string) pb.Execution_Result {
	switch result {
	case models.ResultFailed:
		return pb.Execution_RESULT_FAILED
	case models.ResultPassed:
		return pb.Execution_RESULT_PASSED
	case models.ResultCancelled:
		return pb.Execution_RESULT_CANCELLED
	default:
		return pb.Execution_RESULT_UNKNOWN
	}
}

func ExecutionResultReasonToProto(reason string) pb.Execution_ResultReason {
	switch reason {
	case models.ResultReasonError:
		return pb.Execution_RESULT_REASON_ERROR
	case models.ResultReasonMissingOutputs:
		return pb.Execution_RESULT_REASON_MISSING_OUTPUTS
	case models.ResultReasonTimeout:
		return pb.Execution_RESULT_REASON_TIMEOUT
	case models.ResultReasonUser:
		return pb.Execution_RESULT_REASON_USER
	default:
		return pb.Execution_RESULT_REASON_OK
	}
}

func FindConnectionSourceID(canvasID string, connection *pb.Connection) (*uuid.UUID, error) {
	switch connection.Type {
	case pb.Connection_TYPE_STAGE:
		stage, err := models.FindStageByName(canvasID, connection.Name)
		if err != nil {
			return nil, fmt.Errorf("stage %s not found", connection.Name)
		}

		return &stage.ID, nil

	case pb.Connection_TYPE_EVENT_SOURCE:
		eventSource, err := models.FindExternalEventSourceByName(canvasID, connection.Name)
		if err != nil {
			return nil, fmt.Errorf("event source %s not found", connection.Name)
		}

		return &eventSource.ID, nil

	case pb.Connection_TYPE_CONNECTION_GROUP:
		connectionGroup, err := models.FindConnectionGroupByName(canvasID, connection.Name)
		if err != nil {
			return nil, fmt.Errorf("connection group %s not found", connection.Name)
		}

		return &connectionGroup.ID, nil

	default:
		return nil, errors.New("invalid type")
	}
}

func ValidateConnections(canvasID string, connections []*pb.Connection) ([]models.Connection, error) {
	cs := []models.Connection{}

	if len(connections) == 0 {
		return nil, fmt.Errorf("connections must not be empty")
	}

	for _, connection := range connections {
		sourceID, err := FindConnectionSourceID(canvasID, connection)
		if err != nil {
			return nil, fmt.Errorf("invalid connection: %v", err)
		}

		filters, err := ValidateFilters(connection.Filters)
		if err != nil {
			return nil, err
		}

		cs = append(cs, models.Connection{
			SourceID:       *sourceID,
			SourceName:     connection.Name,
			SourceType:     ProtoToConnectionType(connection.Type),
			FilterOperator: ProtoToFilterOperator(connection.FilterOperator),
			Filters:        filters,
		})
	}

	return cs, nil
}

func ValidateFilters(in []*pb.Filter) ([]models.Filter, error) {
	filters := []models.Filter{}
	for i, f := range in {
		filter, err := validateFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter [%d]: %v", i, err)
		}

		filters = append(filters, *filter)
	}

	return filters, nil
}

func validateFilter(filter *pb.Filter) (*models.Filter, error) {
	switch filter.Type {
	case pb.FilterType_FILTER_TYPE_DATA:
		return validateDataFilter(filter.Data)
	case pb.FilterType_FILTER_TYPE_HEADER:
		return validateHeaderFilter(filter.Header)
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filter.Type)
	}
}

func validateDataFilter(filter *pb.DataFilter) (*models.Filter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.Filter{
		Type: models.FilterTypeData,
		Data: &models.DataFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func validateHeaderFilter(filter *pb.HeaderFilter) (*models.Filter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.Filter{
		Type: models.FilterTypeHeader,
		Header: &models.HeaderFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func ProtoToConnectionType(t pb.Connection_Type) string {
	switch t {
	case pb.Connection_TYPE_STAGE:
		return models.SourceTypeStage
	case pb.Connection_TYPE_EVENT_SOURCE:
		return models.SourceTypeEventSource
	case pb.Connection_TYPE_CONNECTION_GROUP:
		return models.SourceTypeConnectionGroup
	default:
		return ""
	}
}

func ProtoToFilterOperator(in pb.FilterOperator) string {
	switch in {
	case pb.FilterOperator_FILTER_OPERATOR_OR:
		return models.FilterOperatorOr
	default:
		return models.FilterOperatorAnd
	}
}

func FilterOperatorToProto(in string) pb.FilterOperator {
	switch in {
	case models.FilterOperatorOr:
		return pb.FilterOperator_FILTER_OPERATOR_OR
	default:
		return pb.FilterOperator_FILTER_OPERATOR_AND
	}
}

func SerializeConnections(in []models.Connection) ([]*pb.Connection, error) {
	connections := []*pb.Connection{}

	for _, c := range in {
		filters, err := SerializeFilters(c.Filters)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %v", err)
		}

		connections = append(connections, &pb.Connection{
			Type:           ConnectionTypeToProto(c.SourceType),
			Name:           c.SourceName,
			FilterOperator: FilterOperatorToProto(c.FilterOperator),
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

func SerializeFilters(in []models.Filter) ([]*pb.Filter, error) {
	filters := []*pb.Filter{}

	for _, f := range in {
		filter, err := SerializeFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %v", err)
		}

		filters = append(filters, filter)
	}

	return filters, nil
}

func SerializeFilter(in models.Filter) (*pb.Filter, error) {
	switch in.Type {
	case models.FilterTypeData:
		return &pb.Filter{
			Type: pb.FilterType_FILTER_TYPE_DATA,
			Data: &pb.DataFilter{
				Expression: in.Data.Expression,
			},
		}, nil
	case models.FilterTypeHeader:
		return &pb.Filter{
			Type: pb.FilterType_FILTER_TYPE_HEADER,
			Header: &pb.HeaderFilter{
				Expression: in.Header.Expression,
			},
		}, nil
	default:
		return nil, fmt.Errorf("invalid filter type: %s", in.Type)
	}
}

func ConnectionTypeToProto(t string) pb.Connection_Type {
	switch t {
	case models.SourceTypeStage:
		return pb.Connection_TYPE_STAGE
	case models.SourceTypeEventSource:
		return pb.Connection_TYPE_EVENT_SOURCE
	case models.SourceTypeConnectionGroup:
		return pb.Connection_TYPE_CONNECTION_GROUP
	default:
		return pb.Connection_TYPE_UNKNOWN
	}
}

func RejectionReasonToProto(reason string) pb.EventRejection_RejectionReason {
	switch reason {
	case models.EventRejectionReasonFiltered:
		return pb.EventRejection_REJECTION_REASON_FILTERED
	case models.EventRejectionReasonError:
		return pb.EventRejection_REJECTION_REASON_ERROR
	default:
		return pb.EventRejection_REJECTION_REASON_UNKNOWN
	}
}

func ProtoToDomainType(domainType pbAuth.DomainType) (string, error) {
	switch domainType {
	case pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION:
		return models.DomainTypeOrganization, nil
	case pbAuth.DomainType_DOMAIN_TYPE_CANVAS:
		return models.DomainTypeCanvas, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid domain type")
	}
}

func DomainTypeToProto(domainType string) pbAuth.DomainType {
	switch domainType {
	case models.DomainTypeCanvas:
		return pbAuth.DomainType_DOMAIN_TYPE_CANVAS
	case models.DomainTypeOrganization:
		return pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION
	default:
		return pbAuth.DomainType_DOMAIN_TYPE_UNSPECIFIED
	}
}

func ValidateIntegration(canvas *models.Canvas, integrationRef *integrationpb.IntegrationRef) (*models.Integration, error) {
	if integrationRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "integration name is required")
	}

	//
	// If the integration used is on the organization level, we need to find it there.
	//
	if integrationRef.DomainType == pbAuth.DomainType_DOMAIN_TYPE_ORGANIZATION {
		integration, err := models.FindIntegrationByName(models.DomainTypeOrganization, canvas.OrganizationID, integrationRef.Name)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationRef.Name)
		}

		return integration, nil
	}

	//
	// Otherwise, we look for it on the canvas level.
	//
	integration, err := models.FindIntegrationByName(models.DomainTypeCanvas, canvas.ID, integrationRef.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "integration %s not found", integrationRef.Name)
	}

	return integration, nil
}

func ValidateResource(ctx context.Context, registry *registry.Registry, integration *models.Integration, resourceRef *integrationpb.ResourceRef) (integrations.Resource, error) {
	if resourceRef == nil {
		return nil, status.Error(codes.InvalidArgument, "resource reference is required")
	}

	if resourceRef.Type == "" || resourceRef.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "resource type and name are required")
	}

	//
	// If resource record does not exist yet, we need to go to the integration to find it.
	//
	integrationImpl, err := registry.NewResourceManager(ctx, integration)
	if err != nil {
		return nil, fmt.Errorf("error starting integration implementation: %v", err)
	}

	resource, err := integrationImpl.Get(resourceRef.Type, resourceRef.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s %s not found: %v", resourceRef.Type, resourceRef.Name, err)
	}

	return resource, nil
}

func GetDomainForSecret(domainTypeForResource string, domainIdForResource *uuid.UUID, domainType pbAuth.DomainType) (string, *uuid.UUID, error) {
	domainTypeForSecret, err := ProtoToDomainType(domainType)
	if err != nil {
		domainTypeForSecret = domainTypeForResource
	}

	//
	// If an organization-level resource is being created,
	// the secret must be on the organization level as well.
	//
	if domainTypeForResource == models.DomainTypeOrganization {
		if domainTypeForSecret != models.DomainTypeOrganization {
			return "", nil, fmt.Errorf("integration on organization level must use organization-level secret")
		}

		return domainTypeForSecret, domainIdForResource, nil
	}

	//
	// If a canvas-level resource is being created and a canvas-level secret is being used,
	// we can just re-use the same domain type and ID for the resource.
	//
	if domainTypeForSecret == models.DomainTypeCanvas {
		return domainTypeForSecret, domainIdForResource, nil
	}

	//
	// If a canvas-level resource is being created and is using a org-level secret,
	// we need to find the organization ID for the canvas where the resource is being created.
	//
	canvas, err := models.FindUnscopedCanvasByID(domainIdForResource.String())
	if err != nil {
		return "", nil, fmt.Errorf("canvas not found")
	}

	return models.DomainTypeOrganization, &canvas.OrganizationID, nil
}

func SerializeEvent(in models.Event) (*pb.Event, error) {
	event := &pb.Event{
		Id:          in.ID.String(),
		SourceId:    in.SourceID.String(),
		SourceName:  in.SourceName,
		SourceType:  EventSourceTypeToProto(in.SourceType),
		Type:        in.Type,
		State:       EventStateToProto(in.State),
		StateReason: EventStateReasonToProto(in.StateReason),
		ReceivedAt:  timestamppb.New(*in.ReceivedAt),
	}

	if len(in.Raw) > 0 {
		data, err := in.GetData()
		if err != nil {
			return nil, err
		}

		event.Raw, err = structpb.NewStruct(data)

		if err != nil {
			return nil, err
		}
	}

	if len(in.Headers) > 0 {
		headers, err := in.GetHeaders()
		if err != nil {
			return nil, err
		}

		event.Headers, err = structpb.NewStruct(headers)

		if err != nil {
			return nil, err
		}
	}

	return event, nil
}

func EventSourceTypeToProto(sourceType string) pb.EventSourceType {
	switch sourceType {
	case models.SourceTypeEventSource:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE
	case models.SourceTypeStage:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_STAGE
	case models.SourceTypeConnectionGroup:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_CONNECTION_GROUP
	default:
		return pb.EventSourceType_EVENT_SOURCE_TYPE_UNKNOWN
	}
}

func ProtoToEventSourceType(sourceType pb.EventSourceType) string {
	switch sourceType {
	case pb.EventSourceType_EVENT_SOURCE_TYPE_EVENT_SOURCE:
		return models.SourceTypeEventSource
	case pb.EventSourceType_EVENT_SOURCE_TYPE_STAGE:
		return models.SourceTypeStage
	case pb.EventSourceType_EVENT_SOURCE_TYPE_CONNECTION_GROUP:
		return models.SourceTypeConnectionGroup
	default:
		return ""
	}
}

func EventStateToProto(state string) pb.Event_State {
	switch state {
	case models.EventStatePending:
		return pb.Event_STATE_PENDING
	case models.EventStateProcessed:
		return pb.Event_STATE_PROCESSED
	case models.EventStateRejected:
		return pb.Event_STATE_REJECTED
	default:
		return pb.Event_STATE_UNKNOWN
	}
}

func EventStateReasonToProto(stateReason string) pb.Event_StateReason {
	switch stateReason {
	case models.EventStateReasonError:
		return pb.Event_STATE_REASON_ERROR
	case models.EventStateReasonFiltered:
		return pb.Event_STATE_REASON_FILTERED
	case models.EventStateReasonOk:
		return pb.Event_STATE_REASON_OK
	default:
		return pb.Event_STATE_REASON_UNKNOWN
	}
}

func SerializeStageEvent(in models.StageEvent) (*pb.StageEvent, error) {
	e := pb.StageEvent{
		Id:          in.ID.String(),
		State:       StageEventStateToProto(in.State),
		StateReason: StageEventStateReasonToProto(in.StateReason),
		CreatedAt:   timestamppb.New(*in.CreatedAt),
		Approvals:   []*pb.StageEventApproval{},
		Inputs:      []*pb.KeyValuePair{},
		Name:        in.Name,
	}

	if in.DiscardedBy != nil {
		e.DiscardedBy = in.DiscardedBy.String()
	}
	if in.DiscardedAt != nil {
		e.DiscardedAt = timestamppb.New(*in.DiscardedAt)
	}

	//
	// Add inputs
	//
	for k, v := range in.Inputs.Data() {
		e.Inputs = append(e.Inputs, &pb.KeyValuePair{Name: k, Value: v.(string)})
	}

	//
	// Add approvals
	//
	approvals, err := in.FindApprovals()
	if err != nil {
		return nil, err
	}

	for _, approval := range approvals {
		e.Approvals = append(e.Approvals, &pb.StageEventApproval{
			ApprovedBy: approval.ApprovedBy.String(),
			ApprovedAt: timestamppb.New(*approval.ApprovedAt),
		})
	}

	if in.Event != nil {
		serializedTriggerEvent, err := SerializeEvent(*in.Event)
		if err != nil {
			return nil, err
		}
		e.TriggerEvent = serializedTriggerEvent
	}

	return &e, nil
}

func ProtoToScheduleType(scheduleType pb.EventSource_Schedule_Type) (string, error) {
	switch scheduleType {
	case pb.EventSource_Schedule_TYPE_HOURLY:
		return models.ScheduleTypeHourly, nil
	case pb.EventSource_Schedule_TYPE_DAILY:
		return models.ScheduleTypeDaily, nil
	case pb.EventSource_Schedule_TYPE_WEEKLY:
		return models.ScheduleTypeWeekly, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid schedule type")
	}
}

func ScheduleTypeToProto(scheduleType string) pb.EventSource_Schedule_Type {
	switch scheduleType {
	case models.ScheduleTypeHourly:
		return pb.EventSource_Schedule_TYPE_HOURLY
	case models.ScheduleTypeDaily:
		return pb.EventSource_Schedule_TYPE_DAILY
	case models.ScheduleTypeWeekly:
		return pb.EventSource_Schedule_TYPE_WEEKLY
	default:
		return pb.EventSource_Schedule_TYPE_UNKNOWN
	}
}

func ProtoToWeekDay(weekDay pb.EventSource_Schedule_WeekDay) (string, error) {
	switch weekDay {
	case pb.EventSource_Schedule_WEEK_DAY_MONDAY:
		return models.WeekDayMonday, nil
	case pb.EventSource_Schedule_WEEK_DAY_TUESDAY:
		return models.WeekDayTuesday, nil
	case pb.EventSource_Schedule_WEEK_DAY_WEDNESDAY:
		return models.WeekDayWednesday, nil
	case pb.EventSource_Schedule_WEEK_DAY_THURSDAY:
		return models.WeekDayThursday, nil
	case pb.EventSource_Schedule_WEEK_DAY_FRIDAY:
		return models.WeekDayFriday, nil
	case pb.EventSource_Schedule_WEEK_DAY_SATURDAY:
		return models.WeekDaySaturday, nil
	case pb.EventSource_Schedule_WEEK_DAY_SUNDAY:
		return models.WeekDaySunday, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid week day")
	}
}

func WeekDayToProto(weekDay string) pb.EventSource_Schedule_WeekDay {
	switch weekDay {
	case models.WeekDayMonday:
		return pb.EventSource_Schedule_WEEK_DAY_MONDAY
	case models.WeekDayTuesday:
		return pb.EventSource_Schedule_WEEK_DAY_TUESDAY
	case models.WeekDayWednesday:
		return pb.EventSource_Schedule_WEEK_DAY_WEDNESDAY
	case models.WeekDayThursday:
		return pb.EventSource_Schedule_WEEK_DAY_THURSDAY
	case models.WeekDayFriday:
		return pb.EventSource_Schedule_WEEK_DAY_FRIDAY
	case models.WeekDaySaturday:
		return pb.EventSource_Schedule_WEEK_DAY_SATURDAY
	case models.WeekDaySunday:
		return pb.EventSource_Schedule_WEEK_DAY_SUNDAY
	default:
		return pb.EventSource_Schedule_WEEK_DAY_UNKNOWN
	}
}

func ValidateTime(timeStr string) error {
	if timeStr == "" {
		return status.Error(codes.InvalidArgument, "time is required")
	}

	_, err := time.Parse("15:04", timeStr)
	if err != nil {
		return status.Error(codes.InvalidArgument, "time must be in HH:MM format (24-hour)")
	}

	return nil
}

func ValidateCronExpression(expression string) error {
	if expression == "" {
		return status.Error(codes.InvalidArgument, "cron expression is required")
	}

	// Basic cron expression validation (5 or 6 fields)
	cronRegex := regexp.MustCompile(`^(\S+\s+){4}\S+(\s+\S+)?$`)
	if !cronRegex.MatchString(expression) {
		return status.Error(codes.InvalidArgument, "invalid cron expression format")
	}

	return nil
}

func StageEventStateToProto(state string) pb.StageEvent_State {
	switch state {
	case models.StageEventStatePending:
		return pb.StageEvent_STATE_PENDING
	case models.StageEventStateWaiting:
		return pb.StageEvent_STATE_WAITING
	case models.StageEventStateProcessed:
		return pb.StageEvent_STATE_PROCESSED
	case models.StageEventStateDiscarded:
		return pb.StageEvent_STATE_DISCARDED
	default:
		return pb.StageEvent_STATE_UNKNOWN
	}
}

func StageEventStateReasonToProto(stateReason string) pb.StageEvent_StateReason {
	switch stateReason {
	case models.StageEventStateReasonApproval:
		return pb.StageEvent_STATE_REASON_APPROVAL
	case models.StageEventStateReasonTimeWindow:
		return pb.StageEvent_STATE_REASON_TIME_WINDOW
	default:
		return pb.StageEvent_STATE_REASON_UNKNOWN
	}
}
