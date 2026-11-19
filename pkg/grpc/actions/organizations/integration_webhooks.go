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
		setup, err := serializeWebhookSetup(hook, providers[*hook.AppInstallationID])
		if err != nil {
			return nil, err
		}
		id := *hook.AppInstallationID
		byIntegration[id] = append(byIntegration[id], serializeWebhookFailure(hook, setup, providers[id]))
	}
	return byIntegration, nil
}

func describeIntegrationWebhooks(db *gorm.DB, instance *models.Integration, status *pb.Integration_Status) error {
	if instance.AppName != "github" {
		failures, err := integrationWebhookFailures(db, []models.Integration{*instance})
		status.WebhookFailures = failures[instance.ID]
		return err
	}

	hooks, err := models.ListIntegrationWebhooks(db, instance.ID)
	if err != nil {
		return err
	}
	for _, hook := range hooks {
		setup, err := serializeWebhookSetup(hook, instance.AppName)
		if err != nil {
			return err
		}
		status.WebhookSetups = append(status.WebhookSetups, setup)
		if hook.HasSetupFailure() {
			status.WebhookFailures = append(status.WebhookFailures, serializeWebhookFailure(hook, setup, instance.AppName))
		}
	}
	return nil
}

func serializeWebhookSetup(hook models.Webhook, provider string) (*pb.IntegrationWebhookSetup, error) {
	setup := &pb.IntegrationWebhookSetup{Id: hook.ID.String(), State: hook.State}
	if provider != "github" {
		return setup, nil
	}
	var config common.WebhookConfiguration
	if err := mapstructure.Decode(hook.Configuration.Data(), &config); err != nil {
		return nil, err
	}
	setup.Repositories, setup.Events = config.WebhookRepositories(), config.EventTypes
	if len(setup.Events) == 0 && config.EventType != "" {
		setup.Events = []string{config.EventType}
	}
	return setup, nil
}

func serializeWebhookFailure(hook models.Webhook, setup *pb.IntegrationWebhookSetup, provider string) *pb.IntegrationWebhookFailure {
	failure := &pb.IntegrationWebhookFailure{
		Id: setup.Id, State: setup.State, Repositories: setup.Repositories, Events: setup.Events,
		Message: "Webhook setup failed. Check the integration settings and try again.",
	}
	if provider != "github" {
		return failure
	}
	reason := hook.SetupError()
	failure.Message = "GitHub could not create the webhook. Check repository access and try again."
	if limit := githubWebhookLimit.FindStringSubmatch(reason); len(limit) > 0 {
		failure.LimitEvent = limit[1]
		failure.Message = fmt.Sprintf("GitHub allows %s webhooks for the %s event in each repository. Remove an unused webhook, then try again.", limit[2], limit[1])
	}
	// Direct cleanup to the repository that failed in a multi-repository setup.
	if repository := githubWebhookRepository.FindStringSubmatch(reason); len(repository) > 0 {
		failure.Repositories = []string{repository[1]}
	}
	return failure
}
