package factories

import (
	"errors"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type factoryConfigurationValidationInputs struct {
	sources           map[uuid.UUID]*models.AutomationSource
	existingArtifacts map[uuid.UUID]bool
	secrets           map[string]bool
}

func loadFactoryConfigurationValidationInputs(tx *gorm.DB, target *factoryConfigurationTarget, manifest *factoryConfigurationManifest) (*factoryConfigurationValidationInputs, error) {
	inputs := &factoryConfigurationValidationInputs{sources: map[uuid.UUID]*models.AutomationSource{}, existingArtifacts: map[uuid.UUID]bool{}, secrets: map[string]bool{}}
	var sourceIDs []uuid.UUID
	var secretNames []string
	byKey := configurationStagesByKey(target)
	for _, stage := range manifest.Stages {
		live := byKey[stage.Key]
		if live == nil {
			continue
		}
		for _, automation := range stage.Automations {
			current := automation.current
			if current == nil {
				sourceIDs = append(sourceIDs, configurationAutomationSourceID(target.factory.ID, configurationStagePlacement(live), automation.ID))
			}
			if current != nil && (current.templateID != "" || !factoryConfigurationAutomationContentChanged(current, automation, manifest)) {
				continue
			}
			for _, step := range manifest.Definitions[configurationAutomationPath(automation.Uses)].Steps {
				secretNames = append(secretNames, step.Secrets...)
			}
		}
	}
	sources, err := models.FindAutomationSourcesByIDs(tx, target.factory.OrganizationID, sourceIDs)
	if err != nil {
		return nil, err
	}
	var artifactIDs []uuid.UUID
	for i := range sources {
		source := &sources[i]
		inputs.sources[source.ID] = source
		id, err := uuid.Parse(source.ArtifactID)
		if err != nil {
			return nil, err
		}
		artifactIDs = append(artifactIDs, id)
	}
	existing, err := models.FindExistingCanvasIDs(tx, target.factory.OrganizationID, artifactIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range existing {
		inputs.existingArtifacts[id] = true
	}
	secrets, err := models.FindExistingSecretNames(tx, models.DomainTypeOrganization, target.factory.OrganizationID, secretNames)
	if err != nil {
		return nil, err
	}
	for _, name := range secrets {
		inputs.secrets[name] = true
	}
	return inputs, nil
}

func (t *factoryConfigurationTarget) findCanvasByName(tx *gorm.DB, name string) (*models.Canvas, error) {
	if reads := factoryAutomationReads(tx, t.factory); reads != nil {
		for i := range reads.canvases {
			canvas := &reads.canvases[i]
			if canvas.Name == name {
				return canvas, nil
			}
		}
		return nil, nil
	}
	canvas, err := models.FindCanvasByName(tx, t.factory.OrganizationID, &t.factory.ID, name)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return canvas, err
}

func (t *factoryConfigurationTarget) validateCustomSecrets(tx *gorm.DB, document *automationspec.Document) error {
	if t.validation == nil {
		return validateFactoryConfigurationCustomSecrets(tx, t.factory.OrganizationID, document)
	}
	return checkFactoryConfigurationCustomSecrets(document, t.validation.secrets)
}
