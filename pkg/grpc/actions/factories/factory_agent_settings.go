package factories

import (
	"maps"

	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
)

// User-controlled agent options are separate from template-owned steps and
// environment wiring. Missing options keep their new template defaults.
// When adding a supported agent option, add its configuration key here and
// cover it through ResetOrganizationBuiltInAutomations in
// reset_built_in_automation_settings_test.go. Saving an option in the canvas
// does not make it survive admin reset automatically.
var factoryAgentSettingFields = []string{
	"model", "thinkingLevel", "hostedProvider", "credentials", "llmProvider",
	"machineType", "executionTimeoutSeconds", "workingDirectory", "maxTurns",
	"includeVisualEvidence", "disabledAgentResourceIds", "disabledAgentResourceTools",
	"enabledAgentResourceTools", "skipSetupCommands", "environmentFrom",
	"additionalInstructions",
}

type factoryAgentSettings struct {
	component     string
	configuration map[string]any
	concurrency   *models.ConcurrencySpec
}

func factoryAgentSettingsFromNode(node *models.Node) *factoryAgentSettings {
	if node == nil || !isFactoryAgentHarness(node.ComponentName()) {
		return nil
	}
	// Normalize legacy hosted runners without modifying the saved version.
	snapshot := *node
	snapshot.Configuration = maps.Clone(node.Configuration)
	if node.Ref.Component != nil {
		component := *node.Ref.Component
		snapshot.Ref.Component = &component
	}
	models.RewriteHostedProviderRunnerToSuperPlane(&snapshot)
	settings := &factoryAgentSettings{
		component:     snapshot.ComponentName(),
		configuration: map[string]any{},
		concurrency:   snapshot.Concurrency,
	}
	for _, field := range factoryAgentSettingFields {
		if value, present := snapshot.Configuration[field]; present {
			settings.configuration[field] = value
		}
	}
	return settings
}

func (s *factoryAgentSettings) configure(node yaml.Node) yaml.Node {
	if s == nil {
		return node
	}
	// Subscription credentials come from the workspace source. An admin reset
	// must not restore a retired hosted or BYOK source from saved node options.
	credentials, _ := node.Configuration["credentials"].(map[string]any)
	if node.Component == "runnerCodex" && credentials["source"] == "subscription" {
		s = s.forCodexSubscription()
	}
	if node.Component != s.component {
		for _, field := range []string{"credentials", "model", "maxTurns", "llmProvider", "hostedProvider"} {
			delete(node.Configuration, field)
		}
	}
	node.Component = s.component
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	maps.Copy(node.Configuration, s.configuration)
	if s.concurrency != nil {
		if node.Concurrency == nil {
			node.Concurrency = &yaml.ConcurrencySpec{}
		}
		node.Concurrency.Key = s.concurrency.Key
		if s.concurrency.Max != nil {
			max := *s.concurrency.Max
			node.Concurrency.Max = &max
		}
	}
	return node
}

func (s *factoryAgentSettings) forCodexSubscription() *factoryAgentSettings {
	snapshot := *s
	snapshot.component = "runnerCodex"
	snapshot.configuration = maps.Clone(s.configuration)
	if snapshot.configuration == nil {
		snapshot.configuration = map[string]any{}
	}
	credentials, _ := s.configuration["credentials"].(map[string]any)
	if s.component != "runnerCodex" || credentials["source"] != "subscription" {
		delete(snapshot.configuration, "model")
	}
	for _, field := range []string{"maxTurns", "llmProvider", "hostedProvider"} {
		delete(snapshot.configuration, field)
	}
	snapshot.configuration["credentials"] = map[string]any{"source": "subscription"}
	return &snapshot
}

type factoryTriggerSettings struct {
	configuration map[string]any
	integration   *yaml.IntegrationRef
}

func factoryTriggerSettingsFromNode(node *models.Node) *factoryTriggerSettings {
	if node == nil {
		return nil
	}
	settings := &factoryTriggerSettings{configuration: maps.Clone(node.Configuration)}
	delete(settings.configuration, "customName")
	if node.IntegrationID != nil {
		settings.integration = &yaml.IntegrationRef{ID: *node.IntegrationID}
	}
	return settings
}

func (s *factoryTriggerSettings) configure(node yaml.Node) yaml.Node {
	if s == nil {
		return node
	}
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	maps.Copy(node.Configuration, s.configuration)
	if s.integration != nil {
		node.Integration = s.integration
	}
	return node
}

// Admin reset supplies saved settings to the builders. A request to restore
// factory defaults supplies none, so that action still restores defaults.
type factoryAutomationSettings struct {
	agents       map[string]*factoryAgentSettings
	triggers     map[string]*factoryTriggerSettings
	pullRequests map[string]map[string]any
	agentSteps   map[string]any
}

func factoryAutomationSettingsFromVersion(version *models.CanvasVersion) *factoryAutomationSettings {
	settings := &factoryAutomationSettings{
		agents:       map[string]*factoryAgentSettings{},
		triggers:     map[string]*factoryTriggerSettings{},
		pullRequests: map[string]map[string]any{},
		agentSteps:   map[string]any{},
	}
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if agent := factoryAgentSettingsFromNode(node); agent != nil {
			settings.agents[node.ID] = agent
			settings.agentSteps[node.ID] = node.Configuration["steps"]
		}
		if node.Type == models.NodeTypeTrigger {
			settings.triggers[node.ID] = factoryTriggerSettingsFromNode(node)
		}
		if node.ComponentName() == "github.createPullRequest" || node.ComponentName() == "bitbucket.createPullRequest" {
			if draft, present := node.Configuration["draft"]; present {
				settings.pullRequests[node.ID] = map[string]any{"draft": draft}
			}
		}
	}
	return settings
}
