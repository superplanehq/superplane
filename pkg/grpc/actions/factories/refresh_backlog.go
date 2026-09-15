package factories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type intakeAvailabilitySources struct {
	sources           []intakeItemAvailabilitySource
	failedSourceCount int32
}

type intakeItemAvailability struct {
	available bool
	err       error
}

type intakeItemAvailabilityKey struct {
	scope string
	id    string
}

func RefreshBacklog(
	ctx context.Context,
	deps IntakeDependencies,
	organizationID string,
	req *pb.RefreshBacklogRequest,
) (*pb.RefreshBacklogResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to refresh backlog")
	}

	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	archivedBy, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to refresh backlog")
	}

	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to refresh backlog")
	}

	sources, err := connectedIntakeAvailabilitySources(ctx, deps, db, factory)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to refresh backlog")
	}

	orders, err := factory.ListWorkOrders(db, models.ListFactoryWorkOrdersFilters{
		States: []string{models.FactoryWorkOrderStateDraft},
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to refresh backlog")
	}

	lookups := map[intakeItemAvailabilityKey]intakeItemAvailability{}
	archivedCount := int32(0)
	failedItemCount := int32(0)
	for i := range orders {
		archived, failed := refreshBacklogOrder(ctx, db, orgID, factory, &orders[i], sources.sources, lookups, archivedBy)
		if archived {
			archivedCount++
		}
		if failed {
			failedItemCount++
		}
	}

	return &pb.RefreshBacklogResponse{
		ArchivedCount:     archivedCount,
		FailedItemCount:   failedItemCount,
		FailedSourceCount: sources.failedSourceCount,
	}, nil
}

func connectedIntakeAvailabilitySources(
	ctx context.Context,
	deps IntakeDependencies,
	db *gorm.DB,
	factory *models.Factory,
) (intakeAvailabilitySources, error) {
	intakes, err := factory.ListIntakes(db)
	if err != nil {
		return intakeAvailabilitySources{}, err
	}

	result := intakeAvailabilitySources{}
	scopes := map[string]struct{}{}
	readableIntakeCount := 0
	for i := range intakes {
		intake := &intakes[i]
		if !intakeSourceSupportsAvailability(intake.Source) {
			continue
		}
		readableIntakeCount++

		source, err := deps.itemSource(ctx, db, intake)
		if err != nil {
			result.failedSourceCount++
			log.WithError(err).Warnf("failed to connect intake %s for backlog refresh", intake.ID)
			continue
		}
		availabilitySource, ok := source.(intakeItemAvailabilitySource)
		if !ok {
			result.failedSourceCount++
			continue
		}
		scope := availabilitySource.AvailabilityScope()
		if scope == "" {
			result.failedSourceCount++
			continue
		}
		if _, exists := scopes[scope]; exists {
			continue
		}
		scopes[scope] = struct{}{}
		result.sources = append(result.sources, availabilitySource)
	}

	if len(result.sources) > 0 {
		return result, nil
	}
	if readableIntakeCount == 0 {
		return intakeAvailabilitySources{}, errIntakeRefreshUnsupported
	}
	return intakeAvailabilitySources{}, errIntakeNotConnected
}

func intakeSourceSupportsAvailability(source string) bool {
	return source == models.FactoryIntakeSourceGitHubIssues ||
		source == models.FactoryIntakeSourceProductiveTasks
}

func refreshBacklogOrder(
	ctx context.Context,
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	order *models.FactoryWorkOrder,
	sources []intakeItemAvailabilitySource,
	lookups map[intakeItemAvailabilityKey]intakeItemAvailability,
	archivedBy uuid.UUID,
) (archived bool, failed bool) {
	origin := order.Origin()
	if origin == nil {
		return false, false
	}

	matchedOrigin := false
	unavailable := false
	anyFailure := false
	newFailure := false
	for _, source := range sources {
		itemID, ok := source.ItemIDFromOriginURL(origin.URL)
		if !ok {
			continue
		}
		matchedOrigin = true

		available, cached, err := itemAvailableFromSource(ctx, source, itemID, lookups)
		if errors.Is(err, errIntakeItemOutsideScope) {
			unavailable = true
			continue
		}
		if err != nil {
			log.WithError(err).Warnf("failed to refresh backlog item from %s", source.AvailabilityScope())
			anyFailure = true
			newFailure = newFailure || !cached
			continue
		}
		if available {
			return false, false
		}
		unavailable = true
	}

	if !matchedOrigin || !unavailable {
		return false, newFailure
	}
	if anyFailure {
		return false, newFailure
	}
	return archiveDraftWorkOrderIfCurrent(db, orgID, factory, order.ID, archivedBy)
}

func itemAvailableFromSource(
	ctx context.Context,
	source intakeItemAvailabilitySource,
	itemID string,
	lookups map[intakeItemAvailabilityKey]intakeItemAvailability,
) (available bool, cached bool, err error) {
	key := intakeItemAvailabilityKey{scope: source.AvailabilityScope(), id: itemID}
	if lookup, ok := lookups[key]; ok {
		return lookup.available, true, lookup.err
	}

	available, err = source.IsItemAvailable(ctx, itemID)
	lookups[key] = intakeItemAvailability{available: available, err: err}
	return available, false, err
}

func archiveDraftWorkOrderIfCurrent(
	db *gorm.DB,
	orgID uuid.UUID,
	factory *models.Factory,
	orderID uuid.UUID,
	archivedBy uuid.UUID,
) (archived bool, failed bool) {
	var archivedOrder *models.FactoryWorkOrder
	err := db.Transaction(func(tx *gorm.DB) error {
		current, err := factory.FindWorkOrder(tx.Clauses(clause.Locking{Strength: "UPDATE"}), orderID)
		if err != nil {
			return err
		}
		if current.State != models.FactoryWorkOrderStateDraft {
			return nil
		}

		archivedOrder, err = current.Close(tx, models.FactoryWorkOrderResultRejected, &archivedBy)
		if errors.Is(err, models.ErrFactoryWorkOrderInvalidState) {
			archivedOrder = nil
			return nil
		}
		return err
	})
	if err != nil {
		return false, true
	}
	if archivedOrder == nil {
		return false, false
	}

	publishWorkOrderClosed(
		orgID,
		factory,
		archivedOrder,
		archivedBy,
		models.FactoryWorkOrderStateDraft,
		models.FactoryWorkOrderResultRejected,
		false,
	)
	return true, false
}
