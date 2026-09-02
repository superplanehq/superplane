package factories

import (
	"embed"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/factories"
	"github.com/superplanehq/superplane/pkg/yaml"
	goyaml "gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const (
	factoryTemplateMetadataKey = "factoryTemplate"
	factoryTemplateVersion     = 1
	factoryCanvasIDPlaceholder = "__FACTORY_CANVAS_ID__"
)

var installParamPattern = regexp.MustCompile(`\{\{\s*install_params\.(\w+)\s*\}\}`)

//go:embed templates/*.yaml
var factoryTemplateFiles embed.FS

type factoryAppTemplate struct {
	id                    string
	entrypointNodeID      string
	canvasFile            string
	consoleFile           string
	componentIntegrations map[string]string
}

var factoryAppTemplates = map[string]factoryAppTemplate{
	"line-planning": {
		id:               "line-planning",
		entrypointNodeID: "onrun-create-plan",
		canvasFile:       "templates/line-planning.canvas.yaml",
		consoleFile:      "templates/line-app.console.yaml",
	},
	"line-implementation": {
		id:               "line-implementation",
		entrypointNodeID: "onrun-implement",
		canvasFile:       "templates/line-implementation.canvas.yaml",
		consoleFile:      "templates/line-app.console.yaml",
		componentIntegrations: map[string]string{
			"github.createPullRequest": "github",
		},
	},
	"pr-closure": {
		id:               "pr-closure",
		entrypointNodeID: "on-pr-closed",
		canvasFile:       "templates/pr-closure.canvas.yaml",
		consoleFile:      "templates/event-app.console.yaml",
		componentIntegrations: map[string]string{
			"github.onPullRequest": "github",
		},
	},
	"issue-intake": {
		id:               "issue-intake",
		entrypointNodeID: "on-issue-labeled",
		canvasFile:       "templates/issue-intake.canvas.yaml",
		consoleFile:      "templates/event-app.console.yaml",
		componentIntegrations: map[string]string{
			"github.onIssue": "github",
		},
	},
	"create-with-agent": {
		id:               "create-with-agent",
		entrypointNodeID: "onrun-create-with-agent",
		canvasFile:       "templates/create-with-agent.canvas.yaml",
		consoleFile:      "templates/create-with-agent.console.yaml",
	},
}

type factoryTemplateInput struct {
	appID         string
	appName       string
	installParams map[string]string
	integrations  map[string]factoryTemplateIntegration
	agent         *factoryTemplateAgent
}

type factoryTemplateIntegration struct {
	id   string
	name string
}

type factoryTemplateAgent struct {
	component                 string
	model                     string
	planningModel             string
	credentialSource          string
	credentialIntegrationName string
}

type materializedFactoryTemplate struct {
	templateID  string
	canvasYAML  string
	consoleYAML string
}

func materializeFactoryTemplate(templateID string, input factoryTemplateInput) (*materializedFactoryTemplate, error) {
	template, ok := factoryAppTemplates[templateID]
	if !ok {
		return nil, invalidArgument("unknown factory app template")
	}

	raw, err := factoryTemplateFiles.ReadFile(template.canvasFile)
	if err != nil {
		return nil, fmt.Errorf("read factory app template: %w", err)
	}

	content := strings.ReplaceAll(string(raw), factoryCanvasIDPlaceholder, input.appID)
	content = substituteFactoryInstallParams(content, normalizeFactoryInstallParams(input.installParams))
	canvas, err := yaml.CanvasFromYAML([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("parse factory app template: %w", err)
	}

	canvas.Metadata.ID = input.appID
	canvas.Metadata.Name = input.appName
	wireFactoryTemplate(canvas, template, input)
	markFactoryTemplate(canvas, template)

	canvasYAML, err := goyaml.Marshal(canvas)
	if err != nil {
		return nil, fmt.Errorf("encode factory app template: %w", err)
	}

	consoleYAML, err := materializeFactoryConsole(template, input.appID, input.appName)
	if err != nil {
		return nil, err
	}

	return &materializedFactoryTemplate{
		templateID:  template.id,
		canvasYAML:  string(canvasYAML),
		consoleYAML: consoleYAML,
	}, nil
}

func normalizeFactoryInstallParams(params map[string]string) map[string]string {
	normalized := maps.Clone(params)
	if normalized == nil {
		normalized = map[string]string{}
	}
	if strings.TrimSpace(normalized["defaultBranch"]) == "" {
		normalized["defaultBranch"] = "main"
	}
	repository := strings.TrimSpace(normalized["repository"])
	if repository == "" {
		return normalized
	}
	if strings.TrimSpace(normalized["appRepository"]) == "" {
		normalized["appRepository"] = repository
	}
	if strings.TrimSpace(normalized["backlogRepository"]) == "" {
		normalized["backlogRepository"] = repository
	}
	return normalized
}

func substituteFactoryInstallParams(content string, params map[string]string) string {
	return installParamPattern.ReplaceAllStringFunc(content, func(match string) string {
		parts := installParamPattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		value, ok := params[parts[1]]
		if !ok {
			return match
		}
		return value
	})
}

func wireFactoryTemplate(canvas *yaml.Canvas, template factoryAppTemplate, input factoryTemplateInput) {
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if integrationType := template.componentIntegrations[node.Component]; integrationType != "" {
			if integration, ok := input.integrations[integrationType]; ok {
				node.Integration = &yaml.IntegrationRef{ID: integration.id, Name: integration.name}
			}
		}
		rewriteFactoryIntegrationNames(node.Configuration, input.integrations)
		rewriteFactoryAgent(node, input.agent)
		if node.Component == "runApp" && configString(node.Configuration, "app") == input.appID {
			node.Metadata = maps.Clone(node.Metadata)
			if node.Metadata == nil {
				node.Metadata = map[string]any{}
			}
			node.Metadata["app"] = map[string]any{"id": input.appID, "name": input.appName}
		}
	}
}

func rewriteFactoryIntegrationNames(value any, integrations map[string]factoryTemplateIntegration) {
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			rewriteFactoryIntegrationNames(item, integrations)
		}
	case map[string]any:
		if current["source"] == "integration" {
			if ref, ok := current["integration"].(map[string]any); ok {
				if typeName, ok := ref["name"].(string); ok {
					if integration, exists := integrations[typeName]; exists && integration.name != "" {
						ref["name"] = integration.name
					}
				}
			}
		}
		for _, child := range current {
			rewriteFactoryIntegrationNames(child, integrations)
		}
	}
}

func rewriteFactoryAgent(node *yaml.Node, agent *factoryTemplateAgent) {
	if agent == nil || node.Component != "runnerClaudeCode" {
		return
	}
	node.Component = agent.component
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	if agent.credentialSource == "hosted" {
		node.Configuration["credentials"] = map[string]any{"source": "hosted"}
	} else {
		node.Configuration["credentials"] = map[string]any{
			"source":      "integration",
			"integration": map[string]any{"name": agent.credentialIntegrationName},
		}
	}
	model := agent.model
	if (node.ID == "planner-agent-no-issue" || node.ID == "planning-agent") && agent.planningModel != "" {
		model = agent.planningModel
	}
	node.Configuration["model"] = model
}

func markFactoryTemplate(canvas *yaml.Canvas, template factoryAppTemplate) {
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if node.ID != template.entrypointNodeID {
			continue
		}
		node.Metadata = maps.Clone(node.Metadata)
		if node.Metadata == nil {
			node.Metadata = map[string]any{}
		}
		node.Metadata[factoryTemplateMetadataKey] = map[string]any{
			"id":      template.id,
			"version": factoryTemplateVersion,
		}
		return
	}
}

func materializeFactoryConsole(template factoryAppTemplate, appID, appName string) (string, error) {
	raw, err := factoryTemplateFiles.ReadFile(template.consoleFile)
	if err != nil {
		return "", fmt.Errorf("read factory app console template: %w", err)
	}
	console, err := yaml.ConsoleFromYML([]byte(strings.ReplaceAll(string(raw), factoryCanvasIDPlaceholder, appID)))
	if err != nil {
		return "", fmt.Errorf("parse factory app console template: %w", err)
	}
	console.Metadata.CanvasID = appID
	console.Metadata.Name = appName
	encoded, err := goyaml.Marshal(console)
	if err != nil {
		return "", fmt.Errorf("encode factory app console template: %w", err)
	}
	return string(encoded), nil
}

func factoryTemplateInputFromRequest(req *pb.MaterializeFactoryAppTemplateRequest) factoryTemplateInput {
	integrations := make(map[string]factoryTemplateIntegration, len(req.GetIntegrations()))
	for _, integration := range req.GetIntegrations() {
		integrations[integration.GetType()] = factoryTemplateIntegration{
			id:   integration.GetId(),
			name: integration.GetName(),
		}
	}
	var agent *factoryTemplateAgent
	if req.Agent != nil {
		agent = &factoryTemplateAgent{
			component:                 req.Agent.GetComponent(),
			model:                     req.Agent.GetModel(),
			planningModel:             req.Agent.GetPlanningModel(),
			credentialSource:          req.Agent.GetCredentialSource(),
			credentialIntegrationName: req.Agent.GetCredentialIntegrationName(),
		}
	}
	return factoryTemplateInput{
		appID:         req.GetAppId(),
		installParams: req.GetInstallParams(),
		integrations:  integrations,
		agent:         agent,
	}
}

func resolveFactoryTemplate(nodes []models.Node) (factoryAppTemplate, bool) {
	for _, node := range nodes {
		if metadata, ok := node.Metadata[factoryTemplateMetadataKey].(map[string]any); ok {
			if id, ok := metadata["id"].(string); ok {
				template, found := factoryAppTemplates[id]
				if found {
					return template, true
				}
			}
		}
	}
	for _, template := range factoryAppTemplates {
		for _, node := range nodes {
			if node.ID == template.entrypointNodeID {
				return template, true
			}
		}
	}
	return factoryAppTemplate{}, false
}

func deriveFactoryTemplateInput(tx *gorm.DB, canvas *models.Canvas, version *models.CanvasVersion, template factoryAppTemplate) factoryTemplateInput {
	input := factoryTemplateInput{
		appID:         canvas.ID.String(),
		appName:       canvas.Name,
		installParams: deriveFactoryInstallParams(version.Nodes),
		integrations:  map[string]factoryTemplateIntegration{},
		agent:         deriveFactoryAgent(version.Nodes),
	}
	for _, node := range version.Nodes {
		integrationType := template.componentIntegrations[node.ComponentName()]
		if integrationType != "" && node.IntegrationID != nil {
			integrationID, err := uuid.Parse(*node.IntegrationID)
			if err == nil {
				integration, findErr := models.FindIntegrationInTransaction(tx, canvas.OrganizationID, integrationID)
				if findErr == nil {
					input.integrations[integrationType] = factoryTemplateIntegration{
						id:   integration.ID.String(),
						name: integration.InstallationName,
					}
				}
			}
		}
		for _, name := range factoryIntegrationNames(node.Configuration) {
			integration, err := models.FindIntegrationByName(tx, canvas.OrganizationID, name)
			if err != nil {
				continue
			}
			input.integrations[integration.AppName] = factoryTemplateIntegration{
				id:   integration.ID.String(),
				name: integration.InstallationName,
			}
		}
	}
	return input
}

func factoryIntegrationNames(value any) []string {
	names := []string{}
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case []any:
			for _, item := range typed {
				visit(item)
			}
		case map[string]any:
			if typed["source"] == "integration" {
				if ref, ok := typed["integration"].(map[string]any); ok {
					if name, ok := ref["name"].(string); ok && name != "" {
						names = append(names, name)
					}
				}
			}
			for _, child := range typed {
				visit(child)
			}
		}
	}
	visit(value)
	return names
}

func deriveFactoryInstallParams(nodes []models.Node) map[string]string {
	params := map[string]string{}
	for _, node := range nodes {
		if value := configString(node.Configuration, "repository"); value != "" && params["appRepository"] == "" {
			params["appRepository"] = value
			params["backlogRepository"] = value
		}
		if value := configString(node.Configuration, "base"); value != "" && params["defaultBranch"] == "" {
			params["defaultBranch"] = value
		}
		environment, _ := node.Configuration["environment"].([]any)
		for _, item := range environment {
			entry, _ := item.(map[string]any)
			name, _ := entry["name"].(string)
			value, _ := entry["value"].(string)
			if name == "REPO" && value != "" && params["appRepository"] == "" {
				params["appRepository"] = value
			}
			if name == "BASE" && value != "" && params["defaultBranch"] == "" {
				params["defaultBranch"] = value
			}
		}
	}
	return params
}

func deriveFactoryAgent(nodes []models.Node) *factoryTemplateAgent {
	for _, node := range nodes {
		if node.ComponentName() != "runnerClaudeCode" &&
			node.ComponentName() != "runnerCodex" &&
			node.ComponentName() != "runnerOpenRouter" {
			continue
		}
		model := configString(node.Configuration, "model")
		if model == "" {
			return nil
		}
		agent := &factoryTemplateAgent{component: node.ComponentName(), model: model}
		credentials, _ := node.Configuration["credentials"].(map[string]any)
		agent.credentialSource, _ = credentials["source"].(string)
		if integration, ok := credentials["integration"].(map[string]any); ok {
			agent.credentialIntegrationName, _ = integration["name"].(string)
		}
		if agent.credentialSource != "integration" {
			agent.credentialSource = "hosted"
		}
		return agent
	}
	return nil
}

func configString(configuration map[string]any, key string) string {
	value, _ := configuration[key].(string)
	return value
}

func materializeIntakeDefaults(
	tx *gorm.DB,
	canvas *models.Canvas,
	version *models.CanvasVersion,
	intake *models.FactoryIntake,
) (*materializedFactoryTemplate, error) {
	spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
	graph := resolveIntakeGraph(intake.Source, spec)
	if graph.TriggerNodeID == "" {
		return nil, invalidArgument("intake automation has no trigger to reset")
	}

	var trigger models.Node
	for _, node := range version.Nodes {
		if node.ID == graph.TriggerNodeID {
			trigger = node
			break
		}
	}
	binding := &intakeBinding{Configuration: maps.Clone(trigger.Configuration)}
	if trigger.IntegrationID != nil {
		integrationID, parseErr := uuid.Parse(*trigger.IntegrationID)
		if parseErr == nil {
			integration, findErr := models.FindIntegrationInTransaction(tx, canvas.OrganizationID, integrationID)
			if findErr == nil {
				binding.Integration = &yaml.IntegrationRef{ID: integration.ID.String(), Name: integration.InstallationName}
			}
		}
	}

	defaults, err := buildIntakeCanvas(intakeCanvasRequest{
		OrganizationID: canvas.OrganizationID,
		FactoryID:      intake.FactoryID,
		Source:         intake.Source,
		Name:           canvas.Name,
		Binding:        binding,
	})
	if err != nil {
		return nil, err
	}
	settings := intakeSettingsFromGraph(graph, spec)
	for i := range defaults.Spec.Nodes {
		node := &defaults.Spec.Nodes[i]
		if node.ID == intakeFilterNodeID {
			node.Configuration["expression"] = intakeFilterExpressionFor(intake.Source, settings)
		}
		if node.ID == intakeTriggerNodeID {
			node.Metadata = map[string]any{
				factoryTemplateMetadataKey: map[string]any{
					"id":      "intake:" + intake.Source,
					"version": factoryTemplateVersion,
				},
			}
		}
	}
	defaults.Metadata.ID = canvas.ID.String()
	encoded, err := goyaml.Marshal(defaults)
	if err != nil {
		return nil, fmt.Errorf("encode intake defaults: %w", err)
	}
	return &materializedFactoryTemplate{
		templateID: "intake:" + intake.Source,
		canvasYAML: string(encoded),
	}, nil
}

func materializeBacklogDefaults(canvas *models.Canvas, version *models.CanvasVersion) (*materializedFactoryTemplate, error) {
	defaults := buildBacklogCanvas(backlogCanvasRequest{
		Name:  canvas.Name,
		Agent: intakeAgentFromCanvasNodes(version.Nodes),
	})
	defaults.Metadata.ID = canvas.ID.String()
	for i := range defaults.Spec.Nodes {
		node := &defaults.Spec.Nodes[i]
		if node.ID != backlogTriggerNodeID {
			continue
		}
		node.Metadata = map[string]any{
			factoryTemplateMetadataKey: map[string]any{
				"id":      "backlog",
				"version": factoryTemplateVersion,
			},
		}
	}
	encoded, err := goyaml.Marshal(defaults)
	if err != nil {
		return nil, fmt.Errorf("encode Backlog defaults: %w", err)
	}
	return &materializedFactoryTemplate{
		templateID: "backlog",
		canvasYAML: string(encoded),
	}, nil
}

func intakeAgentFromCanvasNodes(nodes []models.Node) *intakeAgent {
	for _, node := range nodes {
		if node.ComponentName() != "runnerClaudeCode" &&
			node.ComponentName() != "runnerCodex" &&
			node.ComponentName() != "runnerOpenRouter" {
			continue
		}
		credentials, _ := node.Configuration["credentials"].(map[string]any)
		return &intakeAgent{
			Component:   node.ComponentName(),
			Credentials: maps.Clone(credentials),
			Model:       configString(node.Configuration, "model"),
		}
	}
	return nil
}

func findFactoryAppForDefaults(tx *gorm.DB, organizationID, factoryID, appID uuid.UUID) (*models.Canvas, *models.CanvasVersion, error) {
	canvas, err := models.FindCanvasInTransaction(tx, organizationID, appID)
	if err != nil {
		return nil, nil, err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != factoryID {
		return nil, nil, fmt.Errorf("canvas %s does not belong to factory %s: %w", appID, factoryID, gorm.ErrRecordNotFound)
	}
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return nil, nil, err
	}
	return canvas, version, nil
}
