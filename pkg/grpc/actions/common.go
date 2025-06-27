package actions

import (
	"errors"
	"fmt"
	"sort"

	uuid "github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	case models.StageExecutionResultFailed:
		return pb.Execution_RESULT_FAILED
	case models.StageExecutionResultPassed:
		return pb.Execution_RESULT_PASSED
	default:
		return pb.Execution_RESULT_UNKNOWN
	}
}

func FindConnectionSourceID(canvas *models.Canvas, connection *pb.Connection) (*uuid.UUID, error) {
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

	case pb.Connection_TYPE_CONNECTION_GROUP:
		connectionGroup, err := canvas.FindConnectionGroupByName(connection.Name)
		if err != nil {
			return nil, fmt.Errorf("connection group %s not found", connection.Name)
		}

		return &connectionGroup.ID, nil

	default:
		return nil, errors.New("invalid type")
	}
}

func ValidateConnections(canvas *models.Canvas, connections []*pb.Connection) ([]models.Connection, error) {
	cs := []models.Connection{}

	if len(connections) == 0 {
		return nil, fmt.Errorf("connections must not be empty")
	}

	for _, connection := range connections {
		sourceID, err := FindConnectionSourceID(canvas, connection)
		if err != nil {
			return nil, fmt.Errorf("invalid connection: %v", err)
		}

		filters, err := validateFilters(connection.Filters)
		if err != nil {
			return nil, err
		}

		cs = append(cs, models.Connection{
			SourceID:       *sourceID,
			SourceName:     connection.Name,
			SourceType:     protoToConnectionType(connection.Type),
			FilterOperator: protoToFilterOperator(connection.FilterOperator),
			Filters:        filters,
		})
	}

	return cs, nil
}

func validateFilters(in []*pb.Connection_Filter) ([]models.ConnectionFilter, error) {
	filters := []models.ConnectionFilter{}
	for i, f := range in {
		filter, err := validateFilter(f)
		if err != nil {
			return nil, fmt.Errorf("invalid filter [%d]: %v", i, err)
		}

		filters = append(filters, *filter)
	}

	return filters, nil
}

func validateFilter(filter *pb.Connection_Filter) (*models.ConnectionFilter, error) {
	switch filter.Type {
	case pb.Connection_FILTER_TYPE_DATA:
		return validateDataFilter(filter.Data)
	case pb.Connection_FILTER_TYPE_HEADER:
		return validateHeaderFilter(filter.Header)
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filter.Type)
	}
}

func validateDataFilter(filter *pb.Connection_DataFilter) (*models.ConnectionFilter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.ConnectionFilter{
		Type: models.FilterTypeData,
		Data: &models.DataFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func validateHeaderFilter(filter *pb.Connection_HeaderFilter) (*models.ConnectionFilter, error) {
	if filter == nil {
		return nil, fmt.Errorf("no filter provided")
	}

	if filter.Expression == "" {
		return nil, fmt.Errorf("expression is empty")
	}

	return &models.ConnectionFilter{
		Type: models.FilterTypeHeader,
		Header: &models.HeaderFilter{
			Expression: filter.Expression,
		},
	}, nil
}

func protoToConnectionType(t pb.Connection_Type) string {
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

func SerializeConnections(in []models.Connection) ([]*pb.Connection, error) {
	connections := []*pb.Connection{}

	for _, c := range in {
		filters, err := serializeFilters(c.Filters)
		if err != nil {
			return nil, fmt.Errorf("invalid filters: %v", err)
		}

		connections = append(connections, &pb.Connection{
			Type:           ConnectionTypeToProto(c.SourceType),
			Name:           c.SourceName,
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

func serializeFilters(in []models.ConnectionFilter) ([]*pb.Connection_Filter, error) {
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

func serializeFilter(in models.ConnectionFilter) (*pb.Connection_Filter, error) {
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
