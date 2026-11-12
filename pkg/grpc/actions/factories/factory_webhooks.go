package factories

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	githubcommon "github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"gorm.io/gorm"
)

var githubHookLimitPattern = regexp.MustCompile(`(?i)cannot have more than\s+\d+\s+hooks?`)
var githubFailedRepositoryPattern = regexp.MustCompile(`(?i)error creating webhook in ([^\s:]+):`)
var githubFailedEventPattern = regexp.MustCompile(`(?i)the "([a-z_]+)" event`)

type factoryWebhookRegistration struct {
	webhook *models.Webhook
	status  *pb.FactoryWebhookStatus
}

func ListFactoryWebhooks(ctx context.Context, organizationID string, req *pb.ListFactoryWebhooksRequest) (*pb.ListFactoryWebhooksResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list workspace webhooks")
	}
	db := database.DB(ctx)
	factory, err := findFactory(db, orgID, req.GetFactoryId())
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list workspace webhooks")
	}
	registrations, err := listFactoryWebhookRegistrations(db, factory)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to list workspace webhooks")
	}
	webhooks := make([]*pb.FactoryWebhookStatus, 0, len(registrations))
	for _, registration := range registrations {
		webhooks = append(webhooks, registration.status)
	}
	slices.SortFunc(webhooks, func(a, b *pb.FactoryWebhookStatus) int {
		if order := strings.Compare(a.GetRepository(), b.GetRepository()); order != 0 {
			return order
		}
		return strings.Compare(a.GetId(), b.GetId())
	})
	return &pb.ListFactoryWebhooksResponse{Webhooks: webhooks}, nil
}

func RetryFactoryWebhook(ctx context.Context, organizationID string, req *pb.RetryFactoryWebhookRequest) (*pb.RetryFactoryWebhookResponse, error) {
	orgID, err := parseOrganizationID(organizationID)
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry workspace webhook")
	}
	webhookID, err := uuid.Parse(req.GetWebhookId())
	if err != nil {
		return nil, factoryErrorToStatus(invalidArgument("invalid webhook id"), "failed to retry workspace webhook")
	}

	var status *pb.FactoryWebhookStatus
	err = database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		factory, err := findFactory(tx, orgID, req.GetFactoryId())
		if err != nil {
			return err
		}
		registrations, err := listFactoryWebhookRegistrations(tx, factory)
		if err != nil {
			return err
		}
		registration, ok := registrations[webhookID]
		if !ok {
			return gorm.ErrRecordNotFound
		}
		status = registration.status
		if registration.webhook.State != models.WebhookStateFailed {
			return nil
		}
		changed, err := registration.webhook.Reprovision(tx, registration.webhook.Configuration.Data())
		if err != nil {
			return err
		}
		if changed {
			status.State = models.WebhookStatePending
			status.Message = ""
		}
		return nil
	})
	if err != nil {
		return nil, factoryErrorToStatus(err, "failed to retry workspace webhook")
	}
	return &pb.RetryFactoryWebhookResponse{Webhook: status}, nil
}

func listFactoryWebhookRegistrations(tx *gorm.DB, factory *models.Factory) (map[uuid.UUID]factoryWebhookRegistration, error) {
	canvases, err := factory.ListCanvases(tx)
	if err != nil {
		return nil, err
	}
	nodes, err := models.ListFactoryWebhookNodes(tx, factory.OrganizationID, factory.ID)
	if err != nil {
		return nil, err
	}
	webhookIDs := make([]uuid.UUID, 0, len(nodes))
	for _, node := range nodes {
		if node.WebhookID != nil {
			webhookIDs = append(webhookIDs, *node.WebhookID)
		}
	}
	nodeWebhooks, err := models.ListWebhooksByIDs(tx, webhookIDs)
	if err != nil {
		return nil, err
	}
	webhookByID := make(map[uuid.UUID]*models.Webhook, len(nodeWebhooks))
	for i := range nodeWebhooks {
		webhookByID[nodeWebhooks[i].ID] = &nodeWebhooks[i]
	}
	integrations, err := models.ListIntegrations(tx, factory.OrganizationID)
	if err != nil {
		return nil, err
	}
	githubIntegrations := map[uuid.UUID]models.Integration{}
	for _, integration := range integrations {
		if integration.AppName == "github" {
			githubIntegrations[integration.ID] = integration
		}
	}
	canvasByID := map[uuid.UUID]models.Canvas{}
	for _, canvas := range canvases {
		canvasByID[canvas.ID] = canvas
	}

	registrations := map[uuid.UUID]factoryWebhookRegistration{}
	for _, node := range nodes {
		if node.WebhookID == nil {
			continue
		}
		canvas, ok := canvasByID[node.WorkflowID]
		if !ok {
			continue
		}
		registration, ok := registrations[*node.WebhookID]
		if !ok {
			webhook, ok := webhookByID[*node.WebhookID]
			if !ok || webhook.AppInstallationID == nil {
				continue
			}
			integration, ok := githubIntegrations[*webhook.AppInstallationID]
			if !ok {
				continue
			}
			registration = factoryWebhookRegistration{webhook: webhook, status: serializeFactoryWebhook(webhook, &integration)}
		}
		addAffectedAutomation(registration.status, canvas)
		registrations[*node.WebhookID] = registration
	}

	config := factory.OnboardingConfigValue()
	if config.EffectiveVCSProvider() != models.ProviderGitHub {
		return registrations, nil
	}
	integrationID, err := uuid.Parse(strings.TrimSpace(config.VCSIntegrationID))
	integration, ok := githubIntegrations[integrationID]
	if err != nil || !ok {
		return registrations, nil
	}
	repositories, err := factory.WorkspaceRepositories(tx)
	if err != nil {
		return nil, err
	}
	allowedRepositories := map[string]bool{}
	for _, repository := range repositories {
		allowedRepositories[models.CanonicalContextRepo(repository)] = true
	}
	webhooks, err := models.ListIntegrationWebhooks(tx, integrationID)
	if err != nil {
		return nil, err
	}
	for i := range webhooks {
		webhook := &webhooks[i]
		if !IsFactoryMergeabilityWebhook(webhook) || !allowedRepositories[models.CanonicalContextRepo(factoryMergeabilityWebhookRepository(webhook.Configuration.Data()))] {
			continue
		}
		registration, ok := registrations[webhook.ID]
		if !ok {
			registration = factoryWebhookRegistration{webhook: webhook, status: serializeFactoryWebhook(webhook, &integration)}
		}
		// Shared hooks refresh workspace pull request status, not automation triggers.
		registrations[webhook.ID] = registration
	}
	return registrations, nil
}

func addAffectedAutomation(status *pb.FactoryWebhookStatus, canvas models.Canvas) {
	for _, automation := range status.AffectedAutomations {
		if automation.GetId() == canvas.ID.String() {
			return
		}
	}
	status.AffectedAutomations = append(status.AffectedAutomations, &pb.Factory_Automation{Id: canvas.ID.String(), Name: canvas.Name})
}

func serializeFactoryWebhook(webhook *models.Webhook, integration *models.Integration) *pb.FactoryWebhookStatus {
	configuration, _ := webhook.Configuration.Data().(map[string]any)
	events := githubWebhookEventTypes(configuration)
	setupError := webhookSetupError(webhook.Metadata.Data())
	repository := githubWebhookRepository(configuration, setupError)
	status := &pb.FactoryWebhookStatus{
		Id:         webhook.ID.String(),
		Repository: repository,
		EventTypes: events,
		State:      webhook.State,
		Host:       githubcommon.HostFromModel(integration).WebHostname(),
	}
	if webhook.State == models.WebhookStateFailed {
		status.Message = githubWebhookFailureMessage(setupError, events)
	}
	return status
}

func githubWebhookEventTypes(configuration map[string]any) []string {
	if configuration == nil {
		return nil
	}
	events := configurationStrings(configuration["eventTypes"])
	if len(events) == 0 {
		if event, ok := configuration["eventType"].(string); ok && strings.TrimSpace(event) != "" {
			events = []string{strings.TrimSpace(event)}
		}
	}
	return events
}

func githubWebhookRepository(configuration map[string]any, setupError string) string {
	if repository, ok := configuration["repository"].(string); ok {
		if repository = strings.TrimSpace(repository); repository != "" && !models.IsWorkspaceRepositoriesExpression(repository) {
			return repository
		}
	}
	if repositories := configurationStrings(configuration["repositories"]); len(repositories) > 0 {
		if match := githubFailedRepositoryPattern.FindStringSubmatch(setupError); len(match) == 2 {
			for _, repository := range repositories {
				if strings.EqualFold(repository, match[1]) {
					return repository
				}
			}
		}
		return repositories[0]
	}
	return ""
}

func githubWebhookFailureMessage(setupError string, events []string) string {
	label := githubFailedEventLabel(setupError, events)
	if githubHookLimitPattern.MatchString(setupError) {
		return fmt.Sprintf("GitHub reached the webhook limit for %s. Remove unused webhooks in the repository settings, then try again.", label)
	}
	return fmt.Sprintf("GitHub could not register the webhook for %s. Check the repository settings, then try again.", label)
}

func githubFailedEventLabel(setupError string, events []string) string {
	event := ""
	if match := githubFailedEventPattern.FindStringSubmatch(setupError); len(match) == 2 && slices.Contains(events, match[1]) {
		event = match[1]
	} else if len(events) == 1 {
		event = events[0]
	}
	switch event {
	case "issues":
		return "issues"
	case "pull_request", "pull_request_review", "pull_request_review_comment":
		return "pull requests"
	case "check_run":
		return "check runs"
	case "check_suite":
		return "check suites"
	case "status":
		return "commit statuses"
	default:
		return "events"
	}
}
