package factories

import (
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const intakeGitHubAppName = "github"

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
func resolveIntakeBinding(tx *gorm.DB, factory *models.Factory, source string) *intakeBinding {
	if source != models.FactoryIntakeSourceGitHubIssues {
		return nil
	}

	config := factory.OnboardingConfigValue()
	if config.VCSIntegrationID == "" || config.BacklogRepository == "" {
		return nil
	}

	integration := findIntakeGitHubIntegration(tx, factory, config.VCSIntegrationID)
	if integration == nil {
		return nil
	}

	return &intakeBinding{
		Integration: &yaml.IntegrationRef{
			ID:   integration.ID.String(),
			Name: integration.InstallationName,
		},
		Configuration: map[string]any{"repository": config.BacklogRepository},
		Installation:  integration,
	}
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
