package factories

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	planningCanvasEntrypointID = "onrun-create-with-agent"
	planningCanvasAgentNodeID  = "planning-agent"
	planningCanvasTimeoutSecs  = 3600
)

func ensurePlanningCanvas(tx *gorm.DB, factoryModel *models.Factory, userID uuid.UUID) (*models.Canvas, string, error) {
	canvas, err := models.FindPlanningCanvas(tx, factoryModel.OrganizationID, factoryModel.ID)
	if err == nil {
		if err := ensurePlanningAgentNode(tx, factoryModel, canvas); err != nil {
			return nil, "", err
		}
		entrypoint, entryErr := planningCanvasEntrypoint(tx, canvas.ID)
		return canvas, entrypoint, entryErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}
	return createPlanningCanvas(tx, factoryModel, userID)
}

func createPlanningCanvas(tx *gorm.DB, factoryModel *models.Factory, userID uuid.UUID) (*models.Canvas, string, error) {
	now := time.Now()
	liveVersionID := uuid.New()
	agent := planningCanvasAgent(tx, factoryModel)
	agentConfig := planningCanvasAgentConfiguration(tx, factoryModel, agent)

	canvas := &models.Canvas{
		ID:             uuid.New(),
		OrganizationID: factoryModel.OrganizationID,
		LiveVersionID:  &liveVersionID,
		FactoryID:      &factoryModel.ID,
		Name:           models.PlanningCanvasName,
		Description:    models.PlanningCanvasDescription,
		CreatedBy:      &userID,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}
	if err := tx.Create(canvas).Error; err != nil {
		return nil, "", err
	}

	trigger := models.CanvasNode{
		WorkflowID: canvas.ID,
		NodeID:     planningCanvasEntrypointID,
		Name:       "Planning",
		Type:       models.NodeTypeTrigger,
		State:      models.CanvasNodeStateReady,
		Ref: datatypes.NewJSONType(models.NodeRef{
			Trigger: &models.TriggerRef{Name: "onRun"},
		}),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := tx.Create(&trigger).Error; err != nil {
		return nil, "", err
	}

	agentNode := models.CanvasNode{
		WorkflowID:    canvas.ID,
		NodeID:        planningCanvasAgentNodeID,
		Name:          "Planning Agent",
		Type:          models.NodeTypeComponent,
		State:         models.CanvasNodeStateReady,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: agent.component()}}),
		Configuration: datatypes.NewJSONType(agentConfig),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := tx.Create(&agentNode).Error; err != nil {
		return nil, "", err
	}

	version := models.CanvasVersion{
		ID:         liveVersionID,
		WorkflowID: canvas.ID,
		OwnerID:    &userID,
		Nodes: datatypes.NewJSONSlice([]models.Node{
			{
				ID:   planningCanvasEntrypointID,
				Name: "Planning",
				Type: models.NodeTypeTrigger,
				Ref:  models.NodeRef{Trigger: &models.TriggerRef{Name: "onRun"}},
			},
			{
				ID:            planningCanvasAgentNodeID,
				Name:          "Planning Agent",
				Type:          models.NodeTypeComponent,
				Ref:           models.NodeRef{Component: &models.ComponentRef{Name: agent.component()}},
				Configuration: agentConfig,
			},
		}),
		Edges: datatypes.NewJSONSlice([]models.Edge{{
			SourceID: planningCanvasEntrypointID,
			TargetID: planningCanvasAgentNodeID,
			Channel:  "default",
		}}),
		CreatedAt: &now,
		UpdatedAt: &now,
	}
	if err := tx.Create(&version).Error; err != nil {
		return nil, "", err
	}
	return canvas, planningCanvasEntrypointID, nil
}

func ensurePlanningAgentNode(tx *gorm.DB, factoryModel *models.Factory, canvas *models.Canvas) error {
	nodes, err := models.FindCanvasNodesInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	agent := planningCanvasAgent(tx, factoryModel)
	agentConfig := planningCanvasAgentConfiguration(tx, factoryModel, agent)
	for i := range nodes {
		if nodes[i].Type == models.NodeTypeComponent {
			return refreshPlanningAgentNode(tx, canvas, &nodes[i], agentConfig)
		}
	}

	entrypoint, err := planningCanvasEntrypoint(tx, canvas.ID)
	if err != nil {
		return err
	}
	version, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}

	now := time.Now()
	agentNode := models.CanvasNode{
		WorkflowID:    canvas.ID,
		NodeID:        planningCanvasAgentNodeID,
		Name:          "Planning Agent",
		Type:          models.NodeTypeComponent,
		State:         models.CanvasNodeStateReady,
		Ref:           datatypes.NewJSONType(models.NodeRef{Component: &models.ComponentRef{Name: agent.component()}}),
		Configuration: datatypes.NewJSONType(agentConfig),
		CreatedAt:     &now,
		UpdatedAt:     &now,
	}
	if err := tx.Create(&agentNode).Error; err != nil {
		return err
	}

	version.Nodes = append(version.Nodes, models.Node{
		ID:            planningCanvasAgentNodeID,
		Name:          "Planning Agent",
		Type:          models.NodeTypeComponent,
		Ref:           models.NodeRef{Component: &models.ComponentRef{Name: agent.component()}},
		Configuration: agentConfig,
	})
	version.Edges = append(version.Edges, models.Edge{
		SourceID: entrypoint,
		TargetID: planningCanvasAgentNodeID,
		Channel:  "default",
	})
	version.UpdatedAt = &now
	return tx.Model(version).Select("Nodes", "Edges", "UpdatedAt").Updates(version).Error
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

func planningCanvasAgent(tx *gorm.DB, factoryModel *models.Factory) *intakeAgent {
	if agent := resolveIntakeAgent(tx, factoryModel); agent != nil {
		return agent
	}
	return &intakeAgent{
		Component: "runnerClaudeCode",
		Credentials: map[string]any{
			"source":      runner.CredentialsSourceIntegration,
			"integration": map[string]any{"name": "claude"},
		},
		Model: "opus",
	}
}

func planningCanvasAgentConfiguration(tx *gorm.DB, factoryModel *models.Factory, agent *intakeAgent) map[string]any {
	configuration := map[string]any{
		"machineType":             runner.MachineTypeE1LargeAMD64,
		"executionTimeoutSeconds": planningCanvasTimeoutSecs,
		"environmentFrom":         planningGitHubEnvironmentFrom(tx, factoryModel),
		"environment": []any{
			map[string]any{
				"name":        "REPO",
				"value":       "{{ root().data.planning_session.repository }}",
				"valueSource": "literal",
			},
		},
		"steps": []any{
			map[string]any{
				"name":    "Clone Repo",
				"type":    runner.AgentStepBash,
				"command": planningCanvasCloneCommand(),
			},
			map[string]any{
				"name":             "Plan with the user",
				"type":             runner.AgentStepPrompt,
				"workingDirectory": "repo",
				"prompt":           planningCanvasPrompt(),
			},
		},
	}
	if credentials := agent.credentials(); credentials != nil {
		configuration["credentials"] = credentials
	}
	if model := agent.model(); model != "" {
		configuration["model"] = model
	}
	return configuration
}

func planningGitHubEnvironmentFrom(tx *gorm.DB, factoryModel *models.Factory) []any {
	name := intakeGitHubAppName
	integrations, err := models.ListIntegrations(tx, factoryModel.OrganizationID)
	if err == nil {
		for i := range integrations {
			if integrations[i].AppName == intakeGitHubAppName && integrations[i].State == models.IntegrationStateReady {
				name = integrations[i].InstallationName
				break
			}
		}
	}
	return []any{
		map[string]any{
			"source": "integration",
			"integration": map[string]any{
				"name": name,
			},
		},
	}
}

func planningCanvasCloneCommand() string {
	return `rm -rf repo
git clone --depth 1 "https://x-access-token:${GITHUB_TOKEN}@github.com/${REPO}.git" repo`
}

func refreshPlanningAgentNode(tx *gorm.DB, canvas *models.Canvas, node *models.CanvasNode, agentConfig map[string]any) error {
	if planningCanvasPromptFromConfig(node.Configuration.Data()) == planningCanvasPrompt() {
		return nil
	}
	now := time.Now()
	node.Configuration = datatypes.NewJSONType(agentConfig)
	node.UpdatedAt = &now
	if err := tx.Model(node).Select("Configuration", "UpdatedAt").Updates(node).Error; err != nil {
		return err
	}
	version, err := models.FindLiveCanvasVersionInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	nodes := append([]models.Node{}, version.Nodes...)
	for i := range nodes {
		if nodes[i].ID == node.NodeID {
			nodes[i].Configuration = agentConfig
		}
	}
	version.Nodes = nodes
	version.UpdatedAt = &now
	return tx.Model(version).Select("Nodes", "UpdatedAt").Updates(version).Error
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

func planningCanvasPrompt() string {
	return `You are in a SuperPlane planning session. The repository is cloned in the working directory.

Use the tools:
- say: post a short message the user can see
- propose_draft: show a draft work order. The user creates or skips it.

Do not create work orders yourself.
Do not call wait_for_user. SuperPlane waits after you stop.

Greet the user with say. Then stop.`
}
