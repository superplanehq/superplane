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

var listRepositoryReviewBotsFromGitHub = listRepositoryReviewBotsForInstallation
var loadFactoryRepositoryReviewBots = loadReviewBotsFromGitHubClient

func ListFactoryRepositoryReviewBots(
	ctx context.Context,
	deps PRFeedbackDependencies,
	organizationID string,
	req *pb.ListFactoryRepositoryReviewBotsRequest,
) (*pb.ListFactoryRepositoryReviewBotsResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository review bots")
	}

	factoryID, err := parseFactoryID(req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository review bots")
	}

	db := database.DB(ctx)
	factory, err := models.FindFactory(db, orgID, factoryID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list repository review bots")
	}

	repository := strings.TrimSpace(req.GetRepository())
	if repository == "" {
		repository = strings.TrimSpace(factory.OnboardingConfigValue().AppRepository)
	}
	if repository == "" {
		return nil, factoryErrorToStatus(invalidArgument("repository is required"), "failed to list repository review bots")
	}

	response := &pb.ListFactoryRepositoryReviewBotsResponse{Repository: repository}
	binding := resolvePRFeedbackBinding(db, factory, repository)
	if binding.installation() == nil {
		return response, nil
	}

	bots, err := listRepositoryReviewBotsFromGitHub(ctx, deps, db, binding.installation(), repository)
	if err != nil {
		log.WithError(err).WithField("factory_id", factory.ID).Warn("repository review bots: GitHub listing failed")
		return response, nil
	}

	response.Bots = serializeRepositoryReviewBots(bots)
	return response, nil
}

func listRepositoryReviewBotsForInstallation(
	ctx context.Context,
	deps PRFeedbackDependencies,
	db *gorm.DB,
	integration *models.Integration,
	repository string,
) ([]repositoryReviewBot, error) {
	client, err := newIntakeGitHubClient(deps, db, integration)
	if err != nil {
		return nil, err
	}

	return loadFactoryRepositoryReviewBots(ctx, client, repository)
}

func serializeRepositoryReviewBots(bots []repositoryReviewBot) []*pb.FactoryRepositoryReviewBot {
	out := make([]*pb.FactoryRepositoryReviewBot, 0, len(bots))
	for _, bot := range bots {
		out = append(out, &pb.FactoryRepositoryReviewBot{
			Login:       bot.Login,
			DisplayName: bot.DisplayName,
		})
	}
	return out
}
