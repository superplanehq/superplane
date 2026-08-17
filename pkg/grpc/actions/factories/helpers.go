package factories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
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

// parseUserIDs validates that every id in `userIDs` is a well-formed UUID
// belonging to an active member of the organization, deduping repeats while
// preserving first-seen order. `label` is used in the invalid-argument error
// message (e.g. "assignee", "mentioned user") so callers get a precise error.
func parseUserIDs(tx *gorm.DB, organizationID uuid.UUID, userIDs []string, label string) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	seen := make(map[uuid.UUID]struct{}, len(userIDs))
	parsed := make([]uuid.UUID, 0, len(userIDs))
	for _, rawID := range userIDs {
		userID, err := uuid.Parse(rawID)
		if err != nil {
			return nil, invalidArgument(fmt.Sprintf("invalid %s id", label))
		}

		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}

		if _, err := models.FindActiveUserByIDInTransaction(tx, organizationID.String(), userID.String()); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, invalidArgument(fmt.Sprintf("%s %s not found", label, rawID))
			}
			return nil, err
		}

		parsed = append(parsed, userID)
	}

	return parsed, nil
}

func parseAssigneeIDs(tx *gorm.DB, organizationID uuid.UUID, assigneeIDs []string) ([]uuid.UUID, error) {
	return parseUserIDs(tx, organizationID, assigneeIDs, "assignee")
}

func listWorkOrderFilters(req *pb.ListWorkOrdersRequest) models.ListFactoryWorkOrdersFilters {
	filters := models.ListFactoryWorkOrdersFilters{
		Unassigned: req.Unassigned,
	}

	for _, state := range req.States {
		if mapped, ok := workOrderStateFromProto(state); ok {
			filters.States = append(filters.States, mapped)
		}
	}

	for _, result := range req.Results {
		if mapped, ok := workOrderResultFromProto(result); ok {
			filters.Results = append(filters.Results, mapped)
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
	mapped, ok := workOrderResultFromProto(result)
	if !ok {
		return "", invalidArgument("result must be completed, rejected, or failed")
	}
	return mapped, nil
}

func workOrderStateFromProto(state pb.WorkOrder_State) (string, bool) {
	switch state {
	case pb.WorkOrder_STATE_DRAFT:
		return models.FactoryWorkOrderStateDraft, true
	case pb.WorkOrder_STATE_OPEN:
		return models.FactoryWorkOrderStateOpen, true
	case pb.WorkOrder_STATE_CLOSED:
		return models.FactoryWorkOrderStateClosed, true
	}
	return "", false
}

func workOrderResultFromProto(result pb.WorkOrder_Result) (string, bool) {
	switch result {
	case pb.WorkOrder_RESULT_COMPLETED:
		return models.FactoryWorkOrderResultCompleted, true
	case pb.WorkOrder_RESULT_REJECTED:
		return models.FactoryWorkOrderResultRejected, true
	case pb.WorkOrder_RESULT_FAILED:
		return models.FactoryWorkOrderResultFailed, true
	}
	return "", false
}
