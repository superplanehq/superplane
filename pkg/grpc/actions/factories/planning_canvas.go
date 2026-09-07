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
	errPlanningAgentRequired  = errors.New("agent is not connected")
	errPlanningGitHubRequired = errors.New("github is not connected")
)

const (
	planningCanvasGreetCloser     = "Greet the user in plain text. Then stop."
	planningCanvasFirstTurnCloser = "" +
		"Refine key: {{ root().data.planning_session.refine_key }}\n" +
		"Refine title: {{ root().data.planning_session.refine_title }}\n" +
		"Refine description: {{ root().data.planning_session.refine_description }}\n\n" +
		"If the refine key is not empty, you already have that draft task. Tell the user you have this task and you are ready to refine it. Ask what they want to change. Do not explore the repository unless you need to understand a requested change. Do not call propose_draft until they say what to change. Then stop.\n\n" +
		"If the refine key is empty, greet the user in plain text. Then stop."
)

func ensurePlanningCanvas(tx *gorm.DB, factoryModel *models.Factory, userID uuid.UUID) (*models.Canvas, string, error) {
	if err := requirePlanningGitHub(tx, factoryModel); err != nil {
		return nil, "", err
	}
	canvas, err := models.FindPlanningCanvas(tx, factoryModel.OrganizationID, factoryModel.ID)
	if err == nil {
		if syncErr := syncPlanningCanvasStockPrompt(tx, canvas.ID); syncErr != nil {
			return nil, "", syncErr
		}
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
		return agent, nil
	}
	return nil, errPlanningAgentRequired
}

func planningTemplateAgent(agent *intakeAgent) *factoryTemplateAgent {
	out := &factoryTemplateAgent{
		component: agent.component(),
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
		case intakeGitHubAppName, "claude", "openai", "openrouter":
			out[integrations[i].AppName] = factoryTemplateIntegration{
				id:   integrations[i].ID.String(),
				name: integrations[i].InstallationName,
			}
		}
	}
	return out
}

func syncPlanningCanvasStockPrompt(tx *gorm.DB, canvasID uuid.UUID) error {
	nodes, err := models.FindCanvasNodesInTransaction(tx, canvasID)
	if err != nil {
		return err
	}
	changed := false
	for i := range nodes {
		if nodes[i].Type != models.NodeTypeComponent {
			continue
		}
		config := nodes[i].Configuration.Data()
		if !rewritePlanningCanvasPrompt(config) {
			continue
		}
		nodes[i].Configuration = datatypes.NewJSONType(config)
		if err := tx.Model(&nodes[i]).Select("Configuration").Updates(&nodes[i]).Error; err != nil {
			return err
		}
		changed = true
	}
	if !changed {
		return nil
	}
	live, err := models.FindLiveCanvasVersionInTransaction(tx, canvasID)
	if err != nil {
		return err
	}
	versionNodes := append([]models.Node(nil), live.Nodes...)
	for i := range versionNodes {
		rewritePlanningCanvasPrompt(versionNodes[i].Configuration)
	}
	now := time.Now()
	live.Nodes = datatypes.NewJSONSlice(versionNodes)
	live.UpdatedAt = &now
	return tx.Model(live).Select("Nodes", "UpdatedAt").Updates(live).Error
}

func rewritePlanningCanvasPrompt(config map[string]any) bool {
	if config == nil {
		return false
	}
	rewritten := false
	for _, step := range planningCanvasConfigSteps(config["steps"]) {
		prompt, _ := step["prompt"].(string)
		next, ok := replacePlanningCanvasGreetCloser(prompt)
		if !ok {
			continue
		}
		step["prompt"] = next
		rewritten = true
	}
	return rewritten
}

func replacePlanningCanvasGreetCloser(prompt string) (string, bool) {
	if !strings.Contains(prompt, planningCanvasGreetCloser) {
		return prompt, false
	}
	if strings.Contains(prompt, "planning_session.refine_key") {
		return prompt, false
	}
	return strings.Replace(prompt, planningCanvasGreetCloser, planningCanvasFirstTurnCloser, 1), true
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
