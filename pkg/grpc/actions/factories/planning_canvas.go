package factories

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	planningCanvasTemplateID   = "create-with-agent"
	planningCanvasEntrypointID = "onrun-create-with-agent"
	planningCanvasAgentNodeID  = "planning-agent"
)

var (
	errPlanningClaudeRequired = errors.New("claude is not connected")
	errPlanningGitHubRequired = errors.New("github is not connected")
)

func ensurePlanningCanvas(tx *gorm.DB, factoryModel *models.Factory, userID uuid.UUID) (*models.Canvas, string, error) {
	if err := requirePlanningGitHub(tx, factoryModel); err != nil {
		return nil, "", err
	}
	canvas, err := models.FindPlanningCanvas(tx, factoryModel.OrganizationID, factoryModel.ID)
	if err == nil {
		entrypoint, entryErr := planningCanvasEntrypoint(tx, canvas.ID)
		return canvas, entrypoint, entryErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}
	return createPlanningCanvasFromTemplate(tx, factoryModel, userID)
}

func createPlanningCanvasFromTemplate(tx *gorm.DB, factoryModel *models.Factory, userID uuid.UUID) (*models.Canvas, string, error) {
	agent, err := planningCanvasAgent(tx, factoryModel)
	if err != nil {
		return nil, "", err
	}

	canvasID := uuid.New()
	result, err := materializeFactoryTemplate(planningCanvasTemplateID, factoryTemplateInput{
		appID:        canvasID.String(),
		appName:      models.PlanningCanvasName,
		integrations: planningTemplateIntegrations(tx, factoryModel),
		agent:        planningTemplateAgent(agent),
	})
	if err != nil {
		return nil, "", err
	}

	resource, err := yaml.CanvasFromYAML([]byte(result.canvasYAML))
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	liveVersionID := uuid.New()
	description := models.PlanningCanvasDescription
	if resource.Metadata != nil && strings.TrimSpace(resource.Metadata.Description) != "" {
		description = resource.Metadata.Description
	}

	canvas := &models.Canvas{
		ID:             canvasID,
		OrganizationID: factoryModel.OrganizationID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryModel.ID,
		Name:           models.PlanningCanvasName,
		Description:    description,
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := tx.Create(canvas).Error; err != nil {
		return nil, "", err
	}

	nodes := resource.Nodes()
	edges := resource.Edges()
	for _, node := range nodes {
		row := models.CanvasNode{
			WorkflowID:    canvas.ID,
			NodeID:        node.ID,
			Name:          node.Name,
			Type:          node.Type,
			State:         models.CanvasNodeStateReady,
			Ref:           datatypes.NewJSONType(node.Ref),
			Configuration: datatypes.NewJSONType(node.Configuration),
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}
		row.SetConcurrencySpec(node.Concurrency)
		if err := tx.Create(&row).Error; err != nil {
			return nil, "", err
		}
	}

	version := models.CanvasVersion{
		ID:         liveVersionID,
		WorkflowID: canvas.ID,
		OwnerID:    &userID,
		Nodes:      datatypes.NewJSONSlice(nodes),
		Edges:      datatypes.NewJSONSlice(edges),
		CreatedAt:  &now,
		UpdatedAt:  &now,
	}
	if console, consoleErr := yaml.ConsoleFromYML([]byte(result.consoleYAML)); consoleErr == nil {
		version.ConsolePanels = datatypes.NewJSONType(console.Panels())
		version.ConsoleLayout = datatypes.NewJSONType(console.Layout())
	}
	if err := tx.Create(&version).Error; err != nil {
		return nil, "", err
	}
	return canvas, planningCanvasEntrypointID, nil
}

func planningCanvasEntrypoint(tx *gorm.DB, canvasID uuid.UUID) (string, error) {
	nodes, err := models.FindCanvasNodesInTransaction(tx, canvasID)
	if err != nil {
		return "", err
	}
	for _, node := range nodes {
		if node.Ref.Data().Trigger != nil && node.Ref.Data().Trigger.Name == "onRun" {
			return node.NodeID, nil
		}
	}
	return "", invalidArgument("planning canvas has no onRun entrypoint")
}

func requirePlanningGitHub(tx *gorm.DB, factoryModel *models.Factory) error {
	integrations, err := models.ListIntegrations(tx, factoryModel.OrganizationID)
	if err != nil {
		return err
	}
	for i := range integrations {
		if integrations[i].AppName != intakeGitHubAppName {
			continue
		}
		if integrations[i].State == models.IntegrationStateReady {
			return nil
		}
	}
	return errPlanningGitHubRequired
}

func planningCanvasAgent(tx *gorm.DB, factoryModel *models.Factory) (*intakeAgent, error) {
	if agent := resolveIntakeAgent(tx, factoryModel); agent != nil {
		switch agent.component() {
		case "runnerClaudeCode", models.SuperPlaneRunnerComponent:
			return agent, nil
		}
	}
	if agent := resolveClaudePlanningAgent(tx, factoryModel); agent != nil {
		return agent, nil
	}
	return nil, errPlanningClaudeRequired
}

func resolveClaudePlanningAgent(tx *gorm.DB, factoryModel *models.Factory) *intakeAgent {
	integrations, err := models.ListIntegrations(tx, factoryModel.OrganizationID)
	if err == nil {
		for i := range integrations {
			if integrations[i].AppName != "claude" {
				continue
			}
			if agent := intakeAgentFromIntegration(&integrations[i]); agent != nil {
				return agent
			}
		}
	}
	return intakeAgentFromHostedProvider(tx, factoryModel)
}

func planningTemplateAgent(agent *intakeAgent) *factoryTemplateAgent {
	if agent.component() == models.SuperPlaneRunnerComponent {
		return &factoryTemplateAgent{
			component:        models.SuperPlaneRunnerComponent,
			credentialSource: "hosted",
		}
	}
	out := &factoryTemplateAgent{
		component: "runnerClaudeCode",
		model:     agent.model(),
	}
	credentials := agent.credentials()
	if credentials == nil {
		return out
	}
	source, _ := credentials["source"].(string)
	out.credentialSource = source
	if integration, ok := credentials["integration"].(map[string]any); ok {
		name, _ := integration["name"].(string)
		out.credentialIntegrationName = name
	}
	return out
}

func planningTemplateIntegrations(tx *gorm.DB, factoryModel *models.Factory) map[string]factoryTemplateIntegration {
	out := map[string]factoryTemplateIntegration{}
	integrations, err := models.ListIntegrations(tx, factoryModel.OrganizationID)
	if err != nil {
		return out
	}
	for i := range integrations {
		if integrations[i].State != models.IntegrationStateReady {
			continue
		}
		switch integrations[i].AppName {
		case intakeGitHubAppName, "claude":
			out[integrations[i].AppName] = factoryTemplateIntegration{
				id:   integrations[i].ID.String(),
				name: integrations[i].InstallationName,
			}
		}
	}
	return out
}

func planningCanvasPromptFromConfig(config map[string]any) string {
	if config == nil {
		return ""
	}
	for _, step := range planningCanvasConfigSteps(config["steps"]) {
		prompt, _ := step["prompt"].(string)
		if strings.TrimSpace(prompt) != "" {
			return prompt
		}
	}
	return ""
}

func planningCanvasConfigSteps(raw any) []map[string]any {
	switch steps := raw.(type) {
	case []any:
		out := make([]map[string]any, 0, len(steps))
		for _, step := range steps {
			if item, ok := step.(map[string]any); ok {
				out = append(out, item)
			}
		}
		return out
	case []map[string]any:
		return steps
	default:
		return nil
	}
}
