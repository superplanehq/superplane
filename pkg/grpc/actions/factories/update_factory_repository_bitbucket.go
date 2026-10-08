package factories

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

func updateBitbucketFactoryRepository(
	ctx context.Context,
	deps IntakeDependencies,
	orgID uuid.UUID,
	organizationID string,
	actorID uuid.UUID,
	req *pb.UpdateFactoryRepositoryRequest,
) (*pb.UpdateFactoryRepositoryResponse, error) {
	repositoryName := strings.TrimSpace(req.GetRepository())
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	previous := factory.OnboardingConfigValue()
	if previous.VCSIntegrationID == "" {
		return nil, factoryErrorToStatus(invalidArgument("connect Bitbucket before selecting a repository"), "failed to update factory repository")
	}
	integration, err := findReadyOnboardingIntegration(db, orgID, previous.VCSIntegrationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	if integration.AppName != models.ProviderBitbucket {
		return nil, factoryErrorToStatus(invalidArgument("repository settings currently require a Bitbucket integration"), "failed to update factory repository")
	}

	selectedRepository := repositoryName
	selectedDefaultBranch := strings.TrimSpace(req.GetDefaultBranch())
	selectedExternalID := previous.AppRepositoryExternalID
	selectedIntegration := integration
	repositoryID := int64(0)
	forgeInstall := forgeInstallationID(integration) != ""
	if forgeInstall {
		accountID, accountErr := linkedBitbucketAccountID(ctx, db, organizationID)
		if accountErr != nil {
			return nil, accountErr
		}
		match, matchErr := findVisibleBitbucketRepository(ctx, db, accountID, repositoryName)
		if matchErr != nil {
			return nil, grpcerrors.Internal(matchErr, "failed to list Bitbucket repositories")
		}
		if match == nil {
			return nil, grpcerrors.PermissionDenied(nil, "VCS repository is not accessible")
		}
		bound, bindErr := models.FindOrCreateBitbucketForgeIntegration(db, orgID, match.InstallationID, match.WorkspaceSlug)
		if bindErr != nil {
			return nil, factoryErrorToStatus(bindErr, "failed to update factory repository")
		}
		selectedIntegration = bound
		selectedRepository = match.FullName
		selectedDefaultBranch = match.DefaultBranch
		selectedExternalID = match.UUID
	} else {
		if err := validateBitbucketTokenRepository(integration, repositoryName); err != nil {
			return nil, factoryErrorToStatus(err, "failed to update factory repository")
		}
		if selectedDefaultBranch == "" {
			return nil, factoryErrorToStatus(invalidArgument("default branch is required"), "failed to update factory repository")
		}
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		loaded, err := models.FindFactory(tx, orgID, factory.ID)
		if err != nil {
			return err
		}
		factory = loaded
		previousBranch, err := currentFactoryDefaultBranch(tx, factory, previous.DefaultBranch)
		if err != nil {
			return err
		}
		if err := factory.SnapshotWorkOrderRepository(tx, previous.AppRepository, previousBranch, previous.EffectiveVCSProvider()); err != nil {
			return err
		}
		selectedIntegrationID := selectedIntegration.ID.String()
		patch := models.FactoryOnboardingPatch{
			VCSIntegrationID:  &selectedIntegrationID,
			AppRepository:     &selectedRepository,
			BacklogRepository: &selectedRepository,
			DefaultBranch:     &selectedDefaultBranch,
		}
		if forgeInstall {
			patch.AppRepositoryID = &repositoryID
			patch.BacklogRepositoryID = &repositoryID
			patch.AppRepositoryExternalID = &selectedExternalID
		}
		if err := factory.UpdateOnboarding(tx, patch); err != nil {
			return err
		}
		return reconcileFactoryRepository(
			ctx,
			tx,
			deps,
			factory,
			actorID,
			previous.VCSIntegrationID,
			selectedIntegrationID,
			previous.AppRepository,
			previous.BacklogRepository,
			previousBranch,
			selectedRepository,
		)
	})
	if err != nil {
		if _, _, ok := grpcerrors.HandlerStatus(err); ok {
			return nil, err
		}
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}

	lines, err := factory.ListLines(db)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	serialized, err := serializeFactoryWithLineMetrics(db, factory, lines)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to update factory repository")
	}
	return &pb.UpdateFactoryRepositoryResponse{Factory: serialized}, nil
}

// validateBitbucketTokenRepository confines token-mode repository changes to
// the integration scope: exact repository for repository tokens, workspace
// prefix otherwise. Forge installs revalidate through the visible list above.
func validateBitbucketTokenRepository(integration *models.Integration, repository string) error {
	repository = strings.TrimSpace(repository)
	workspace, slug, ok := strings.Cut(repository, "/")
	slug = strings.TrimSuffix(strings.TrimSpace(slug), ".git")
	if !ok || strings.TrimSpace(workspace) == "" || slug == "" || strings.Contains(slug, "/") {
		return invalidArgument("repository must be in workspace/repository format")
	}
	metadata := integration.Metadata.Data()
	if metadata != nil {
		if repo, ok := metadata["repository"].(map[string]any); ok && repo != nil {
			if fullName, _ := repo["full_name"].(string); strings.TrimSpace(fullName) != "" {
				configured := strings.TrimSuffix(strings.TrimSpace(fullName), ".git")
				if !strings.EqualFold(configured, strings.TrimSuffix(repository, ".git")) {
					return invalidArgument("VCS repository is not accessible")
				}
				return nil
			}
		}
		if ws, ok := metadata["workspace"].(map[string]any); ok && ws != nil {
			if configured, _ := ws["slug"].(string); strings.TrimSpace(configured) != "" {
				if !strings.EqualFold(strings.TrimSpace(configured), strings.TrimSpace(workspace)) {
					return invalidArgument("VCS repository is not accessible")
				}
				return nil
			}
		}
	}
	// ponytail: metadata-scope check only; live revalidation happens at sync/selection
	return invalidArgument("VCS repository is not accessible")
}
