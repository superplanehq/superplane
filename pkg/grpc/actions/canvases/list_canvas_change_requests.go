package canvases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func ListCanvasChangeRequests(
	ctx context.Context,
	organizationID string,
	canvasID string,
	limit uint32,
	before *timestamppb.Timestamp,
	statusFilter string,
	onlyMine bool,
	query string,
) (*pb.ListCanvasChangeRequestsResponse, error) {
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas id: %v", err)
	}

	canvas, err := models.FindCanvas(uuid.MustParse(organizationID), canvasUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}

	resolvedStatuses, err := resolveChangeRequestStatusFilter(statusFilter)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid status filter: %v", err)
	}

	limit = getLimit(limit)
	beforeTime := getBefore(before)
	listOptions := models.CanvasChangeRequestListOptions{
		Limit:    int(limit),
		Before:   beforeTime,
		Statuses: resolvedStatuses,
		Query:    strings.TrimSpace(query),
	}
	if onlyMine {
		listOptions.OwnerID = &userUUID
	}

	var requests []models.CanvasChangeRequest
	var totalCount int64
	err = database.Conn().Transaction(func(tx *gorm.DB) error {
		items, listErr := models.ListCanvasChangeRequestsFilteredInTransaction(tx, canvas.ID, listOptions)
		if listErr != nil {
			return listErr
		}
		requests = items

		countOptions := listOptions
		countOptions.Limit = 0
		countOptions.Before = nil
		count, countErr := models.CountCanvasChangeRequestsFilteredInTransaction(tx, canvas.ID, countOptions)
		if countErr != nil {
			return countErr
		}
		totalCount = count
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list canvas change requests: %v", err)
	}

	protoRequests := make([]*pb.CanvasChangeRequest, 0, len(requests))
	for i := range requests {
		request := requests[i]
		version, versionErr := models.FindCanvasVersion(request.WorkflowID, request.VersionID)
		if versionErr != nil {
			return nil, status.Errorf(codes.Internal, "failed to load change request version: %v", versionErr)
		}

		protoRequests = append(protoRequests, SerializeCanvasChangeRequest(&request, version, organizationID))
	}

	return &pb.ListCanvasChangeRequestsResponse{
		ChangeRequests: protoRequests,
		TotalCount:     uint32(totalCount),
		HasNextPage:    hasNextPage(len(requests), int(limit), totalCount),
		LastTimestamp:  getLastCanvasChangeRequestTimestamp(requests),
	}, nil
}

func resolveChangeRequestStatusFilter(filter string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return nil, nil
	case "open":
		return []string{models.CanvasChangeRequestStatusOpen}, nil
	case "merged", "published":
		return []string{models.CanvasChangeRequestStatusPublished}, nil
	default:
		return nil, fmt.Errorf("unsupported filter %q", filter)
	}
}

func getLastCanvasChangeRequestTimestamp(requests []models.CanvasChangeRequest) *timestamppb.Timestamp {
	if len(requests) == 0 {
		return nil
	}

	lastRequest := requests[len(requests)-1]
	if lastRequest.CreatedAt == nil {
		return nil
	}

	return timestamppb.New(*lastRequest.CreatedAt)
}
