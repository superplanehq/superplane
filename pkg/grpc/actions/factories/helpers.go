package factories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"
)

func parseOrganizationID(organizationID string) (uuid.UUID, error) {
	orgID, err := uuid.Parse(organizationID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid organization id")
	}

	return orgID, nil
}

func parseFactoryID(factoryID string) (uuid.UUID, error) {
	id, err := uuid.Parse(factoryID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid factory id")
	}

	return id, nil
}

func parseOrderID(orderID string) (uuid.UUID, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid work order id")
	}

	return id, nil
}

func parseAgentID(agentID string) (uuid.UUID, error) {
	id, err := uuid.Parse(agentID)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid agent id")
	}

	return id, nil
}

func parseAssignmentID(id string) (uuid.UUID, error) {
	assignmentID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, invalidArgument("invalid assignment id")
	}

	return assignmentID, nil
}

func loadFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*models.Factory, error) {
	factory, err := models.FindFactory(tx, organizationID, factoryID)
	if err != nil {
		return nil, err
	}

	return factory, nil
}

func resolveIntegrationRef(tx *gorm.DB, organizationID uuid.UUID, ref *pb.IntegrationRef) (uuid.UUID, error) {
	if ref == nil {
		return uuid.Nil, invalidArgument("integration is required")
	}

	if ref.Id != nil && *ref.Id != "" {
		integrationID, err := uuid.Parse(*ref.Id)
		if err != nil {
			return uuid.Nil, invalidArgument("invalid integration id")
		}

		if _, err := models.FindIntegrationInTransaction(tx, organizationID, integrationID); err != nil {
			return uuid.Nil, err
		}

		return integrationID, nil
	}

	if ref.Name != nil && *ref.Name != "" {
		integration, err := models.FindIntegrationByName(tx, organizationID, *ref.Name)
		if err != nil {
			return uuid.Nil, err
		}

		return integration.ID, nil
	}

	return uuid.Nil, invalidArgument("integration id or name is required")
}

func structToMap(value *structpb.Struct) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	return value.AsMap()
}

func parseAssigneeIDs(tx *gorm.DB, organizationID uuid.UUID, assigneeIDs []string) ([]uuid.UUID, error) {
	if len(assigneeIDs) == 0 {
		return nil, nil
	}

	parsed := make([]uuid.UUID, 0, len(assigneeIDs))
	for _, assigneeID := range assigneeIDs {
		userID, err := uuid.Parse(assigneeID)
		if err != nil {
			return nil, invalidArgument("invalid assignee id")
		}

		if _, err := models.FindActiveUserByIDInTransaction(tx, organizationID.String(), userID.String()); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, invalidArgument(fmt.Sprintf("assignee %s not found", assigneeID))
			}
			return nil, err
		}

		parsed = append(parsed, userID)
	}

	return parsed, nil
}

func listWorkOrderFilters(req *pb.ListWorkOrdersRequest) models.ListFactoryWorkOrdersFilters {
	filters := models.ListFactoryWorkOrdersFilters{
		Unassigned: req.Unassigned,
	}

	for _, state := range req.States {
		switch state {
		case pb.WorkOrder_STATE_OPEN:
			filters.States = append(filters.States, models.FactoryWorkOrderStateOpen)
		case pb.WorkOrder_STATE_CLOSED:
			filters.States = append(filters.States, models.FactoryWorkOrderStateClosed)
		}
	}

	for _, result := range req.Results {
		switch result {
		case pb.WorkOrder_RESULT_COMPLETED:
			filters.Results = append(filters.Results, models.FactoryWorkOrderResultCompleted)
		case pb.WorkOrder_RESULT_REJECTED:
			filters.Results = append(filters.Results, models.FactoryWorkOrderResultRejected)
		}
	}

	for _, assigneeID := range req.AssigneeIds {
		userID, err := uuid.Parse(assigneeID)
		if err != nil {
			continue
		}
		filters.AssigneeIDs = append(filters.AssigneeIDs, userID)
	}

	return filters
}

func closeWorkOrderResult(result pb.WorkOrder_Result) (string, error) {
	switch result {
	case pb.WorkOrder_RESULT_COMPLETED:
		return models.FactoryWorkOrderResultCompleted, nil
	case pb.WorkOrder_RESULT_REJECTED:
		return models.FactoryWorkOrderResultRejected, nil
	default:
		return "", invalidArgument("result must be completed or rejected")
	}
}

func parseOrderIDs(orderIDs []string) ([]uuid.UUID, error) {
	if len(orderIDs) == 0 {
		return nil, invalidArgument("order_ids is required")
	}

	parsed := make([]uuid.UUID, 0, len(orderIDs))
	for _, orderID := range orderIDs {
		id, err := parseOrderID(orderID)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, id)
	}
	return parsed, nil
}
