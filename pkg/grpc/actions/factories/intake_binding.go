package factories

import (
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const intakeGitHubAppName = "github"
const intakeJiraAppName = "jira"
const intakeProductiveAppName = "productive"
const intakeSentryAppName = "sentry"

// intakeBinding points the generated trigger at a concrete integration and
// resource. A trigger without one registers no webhook, so the intake would
// exist but never receive an item.
type intakeBinding struct {
	Integration *yaml.IntegrationRef
	// Configuration carries the trigger fields that name the resource to
	// listen on, such as the repository of a GitHub intake.
	Configuration map[string]any
	// Installation is the row the reference points at. Seeding the intake
	// reads the source through the same installation the trigger listens with.
	Installation *models.Integration
}

func (b *intakeBinding) integrationRef() *yaml.IntegrationRef {
	if b == nil {
		return nil
	}
	return b.Integration
}

func (b *intakeBinding) configuration() map[string]any {
	if b == nil {
		return nil
	}
	return b.Configuration
}

func (b *intakeBinding) installation() *models.Integration {
	if b == nil {
		return nil
	}
	return b.Installation
}

// resolveIntakeBinding takes what the trigger needs from the workspace setup:
// setup records the connected version control integration and the backlog
// repository, which is what a GitHub intake listens on. A source that setup
// says nothing about stays unbound, and the user finishes it in the canvas.
func resolveIntakeBinding(
	tx *gorm.DB,
	factory *models.Factory,
	source string,
	integrationID string,
	resourceID string,
) (*intakeBinding, error) {
	if source == models.FactoryIntakeSourceProductiveTasks {
		return resolveProductiveIntakeBinding(tx, factory, integrationID, resourceID)
	}
	if source == models.FactoryIntakeSourceSentryExceptions {
		return resolveSentryIntakeBinding(tx, factory, integrationID, resourceID)
	}
	if source == models.FactoryIntakeSourceJiraIssues {
		return resolveJiraIntakeBinding(tx, factory, integrationID, resourceID)
	}
	if source != models.FactoryIntakeSourceGitHubIssues && source != models.FactoryIntakeSourceDependabotAlerts {
		return nil, nil
	}

	config := factory.OnboardingConfigValue()
	if config.VCSIntegrationID == "" || config.BacklogRepository == "" {
		if source == models.FactoryIntakeSourceDependabotAlerts {
			return nil, invalidArgument("GitHub connection and backlog repository are required")
		}
		return nil, nil
	}

	integration := findIntakeGitHubIntegration(tx, factory, config.VCSIntegrationID)
	if integration == nil {
		if source == models.FactoryIntakeSourceDependabotAlerts {
			return nil, invalidArgument("workspace GitHub integration is not ready")
		}
		return nil, nil
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"repository": config.BacklogRepository},
		Installation:  integration,
	}, nil
}

func resolveJiraIntakeBinding(
	tx *gorm.DB,
	factory *models.Factory,
	integrationID string,
	projectKey string,
) (*intakeBinding, error) {
	integrationID = strings.TrimSpace(integrationID)
	projectKey = strings.TrimSpace(projectKey)
	if integrationID == "" && projectKey == "" {
		return nil, nil
	}
	if integrationID == "" || projectKey == "" {
		return nil, invalidArgument("Jira integration and project are required")
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, invalidArgument("Jira integration is invalid")
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return nil, invalidArgument("Jira integration was not found")
	}
	if integration.AppName != intakeJiraAppName {
		return nil, invalidArgument("selected integration is not Jira")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, invalidArgument("Jira integration is not ready")
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"project": projectKey},
		Installation:  integration,
	}, nil
}

func resolveProductiveIntakeBinding(
	tx *gorm.DB,
	factory *models.Factory,
	integrationID string,
	projectID string,
) (*intakeBinding, error) {
	integrationID = strings.TrimSpace(integrationID)
	projectID = strings.TrimSpace(projectID)
	if integrationID == "" && projectID == "" {
		return nil, nil
	}
	if integrationID == "" || projectID == "" {
		return nil, invalidArgument("Productive.io integration and project are required")
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, invalidArgument("Productive.io integration is invalid")
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return nil, invalidArgument("Productive.io integration was not found")
	}
	if integration.AppName != intakeProductiveAppName {
		return nil, invalidArgument("selected integration is not Productive.io")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, invalidArgument("Productive.io integration is not ready")
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"project": projectID},
		Installation:  integration,
	}, nil
}

func resolveSentryIntakeBinding(
	tx *gorm.DB,
	factory *models.Factory,
	integrationID string,
	projectSlug string,
) (*intakeBinding, error) {
	integrationID = strings.TrimSpace(integrationID)
	projectSlug = strings.TrimSpace(projectSlug)
	if integrationID == "" && projectSlug == "" {
		return nil, nil
	}
	if integrationID == "" || projectSlug == "" {
		return nil, invalidArgument("Sentry integration and project are required")
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, invalidArgument("Sentry integration is invalid")
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		return nil, invalidArgument("Sentry integration was not found")
	}
	if integration.AppName != intakeSentryAppName {
		return nil, invalidArgument("selected integration is not Sentry")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, invalidArgument("Sentry integration is not ready")
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"project": projectSlug},
		Installation:  integration,
	}, nil
}

// findIntakeGitHubIntegration reports nil when the workspace has no GitHub
// installation to listen with. An unbound intake is still worth creating, so a
// miss is logged instead of failing the request.
func findIntakeGitHubIntegration(tx *gorm.DB, factory *models.Factory, integrationID string) *models.Integration {
	id, err := uuid.Parse(integrationID)
	if err != nil {
		log.Warnf("factory %s: intake left unbound, invalid integration id %q", factory.ID, integrationID)
		return nil
	}

	integration, err := models.FindIntegrationInTransaction(tx, factory.OrganizationID, id)
	if err != nil {
		log.Warnf("factory %s: intake left unbound, integration %s not found: %v", factory.ID, id, err)
		return nil
	}

	if integration.AppName != intakeGitHubAppName {
		log.Warnf("factory %s: intake left unbound, integration %s is a %s installation", factory.ID, id, integration.AppName)
		return nil
	}

	return integration
}
