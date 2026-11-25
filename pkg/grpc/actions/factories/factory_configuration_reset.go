package factories

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func factoryConfigurationSavedBinding(tx *gorm.DB, canvas *models.Canvas) (map[string]any, error) {
	if tx == nil {
		return nil, nil
	}
	owner, err := models.FindFactoryConfigurationAutomationByCanvas(tx, canvas.ID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return owner.Binding, nil
}

func preserveFactoryConfigurationPRFeedbackBinding(tx *gorm.DB, canvas *models.Canvas, request *prFeedbackBuildRequest) error {
	binding, err := factoryConfigurationSavedBinding(tx, canvas)
	if err != nil {
		return err
	}
	if resource, ok := binding["resource_id"].(string); ok {
		request.Repository = resource
		if request.Binding == nil {
			request.Binding = &intakeBinding{}
		}
		request.Binding.Configuration = map[string]any{"repository": resource}
	}
	raw, explicit := binding["integration_id"].(string)
	if !explicit {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return err
	}
	integration, err := factoryAutomationIntegrationByID(tx, canvas.OrganizationID, id)
	if err != nil {
		return err
	}
	if request.Binding == nil {
		request.Binding = &intakeBinding{Configuration: map[string]any{"repository": request.Repository}}
	}
	request.Binding.Integration = &yaml.IntegrationRef{ID: raw, Name: integration.InstallationName}
	request.Binding.Installation = integration
	return nil
}

func materializeFactoryConfigurationBoundTemplate(tx *gorm.DB, canvas *models.Canvas, template factoryAppTemplate, input factoryTemplateInput, binding map[string]any) (*materializedFactoryTemplate, error) {
	repository, pinnedRepository := binding["resource_id"].(string)
	if pinnedRepository {
		input.installParams["appRepository"] = repository
		input.installParams["backlogRepository"] = repository
	}
	raw, pinnedIntegration := binding["integration_id"].(string)
	rewrites := map[string]factoryTemplateIntegration{}
	var integration *models.Integration
	if pinnedIntegration {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, err
		}
		integration, err = factoryAutomationIntegrationByID(tx, canvas.OrganizationID, id)
		if err != nil {
			return nil, err
		}
		input.installParams["githubHost"] = githubHostForVCSIntegration(tx, canvas.OrganizationID, raw)
		replacement := factoryTemplateIntegration{id: raw, name: integration.InstallationName}
		rewrites[template.provider] = replacement
		if previous, exists := input.integrations[template.provider]; exists && previous.name != "" {
			rewrites[previous.name] = replacement
		}
		if input.integrations == nil {
			input.integrations = map[string]factoryTemplateIntegration{}
		}
		input.integrations[template.provider] = replacement
	}
	result, err := materializeFactoryTemplate(template.id, template.provider, input)
	if err != nil || !pinnedRepository && !pinnedIntegration {
		return result, err
	}
	document, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	if err != nil {
		return nil, err
	}
	for i := range document.Spec.Nodes {
		node := &document.Spec.Nodes[i]
		if strings.HasPrefix(node.Component, template.provider+".") {
			if pinnedRepository && node.Type == yaml.NodeTypeTrigger {
				if _, listensOnRepository := node.Configuration["repository"]; listensOnRepository {
					node.Configuration["repository"] = repository
				}
			}
			if pinnedIntegration {
				node.Integration = &yaml.IntegrationRef{ID: raw, Name: integration.InstallationName}
			}
		}
		if pinnedIntegration {
			rewriteFactoryIntegrationNames(node.Configuration, rewrites)
		}
	}
	encoded, err := goyaml.Marshal(document)
	if err != nil {
		return nil, err
	}
	result.canvasYAML = string(encoded)
	return result, nil
}
