package factories

import (
	"context"
	"encoding/json"
	"maps"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// These inputs live for one configuration operation and one transaction only.
// Publication and row locking still read through their existing model paths.
type factoryAutomationReadInputs struct {
	connection            gorm.ConnPool
	factory               *models.Factory
	canvases              []models.Canvas
	versions              map[uuid.UUID]*models.CanvasVersion
	lines                 []models.FactoryLine
	integrations          []models.Integration
	integrationsByID      map[uuid.UUID]*models.Integration
	integrationsByName    map[string]*models.Integration
	agent                 *intakeAgent
	agentResolved         bool
	hostedAgent           *intakeAgent
	hostedAgentResolved   bool
	vcsName               string
	vcsResolved           bool
	policies              map[string]any
	workspaceRepositories []string
}

type factoryAutomationReadInputsKey struct{}

func newFactoryAutomationReadInputs(tx *gorm.DB, factory *models.Factory, canvases []models.Canvas, versions map[uuid.UUID]*models.CanvasVersion, lines []models.FactoryLine) (*factoryAutomationReadInputs, error) {
	integrations, err := models.ListIntegrations(tx, factory.OrganizationID)
	if err != nil {
		return nil, err
	}
	reads := &factoryAutomationReadInputs{
		connection: tx.Statement.ConnPool, factory: factory,
		canvases: canvases, versions: versions, lines: lines,
		integrations:       integrations,
		integrationsByID:   map[uuid.UUID]*models.Integration{},
		integrationsByName: map[string]*models.Integration{},
		policies:           map[string]any{},
	}
	for i := range reads.integrations {
		integration := &reads.integrations[i]
		reads.integrationsByID[integration.ID] = integration
		reads.integrationsByName[integration.InstallationName] = integration
	}
	return reads, nil
}

func withFactoryAutomationReadInputs(tx *gorm.DB, reads *factoryAutomationReadInputs) *gorm.DB {
	return tx.WithContext(context.WithValue(tx.Statement.Context, factoryAutomationReadInputsKey{}, reads))
}

func factoryAutomationReadsForOrganization(tx *gorm.DB, organizationID uuid.UUID) *factoryAutomationReadInputs {
	if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
		return nil
	}
	reads, _ := tx.Statement.Context.Value(factoryAutomationReadInputsKey{}).(*factoryAutomationReadInputs)
	if reads == nil || reads.connection != tx.Statement.ConnPool || reads.factory.OrganizationID != organizationID {
		return nil
	}
	return reads
}

func factoryAutomationReads(tx *gorm.DB, factory *models.Factory) *factoryAutomationReadInputs {
	if factory == nil {
		return nil
	}
	reads := factoryAutomationReadsForOrganization(tx, factory.OrganizationID)
	if reads == nil || reads.factory.ID != factory.ID {
		return nil
	}
	return reads
}

func factoryAutomationIntegrationByID(tx *gorm.DB, organizationID, id uuid.UUID) (*models.Integration, error) {
	if reads := factoryAutomationReadsForOrganization(tx, organizationID); reads != nil {
		integration := reads.integrationsByID[id]
		if integration == nil || integration.OrganizationID != organizationID {
			return nil, gorm.ErrRecordNotFound
		}
		return integration, nil
	}
	return models.FindIntegrationInTransaction(tx, organizationID, id)
}

func factoryAutomationIntegrationByName(tx *gorm.DB, organizationID uuid.UUID, name string) (*models.Integration, error) {
	if reads := factoryAutomationReadsForOrganization(tx, organizationID); reads != nil {
		integration := reads.integrationsByName[name]
		if integration == nil || integration.OrganizationID != organizationID {
			return nil, gorm.ErrRecordNotFound
		}
		return integration, nil
	}
	return models.FindIntegrationByName(tx, organizationID, name)
}

func factoryAutomationIntegrations(tx *gorm.DB, organizationID uuid.UUID) ([]models.Integration, error) {
	if reads := factoryAutomationReadsForOrganization(tx, organizationID); reads != nil {
		return reads.integrations, nil
	}
	return models.ListIntegrations(tx, organizationID)
}

func factoryAutomationLines(tx *gorm.DB, factory *models.Factory) ([]models.FactoryLine, error) {
	if reads := factoryAutomationReads(tx, factory); reads != nil {
		return reads.lines, nil
	}
	return factory.ListLines(tx)
}

func cloneFactoryAutomationAgent(agent *intakeAgent) *intakeAgent {
	if agent == nil {
		return nil
	}
	clone := *agent
	clone.Credentials = maps.Clone(agent.Credentials)
	for field, value := range clone.Credentials {
		if nested, ok := value.(map[string]any); ok {
			clone.Credentials[field] = maps.Clone(nested)
		}
	}
	return &clone
}

// Include every lookup input, including credential identity and actor. Only raw
// model results are shared; document diagnostics keep their own source location.
func factoryAutomationPolicy[T any](tx *gorm.DB, factory *models.Factory, operation string, inputs []any, lookup func() (T, error)) (T, error) {
	reads := factoryAutomationReads(tx, factory)
	if reads == nil {
		return lookup()
	}
	encoded, err := json.Marshal(inputs)
	if err != nil {
		return lookup()
	}
	key := operation + "\x00" + string(encoded)
	if result, ok := reads.policies[key]; ok {
		return result.(T), nil
	}
	result, err := lookup()
	if err == nil {
		reads.policies[key] = result
	}
	return result, err
}
