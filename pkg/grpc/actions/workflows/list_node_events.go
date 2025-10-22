package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ListNodeEvents(ctx context.Context, registry *registry.Registry, workflowID uuid.UUID, nodeID string, limit uint32, before *timestamppb.Timestamp) (*pb.ListNodeEventsResponse, error) {
	limit = getLimit(limit)
	beforeTime := getBefore(before)

	//
	// List and count events
	//
	events, err := models.ListWorkflowEvents(workflowID, nodeID, int(limit), beforeTime)
	if err != nil {
		return nil, err
	}

	totalCount, err := models.CountWorkflowEvents(workflowID, nodeID)
	if err != nil {
		return nil, err
	}

	serialized, err := SerializeWorkflowEvents(events)
	if err != nil {
		return nil, err
	}

	return &pb.ListNodeEventsResponse{
		Events:        serialized,
		TotalCount:    uint32(totalCount),
		HasNextPage:   hasNextPage(len(events), int(limit), totalCount),
		LastTimestamp: getLastEventTimestamp(events),
	}, nil
}
