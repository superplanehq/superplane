package factories

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

var listRepositoryStatusChecksFromGitHub = listRepositoryStatusChecksForInstallation
var loadFactoryRepositoryStatusCheckSnapshot = loadSnapshotFromGitHubClient

func ListFactoryRepositoryStatusChecks(
	ctx context.Context,
	deps PRFeedbackDependencies,
	organizationID string,
	req *pb.ListFactoryRepositoryStatusChecksRequest,
) (*pb.ListFactoryRepositoryStatusChecksResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository status checks")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository status checks")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository status checks")
	}

	repository := strings.TrimSpace(req.GetRepository())
	if repository == "" {
		repository = strings.TrimSpace(factory.OnboardingConfigValue().AppRepository)
	}
	if repository == "" {
		return nil, factoryErrorToStatus(invalidArgument("repository is required"), "failed to list repository status checks")
	}

	response := &pb.ListFactoryRepositoryStatusChecksResponse{Repository: repository}
	binding := resolvePRFeedbackBinding(db, factory, repository)
	if binding.installation() == nil {
		return response, nil
	}

	checks, err := listRepositoryStatusChecksFromGitHub(ctx, deps, db, binding.installation(), repository)
	if err != nil {
		log.WithError(err).WithField("factory_id", factory.ID).Warn("repository status checks: GitHub listing failed")
		return response, nil
	}

	response.Checks = serializeRepositoryStatusChecks(checks)
	return response, nil
}

func listRepositoryStatusChecksForInstallation(
	ctx context.Context,
	deps PRFeedbackDependencies,
	db *gorm.DB,
	integration *models.Integration,
	repository string,
) ([]repositoryStatusCheck, error) {
	client, err := newIntakeGitHubClient(deps, db, integration)
	if err != nil {
		return nil, err
	}

	snapshot, err := loadFactoryRepositoryStatusCheckSnapshot(ctx, client, repository)
	if err != nil {
		return nil, err
	}

	return mergeRepositoryStatusChecks(snapshot.Required, snapshot.Observed), nil
}

func serializeRepositoryStatusChecks(checks []repositoryStatusCheck) []*pb.FactoryRepositoryStatusCheck {
	out := make([]*pb.FactoryRepositoryStatusCheck, 0, len(checks))
	for _, check := range checks {
		out = append(out, &pb.FactoryRepositoryStatusCheck{
			Name:                 check.Name,
			Required:             check.Required,
			DetailsUrl:           check.DetailsURL,
			SuggestedIntegration: check.SuggestedIntegration,
		})
	}
	return out
}
