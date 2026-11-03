package factories

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

// Search uses the workspace connection, never credentials supplied by the caller.
func SearchFactoryPullRequestImportItems(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.SearchFactoryPullRequestImportItemsRequest) (*pb.SearchFactoryPullRequestImportItemsResponse, error) {
	db := database.DB(ctx)
	factory, err := pullRequestImportFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search pull requests")
	}
	query := strings.TrimSpace(req.GetQuery())
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 5
	}
	limit = min(limit, 50)
	items, err := searchPullRequestImportItems(ctx, db, deps, factory, query, limit)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to search pull requests")
	}
	if err := attachExistingPullRequestImportTasks(ctx, factory, items); err != nil {
		return nil, factoryErrorToStatus(err, "failed to load imported tasks")
	}
	return &pb.SearchFactoryPullRequestImportItemsResponse{Items: items}, nil
}

func ImportFactoryPullRequest(ctx context.Context, deps IntakeDependencies, organizationID string, req *pb.ImportFactoryPullRequestRequest) (*pb.ImportFactoryPullRequestResponse, error) {
	db := database.DB(ctx)
	factory, err := pullRequestImportFactory(db, organizationID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import pull request")
	}
	if _, err := validateImportPullRequestURL(factory, req.GetPullRequestUrl()); err != nil {
		return nil, factoryErrorToStatus(err, "failed to import pull request")
	}
	// Existing non-archived tasks are navigation only.
	existing, err := existingPullRequestImport(db, factory, req.GetPullRequestUrl())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to find imported task")
	}
	if existing != nil {
		return serializePullRequestImport(ctx, factory, existing, false)
	}
	userID, ok := authentication.GetUserIdFromMetadata(ctx)
	if !ok {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}
	actor, err := uuid.Parse(userID)
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid user id"), "failed to import pull request")
	}
	lineID, err := uuid.Parse(req.GetLineId())
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid line id"), "failed to import pull request")
	}
	// Fetch again at selection time so task content and eligibility use current metadata.
	pr, err := loadHandOffPullRequest(ctx, db, deps, factory, req.GetPullRequestUrl())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load pull request")
	}
	var order *models.FactoryWorkOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := requireCallerCodexConnection(tx, factory, actor); err != nil {
			return err
		}
		line, err := factory.FindLine(tx, lineID)
		if err != nil {
			return err
		}
		stages, err := classifyHandOffStages(tx, factory, line)
		if err != nil {
			return err
		}
		verifyIndex, err := resolveVerifyStageIndex(stages)
		if err != nil {
			return err
		}
		assignees, err := factory.DefaultSubscriptionAssignees(tx, &actor)
		if err != nil {
			return err
		}
		if factory.OnboardingConfigValue().CodexSubscription && len(assignees) == 0 {
			return errCodexConnectionRequired
		}
		if err := factory.ReleaseArchivedPullRequest(tx, models.FactoryPullRequestLookup{
			Provider: pr.Provider, Repository: pr.Repository, Number: pr.Number,
		}); err != nil {
			return err
		}
		origin := models.OriginFromImportedIntakeItem(pr.URL, pr.Title)
		order, err = factory.CreateWorkOrderWithOrigin(tx, pr.Title, pr.Description, &actor, assignees, nil, origin)
		if err != nil {
			return err
		}
		return recordVerifyHandOff(tx, factory, order, line, &actor, pr, verifyIndex, models.FactoryLineStartOptions{})
	})
	if errors.Is(err, models.ErrFactoryPullRequestAlreadyExists) {
		// The losing transaction rolled back the task and sequence allocation.
		existing, lookupErr := existingPullRequestImport(db, factory, req.GetPullRequestUrl())
		if lookupErr != nil {
			return nil, factoryErrorToStatus(lookupErr, "failed to find imported task")
		}
		if existing != nil {
			return serializePullRequestImport(ctx, factory, existing, false)
		}
	}
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to import pull request")
	}
	captureFactoryProductEvent(ctx, db, factory, order.ID,
		productanalytics.EventFactoryTaskCreated,
		productanalytics.StableEventID(productanalytics.EventFactoryTaskCreated, order.ID.String()),
		productanalytics.Properties{"creation_source": "pull_request_import"},
	)
	if _, err := publishVerificationHandOff(ctx, deps, db, factory, order); err != nil {
		return nil, factoryErrorToStatus(err, "failed to publish imported task")
	}
	return serializePullRequestImport(ctx, factory, order, true)
}

func serializePullRequestImport(ctx context.Context, factory *models.Factory, order *models.FactoryWorkOrder, created bool) (*pb.ImportFactoryPullRequestResponse, error) {
	serialized, err := loadAndSerializeWorkOrder(ctx, factory, order)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to load imported task")
	}
	return &pb.ImportFactoryPullRequestResponse{Order: serialized, Created: created}, nil
}

func pullRequestImportFactory(db *gorm.DB, organizationID, factoryID string) (*models.Factory, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	org, err := models.FindOrganizationByIDInTransaction(db, orgID.String())
	if err != nil {
		return nil, err
	}
	if !org.HasExperimentalFeature(features.FeatureFactoryPullRequestImport) {
		return nil, grpcerrors.PermissionDenied(nil, "Pull request import is not enabled.")
	}
	factory, err := findFactory(db, orgID, factoryID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(factory.OnboardingConfigValue().AppRepository) == "" {
		return nil, invalidArgument("This workspace has no repository.")
	}
	return factory, nil
}

func validateImportPullRequestURL(factory *models.Factory, rawURL string) (parsedHandOffPullRequestURL, error) {
	parsed, err := validateHandOffPullRequestURL(factory, rawURL)
	if err != nil {
		return parsed, err
	}
	if !strings.EqualFold(parsed.Repository, factory.OnboardingConfigValue().AppRepository) {
		return parsed, invalidArgument("The pull request repository does not match the workspace repository.")
	}
	return parsed, nil
}

func existingPullRequestImport(db *gorm.DB, factory *models.Factory, rawURL string) (*models.FactoryWorkOrder, error) {
	parsed, err := validateImportPullRequestURL(factory, rawURL)
	if err != nil {
		return nil, err
	}
	// Match the provider identity so URL suffixes and repository casing do not matter.
	pr, err := factory.FindPullRequest(db, models.FactoryPullRequestLookup{Provider: parsed.Provider, Repository: parsed.Repository, Number: int64(parsed.Number)})
	if errors.Is(err, models.ErrFactoryPullRequestNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	order, err := factory.FindWorkOrder(db, pr.WorkOrderID)
	if err != nil {
		return nil, err
	}
	if order.IsArchived() {
		return nil, nil
	}
	return order, nil
}
