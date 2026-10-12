package factories

import (
	"fmt"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

const (
	// intakeLinearSeedSize is how many open Linear issues a new intake
	// imports. The wizard tells the user this number.
	intakeLinearSeedSize = 10
	// intakeLinearSeedFetchSize is the page read before label filtering, so
	// a label filter can still fill intakeLinearSeedSize.
	intakeLinearSeedFetchSize = 50
)

func seedLinearIssues(
	deps IntakeDependencies,
	tx *gorm.DB,
	canvasID uuid.UUID,
	binding *intakeBinding,
	installation *models.Integration,
) (intakeSeedResult, error) {
	client, err := newIntakeLinearClient(deps, tx, installation)
	if err != nil {
		return intakeSeedResult{}, err
	}

	projectIDs := configurationStrings(binding.Configuration["projects"])
	labels := linearLabelsFromConfiguration(binding.Configuration["labels"])
	issues, err := newestLinearSeedIssues(client, projectIDs, labels)
	if err != nil {
		return intakeSeedResult{}, err
	}

	issues, err = filterLinearIssuesForSeed(tx, canvasID, issues)
	if err != nil {
		return intakeSeedResult{}, err
	}

	payloads := make([]map[string]any, 0, len(issues))
	for _, issue := range issues {
		payloads = append(payloads, linear.IssueEventPayload(issue))
	}
	if err := emitIntakeEvents(tx, canvasID, linear.IssuePayloadType, payloads); err != nil {
		return intakeSeedResult{}, err
	}
	return intakeSeedResult{itemCount: len(payloads)}, nil
}

func newestLinearSeedIssues(client *linear.Client, projectIDs, labels []string) ([]linear.Issue, error) {
	if len(projectIDs) == 0 {
		return nil, fmt.Errorf("linear intake has no projects")
	}

	issues, err := client.ListOpenProjectIssues(projectIDs, intakeLinearSeedFetchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list Linear issues: %w", err)
	}

	matched := make([]linear.Issue, 0, intakeLinearSeedSize)
	for _, issue := range issues {
		if !linearIssueMatchesLabels(issue, labels) {
			continue
		}
		matched = append(matched, issue)
		if len(matched) == intakeLinearSeedSize {
			break
		}
	}
	return matched, nil
}

func filterLinearIssuesForSeed(tx *gorm.DB, canvasID uuid.UUID, issues []linear.Issue) ([]linear.Issue, error) {
	if len(issues) == 0 {
		return issues, nil
	}

	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, canvasID)
	if err != nil {
		return nil, fmt.Errorf("load intake canvas: %w", err)
	}
	if canvas.FactoryID == nil {
		return nil, fmt.Errorf("intake canvas is not owned by a factory")
	}

	factory, err := models.FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return nil, err
	}

	kept := make([]linear.Issue, 0, len(issues))
	for _, issue := range issues {
		ref, ok := linear.IssueRefFromURL(issue.URL)
		if !ok {
			kept = append(kept, issue)
			continue
		}
		ref.ID = issue.ID
		hasOrder, err := linear.IssueHasWorkOrder(tx, factory, ref)
		if err != nil {
			return nil, err
		}
		if hasOrder {
			log.Infof("skipping Linear issue %s: work order already exists", ref.Identifier)
			continue
		}
		kept = append(kept, issue)
	}
	return kept, nil
}

func newIntakeLinearClient(
	deps IntakeDependencies,
	tx *gorm.DB,
	integration *models.Integration,
) (*linear.Client, error) {
	if deps.Registry == nil {
		return nil, fmt.Errorf("integration registry is unavailable")
	}
	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %s is not ready", integration.ID)
	}

	integrationContext := contexts.NewIntegrationContext(tx, nil, integration, deps.Encryptor, deps.Registry, nil)
	client, err := linear.NewClient(deps.Registry.HTTPContext(), integrationContext)
	if err != nil {
		return nil, fmt.Errorf("failed to build Linear client: %w", err)
	}
	return client, nil
}
