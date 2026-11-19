package organizations

import (
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"gorm.io/gorm"
)

var githubWebhookLimit = regexp.MustCompile(`The "([a-z_]+)" event cannot have more than ([0-9]+) hooks`)
var githubWebhookRepository = regexp.MustCompile(`/repos/([a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+)/hooks`)

func integrationWebhookFailures(tx *gorm.DB, instances []models.Integration) (map[uuid.UUID][]*pb.IntegrationWebhookFailure, error) {
	ids := make([]uuid.UUID, 0, len(instances))
	providers := make(map[uuid.UUID]string, len(instances))
	for _, instance := range instances {
		ids = append(ids, instance.ID)
		providers[instance.ID] = instance.AppName
	}
	hooks, err := models.ListIntegrationWebhookFailures(tx, ids)
	if err != nil {
		return nil, err
	}
	byIntegration := map[uuid.UUID][]*pb.IntegrationWebhookFailure{}
	for _, hook := range hooks {
		failure := &pb.IntegrationWebhookFailure{
			Id: hook.ID.String(), State: hook.State,
			Message: "Webhook setup failed. Check the integration settings and try again.",
		}
		if providers[*hook.AppInstallationID] == "github" {
			var config common.WebhookConfiguration
			if err := mapstructure.Decode(hook.Configuration.Data(), &config); err != nil {
				return nil, err
			}
			failure.Repositories, failure.Events = config.WebhookRepositories(), config.EventTypes
			if len(failure.Events) == 0 && config.EventType != "" {
				failure.Events = []string{config.EventType}
			}
			metadata, _ := hook.Metadata.Data().(map[string]any)
			reason, _ := metadata["setupError"].(string)
			failure.Message = "GitHub could not create the webhook. Check repository access and try again."
			if limit := githubWebhookLimit.FindStringSubmatch(reason); len(limit) > 0 {
				failure.LimitEvent = limit[1]
				failure.Message = fmt.Sprintf("GitHub allows %s webhooks for the %s event in each repository. Remove an unused webhook, then try again.", limit[2], limit[1])
			}
			// In a workspace with several repositories, identify the one whose
			// registration failed rather than directing cleanup at every repository.
			if repository := githubWebhookRepository.FindStringSubmatch(reason); len(repository) > 0 {
				failure.Repositories = []string{repository[1]}
			}
		}
		byIntegration[*hook.AppInstallationID] = append(byIntegration[*hook.AppInstallationID], failure)
	}
	return byIntegration, nil
}
