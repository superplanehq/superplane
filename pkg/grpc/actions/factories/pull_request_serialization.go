package factories

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/models"
	canvasespb "github.com/superplanehq/superplane/pkg/protos/canvases"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
)

func serializeFactoryPullRequests(
	tx *gorm.DB,
	pullRequests []models.FactoryPullRequest,
) ([]*pb.FactoryPullRequest, error) {
	if len(pullRequests) == 0 {
		return nil, nil
	}

	pullRequestIDs := make([]uuid.UUID, len(pullRequests))
	workOrderIDs := make([]uuid.UUID, 0, len(pullRequests))
	seenWorkOrders := map[uuid.UUID]bool{}
	for i := range pullRequests {
		pullRequestIDs[i] = pullRequests[i].ID
		if seenWorkOrders[pullRequests[i].WorkOrderID] {
			continue
		}
		seenWorkOrders[pullRequests[i].WorkOrderID] = true
		workOrderIDs = append(workOrderIDs, pullRequests[i].WorkOrderID)
	}

	numbers, err := workOrderNumbersByID(tx, workOrderIDs)
	if err != nil {
		return nil, err
	}

	runsByPullRequest, err := models.ListPullRequestRuns(tx, pullRequestIDs)
	if err != nil {
		return nil, err
	}

	serialized := make([]*pb.FactoryPullRequest, 0, len(pullRequests))
	for i := range pullRequests {
		serialized = append(serialized, serializeFactoryPullRequest(
			&pullRequests[i],
			numbers[pullRequests[i].WorkOrderID],
			runsByPullRequest[pullRequests[i].ID],
		))
	}
	return serialized, nil
}

func serializeFactoryPullRequest(
	pullRequest *models.FactoryPullRequest,
	workOrderNumber int64,
	runs []models.FactoryPullRequestLinkedRun,
) *pb.FactoryPullRequest {
	serialized := &pb.FactoryPullRequest{
		Id:              pullRequest.ID.String(),
		FactoryId:       pullRequest.FactoryID.String(),
		WorkOrderId:     pullRequest.WorkOrderID.String(),
		WorkOrderNumber: workOrderNumber,
		Provider:        pullRequestProviderToProto(pullRequest.Provider),
		Repository:      pullRequest.Repository,
		Number:          pullRequest.Number,
		Url:             pullRequest.URL,
		Title:           pullRequest.Title,
		State:           pullRequestStateToProto(pullRequest.State),
		CreatedAt:       timestamppb.New(pullRequest.CreatedAt),
		UpdatedAt:       timestamppb.New(pullRequest.UpdatedAt),
		Runs:            serializePullRequestRuns(runs),
	}
	if pullRequest.ExternalID != nil {
		serialized.ExternalId = *pullRequest.ExternalID
	}
	if pullRequest.MergedAt != nil {
		serialized.MergedAt = timestamppb.New(*pullRequest.MergedAt)
	}
	if pullRequest.ClosedAt != nil {
		serialized.ClosedAt = timestamppb.New(*pullRequest.ClosedAt)
	}
	return serialized
}

func serializePullRequestRuns(runs []models.FactoryPullRequestLinkedRun) []*canvasespb.CanvasRunRef {
	refs := make([]*canvasespb.CanvasRunRef, 0, len(runs))
	for _, linked := range runs {
		ref := canvases.SerializeCanvasRunRef(linked.Run)
		ref.Description = linked.Description
		refs = append(refs, ref)
	}
	return refs
}

func workOrderNumbersByID(tx *gorm.DB, workOrderIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	result := map[uuid.UUID]int64{}
	if len(workOrderIDs) == 0 {
		return result, nil
	}

	var orders []models.FactoryWorkOrder
	err := tx.Select("id", "number").Where("id IN ?", workOrderIDs).Find(&orders).Error
	if err != nil {
		return nil, err
	}
	for _, order := range orders {
		result[order.ID] = order.Number
	}
	return result, nil
}

func pullRequestProviderToProto(provider string) pb.FactoryPullRequest_Provider {
	switch provider {
	case models.FactoryPullRequestProviderBitbucket:
		return pb.FactoryPullRequest_PROVIDER_BITBUCKET
	case models.FactoryPullRequestProviderGitHub:
		return pb.FactoryPullRequest_PROVIDER_GITHUB
	default:
		return pb.FactoryPullRequest_PROVIDER_UNSPECIFIED
	}
}

func pullRequestProviderFromProto(provider pb.FactoryPullRequest_Provider) string {
	switch provider {
	case pb.FactoryPullRequest_PROVIDER_BITBUCKET:
		return models.FactoryPullRequestProviderBitbucket
	case pb.FactoryPullRequest_PROVIDER_GITHUB:
		return models.FactoryPullRequestProviderGitHub
	default:
		return ""
	}
}

func pullRequestStateToProto(state string) pb.FactoryPullRequest_State {
	switch state {
	case models.FactoryPullRequestStateOpen:
		return pb.FactoryPullRequest_STATE_OPEN
	case models.FactoryPullRequestStateDraft:
		return pb.FactoryPullRequest_STATE_DRAFT
	case models.FactoryPullRequestStateClosed:
		return pb.FactoryPullRequest_STATE_CLOSED
	case models.FactoryPullRequestStateMerged:
		return pb.FactoryPullRequest_STATE_MERGED
	default:
		return pb.FactoryPullRequest_STATE_UNSPECIFIED
	}
}

func pullRequestStateFromProto(state pb.FactoryPullRequest_State) string {
	switch state {
	case pb.FactoryPullRequest_STATE_OPEN:
		return models.FactoryPullRequestStateOpen
	case pb.FactoryPullRequest_STATE_DRAFT:
		return models.FactoryPullRequestStateDraft
	case pb.FactoryPullRequest_STATE_CLOSED:
		return models.FactoryPullRequestStateClosed
	case pb.FactoryPullRequest_STATE_MERGED:
		return models.FactoryPullRequestStateMerged
	default:
		return ""
	}
}

func timestampPointer(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	parsed := value.AsTime()
	return &parsed
}
