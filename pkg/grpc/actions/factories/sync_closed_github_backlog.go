package factories

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gitHubIssueStateSource interface {
	Repository() string
	IsIssueClosed(ctx context.Context, number int) (bool, error)
}

type gitHubIssueSourceIndex struct {
	byRepository map[string]gitHubIssueStateSource
	failedRepos  map[string]struct{}
	connectFails int
}

type gitHubIssueClosedLookup struct {
	closed bool
	err    error
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

	lookups := map[string]gitHubIssueClosedLookup{}
	closedCount := int32(0)
	failedCount := int32(sources.connectFails)
	for i := range orders {
		closed, failed := syncDraftOrderFromGitHub(ctx, db, orgID, factory, &orders[i], sources, lookups, closedBy)
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
) (gitHubIssueSourceIndex, error) {
	intakes, err := factory.ListIntakes(db)
	if err != nil {
		return gitHubIssueSourceIndex{}, err
	}

	index := gitHubIssueSourceIndex{
		byRepository: map[string]gitHubIssueStateSource{},
		failedRepos:  map[string]struct{}{},
	}
	var connectErr error
	for i := range intakes {
		intake := &intakes[i]
		if intake.Source != models.FactoryIntakeSourceGitHubIssues {
			continue
		}
		source, err := deps.itemSource(ctx, db, intake)
		if err != nil {
			connectErr = err
			recordFailedGitHubIntake(db, intake, &index)
			continue
		}
		githubSource, ok := source.(gitHubIssueStateSource)
		if !ok {
			continue
		}
		repository := normalizeGitHubRepository(githubSource.Repository())
		if repository == "" {
			continue
		}
		if _, exists := index.byRepository[repository]; exists {
			continue
		}
		index.byRepository[repository] = githubSource
		delete(index.failedRepos, repository)
	}

	if len(index.byRepository) > 0 {
		return index, nil
	}
	if connectErr != nil {
		return gitHubIssueSourceIndex{}, connectErr
	}
	return gitHubIssueSourceIndex{}, errIntakeNotConnected
}

func recordFailedGitHubIntake(db *gorm.DB, intake *models.FactoryIntake, index *gitHubIssueSourceIndex) {
	repository := gitHubIntakeRepository(db, intake)
	if repository == "" {
		index.connectFails++
		return
	}
	if _, exists := index.byRepository[repository]; exists {
		return
	}
	index.failedRepos[repository] = struct{}{}
}

func gitHubIntakeRepository(db *gorm.DB, intake *models.FactoryIntake) string {
	trigger, _, err := resolveLiveIntakeTrigger(db, intake)
	if err != nil || trigger == nil {
		return ""
	}
	repository, _ := trigger.Configuration["repository"].(string)
	return normalizeGitHubRepository(repository)
}

func normalizeGitHubRepository(repository string) string {
	return strings.ToLower(strings.TrimSpace(repository))
}

func syncDraftOrderFromGitHub(
	ctx context.Context,
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	sources gitHubIssueSourceIndex,
	lookups map[string]gitHubIssueClosedLookup,
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

	repository := normalizeGitHubRepository(ref.Repository)
	source := sources.byRepository[repository]
	if source == nil {
		if _, failedRepo := sources.failedRepos[repository]; failedRepo {
			return false, true
		}
		return false, false
	}

	issueClosed, cached, err := issueClosedFromSource(ctx, source, repository, ref.Number, lookups)
	if err != nil {
		return false, !cached
	}
	if !issueClosed {
		return false, false
	}

	return closeDraftWorkOrderIfCurrent(db, orgID, factory, order.ID, closedBy)
}

func issueClosedFromSource(
	ctx context.Context,
	source gitHubIssueStateSource,
	repository string,
	number int,
	lookups map[string]gitHubIssueClosedLookup,
) (closed bool, cached bool, err error) {
	key := repository + "#" + strconv.Itoa(number)
	if lookup, ok := lookups[key]; ok {
		return lookup.closed, true, lookup.err
	}

	closed, err = source.IsIssueClosed(ctx, number)
	lookups[key] = gitHubIssueClosedLookup{closed: closed, err: err}
	return closed, false, err
}

func closeDraftWorkOrderIfCurrent(
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	orderID uuid.UUID,
	closedBy uuid.UUID,
) (closed bool, failed bool) {
	var closedOrder *models.FactoryWorkOrder
	var fromState string
	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := factory.FindWorkOrder(tx.Clauses(clause.Locking{Strength: "UPDATE"}), orderID)
		if err != nil {
			return err
		}
		if current.State != models.FactoryWorkOrderStateDraft {
			return nil
		}
		fromState = current.State
		closedOrder, err = current.Close(tx, models.FactoryWorkOrderResultRejected, &closedBy)
		if err != nil {
			if errors.Is(err, models.ErrFactoryWorkOrderInvalidState) {
				closedOrder = nil
				return nil
			}
			return err
		}
		return nil
	})
	if err != nil {
		return false, true
	}
	if closedOrder == nil {
		return false, false
	}
	publishWorkOrderClosed(
		orgID,
		factory,
		closedOrder,
		closedBy,
		fromState,
		models.FactoryWorkOrderResultRejected,
		false,
	)
	return true, false
}
