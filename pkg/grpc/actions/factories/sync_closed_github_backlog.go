package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

type gitHubIssueStateSource interface {
	Repository() string
	IsIssueClosed(ctx context.Context, number int) (bool, error)
}

func SyncClosedGitHubBacklog(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.SyncClosedGitHubBacklogRequest,
) (*pb.SyncClosedGitHubBacklogResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync closed GitHub issues")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	closedBy, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to sync closed GitHub issues")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync closed GitHub issues")
	}

	sources, err := connectedGitHubIssueSources(ctx, deps, db, factory)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync closed GitHub issues")
	}

	orders, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{
		States: []string{models.FactoryWorkOrderStateDraft},
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to sync closed GitHub issues")
	}

	closedCount := int32(0)
	failedCount := int32(0)
	for i := range orders {
		closed, failed := syncDraftOrderFromGitHub(ctx, db, orgID, factory, &orders[i], sources, closedBy)
		if closed {
			closedCount++
		}
		if failed {
			failedCount++
		}
	}

	return &pb.SyncClosedGitHubBacklogResponse{
		ClosedCount: closedCount,
		FailedCount: failedCount,
	}, nil
}

func connectedGitHubIssueSources(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
) (map[string]gitHubIssueStateSource, error) {
	intakes, err := factory.ListIntakes(db)
	if err != nil {
		return nil, err
	}

	sources := map[string]gitHubIssueStateSource{}
	var connectErr error
	for i := range intakes {
		intake := &intakes[i]
		if intake.Source != models.FactoryIntakeSourceGitHubIssues {
			continue
		}
		source, err := deps.itemSource(ctx, db, intake)
		if err != nil {
			connectErr = err
			continue
		}
		githubSource, ok := source.(gitHubIssueStateSource)
		if !ok {
			continue
		}
		repository := strings.ToLower(strings.TrimSpace(githubSource.Repository()))
		if repository == "" {
			continue
		}
		sources[repository] = githubSource
	}

	if len(sources) > 0 {
		return sources, nil
	}
	if connectErr != nil {
		return nil, connectErr
	}
	return nil, errIntakeNotConnected
}

func syncDraftOrderFromGitHub(
	ctx context.Context,
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	sources map[string]gitHubIssueStateSource,
	closedBy uuid.UUID,
) (closed bool, failed bool) {
	origin := order.Origin()
	if origin == nil {
		return false, false
	}

	ref, ok := models.ParseGitHubIssueOrigin(origin.URL)
	if !ok {
		return false, false
	}

	source := sources[strings.ToLower(ref.Repository)]
	if source == nil {
		return false, false
	}

	issueClosed, err := source.IsIssueClosed(ctx, ref.Number)
	if err != nil {
		return false, true
	}
	if !issueClosed {
		return false, false
	}

	if _, err := closeWorkOrderAsUser(db, orgID, factory, order, models.FactoryWorkOrderResultRejected, closedBy); err != nil {
		return false, true
	}
	return true, false
}
