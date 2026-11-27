package factories

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/superplanehq/superplane/pkg/automationspec"
	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

type factoryConfigurationSettingError struct {
	Field   string
	Message string
}

func (e *factoryConfigurationSettingError) Error() string {
	return e.Field + ": " + e.Message
}

func factoryConfigurationBuiltInAlias(templateID string) (string, bool) {
	if _, ok := lookupBuiltInAutomation(templateID); !ok {
		return "", false
	}
	switch templateID {
	case models.FactoryAppTemplateBacklogID:
		return "@superplane/task-analysis", true
	case lineImplementationTemplateID:
		return "@superplane/implement", true
	case "risk-score":
		return "@superplane/merge-confidence", true
	case prFeedbackDiscussionTemplateID:
		return "@superplane/pr-discussion", true
	case prFeedbackChecksTemplateID:
		return "@superplane/pr-checks", true
	default:
		return "@superplane/" + strings.ReplaceAll(templateID, ":", "/"), true
	}
}

// Only these public options cross the YAML boundary. Credentials, managed
// prompts, trigger wiring, and other workspace settings remain in the graph.
var factoryConfigurationAgentFields = map[string]string{
	"additional_instructions":  "additionalInstructions",
	"model":                    "model",
	"thinking_level":           "thinkingLevel",
	"timeout":                  "executionTimeoutSeconds",
	"skip_setup_commands":      "skipSetupCommands",
	"skip_repository_checkout": "skipRepositoryCheckout",
	"include_visual_evidence":  "includeVisualEvidence",
}

// A PR discussion has several execution branches for one logical agent. Other
// templates have one agent; additional harness nodes require an explicit review.
func factoryConfigurationLogicalAgents(version *models.CanvasVersion, templateID string) ([]*models.Node, error) {
	var agents []*models.Node
	if templateID == prFeedbackDiscussionTemplateID {
		spec := models.LiveCanvasSpec{Nodes: version.Nodes, Edges: version.Edges}
		ids := resolvePRFeedbackGraph(spec).discussionRunnerNodeIDs(spec)
		for i := range version.Nodes {
			node := &version.Nodes[i]
			if !isFactoryAgentHarness(node.ComponentName()) {
				continue
			}
			if !ids[node.ID] {
				return nil, configurationSettingError("agent", "This automation has an agent outside its supported execution branches.")
			}
			agents = append(agents, node)
		}
	} else {
		for i := range version.Nodes {
			if isFactoryAgentHarness(version.Nodes[i].ComponentName()) {
				agents = append(agents, &version.Nodes[i])
			}
		}
		if len(agents) > 1 {
			return nil, configurationSettingError("agent", "Multiple distinct agents cannot use one logical agent setting.")
		}
	}
	if len(agents) == 0 && configurationBuiltInNeedsAgent(templateID) {
		return nil, configurationSettingError("agent", "The built-in automation agent is missing or unsupported.")
	}
	return agents, nil
}

func exportFactoryConfigurationBuiltIn(version *models.CanvasVersion, templateID string) (map[string]any, error) {
	if _, ok := factoryConfigurationBuiltInAlias(templateID); !ok {
		return nil, fmt.Errorf("unknown built-in automation %q", templateID)
	}
	agents, err := factoryConfigurationLogicalAgents(version, templateID)
	if err != nil {
		return nil, err
	}
	settings := map[string]any{}
	var logical map[string]any
	for _, node := range agents {
		fields := map[string]any{}
		for field, key := range factoryConfigurationAgentFields {
			value, present := node.Configuration[key]
			if !present {
				continue
			}
			if field == "timeout" {
				if !validFactoryConfigurationTimeout(value) {
					return nil, configurationSettingError("agent.timeout", "The stored timeout cannot be represented as a whole-second duration within the supported limit.")
				}
				value = automationspec.FormatDuration(factoryConfigurationTimeoutSeconds(value))
			}
			if err := validateFactoryConfigurationAgentField(node, field, value); err != nil {
				return nil, prefixFactoryConfigurationSettingError("agent.", err)
			}
			fields[field] = value
		}
		if logical != nil {
			for field := range factoryConfigurationAgentFields {
				zero := factoryConfigurationAgentZero(field)
				if field == "timeout" {
					zero = "0s"
				}
				value, present := fields[field]
				if !present {
					value = zero
				}
				previous, present := logical[field]
				if !present {
					previous = zero
				}
				if !equalCanvasJSON(value, previous) {
					return nil, configurationSettingError("agent."+field, "Execution branches have distinct agent settings. Make their settings equal before export.")
				}
			}
		}

		if logical == nil {
			logical = fields
		}
	}
	if err := exportFactoryConfigurationPublicSettings(version, templateID, settings); err != nil {
		return nil, err
	}
	if len(logical) > 0 {
		settings["agent"] = logical
	}
	return settings, nil
}

func copyFactoryConfigurationBuiltIn(version, defaults *models.CanvasVersion, templateID string, settings map[string]any) ([]models.Node, []models.Edge, error) {
	if _, ok := factoryConfigurationBuiltInAlias(templateID); !ok {
		return nil, nil, fmt.Errorf("unknown built-in automation %q", templateID)
	}
	encoded, err := json.Marshal(version)
	if err != nil {
		return nil, nil, fmt.Errorf("copy automation: %w", err)
	}
	var snapshot models.CanvasVersion
	if err := json.Unmarshal(encoded, &snapshot); err != nil {
		return nil, nil, fmt.Errorf("copy automation: %w", err)
	}
	agents, err := factoryConfigurationLogicalAgents(&snapshot, templateID)
	if err != nil {
		return nil, nil, err
	}
	// Omitted public settings restore defaults without replacing the installed graph.
	for _, node := range agents {
		baseline := findIntakeNode(defaults.Nodes, node.ID)
		if baseline == nil {
			return nil, nil, fmt.Errorf("agent %q has no factory defaults", node.ID)
		}
		for field, key := range factoryConfigurationAgentFields {
			value, present := baseline.Configuration[key]
			if !present {
				value = factoryConfigurationAgentZero(field)
			}
			current, exists := node.Configuration[key]
			if !exists {
				current = factoryConfigurationAgentZero(field)
			}
			if equalCanvasJSON(current, value) {
				continue
			}
			if present {
				if node.Configuration == nil {
					node.Configuration = map[string]any{}
				}
				node.Configuration[key] = value
			} else {
				delete(node.Configuration, key)
			}
		}
	}
	for field := range settings {
		if field == "agent" || field == "intake" || field == "pr_feedback" || field == "template" {
			continue
		}
		return nil, nil, &factoryConfigurationSettingError{Field: field, Message: "This built-in setting is not supported. Use agent for agent options."}
	}
	if raw, present := settings["agent"]; present {
		fields, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, configurationSettingError("agent", "Use a mapping of supported agent settings.")
		}
		if len(agents) == 0 {
			return nil, nil, configurationSettingError("agent", "This automation has no logical agent.")
		}
		for _, agent := range agents {
			for field, value := range fields {
				if err := applyFactoryConfigurationAgentField(agent, field, value); err != nil {
					return nil, nil, prefixFactoryConfigurationSettingError("agent.", err)
				}
			}
		}
	}
	return snapshot.Nodes, snapshot.Edges, nil
}

func applyFactoryConfigurationAgentField(node *models.Node, field string, value any) error {
	if err := validateFactoryConfigurationAgentField(node, field, value); err != nil {
		return err
	}
	if field == "timeout" {
		seconds, _ := automationspec.ParseDuration(value.(string))
		value = seconds
	}
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	node.Configuration[factoryConfigurationAgentFields[field]] = value

	return nil
}

func validateFactoryConfigurationAgentField(node *models.Node, field string, value any) error {
	if _, ok := factoryConfigurationAgentFields[field]; !ok {
		return &factoryConfigurationSettingError{Field: field, Message: "This agent setting is not supported. Use timeout for duration values."}
	}
	switch field {
	case "skip_setup_commands", "skip_repository_checkout", "include_visual_evidence":
		if _, ok := value.(bool); !ok {
			return &factoryConfigurationSettingError{Field: field, Message: "Use a boolean value."}
		}
	case "timeout":
		duration, ok := value.(string)
		if !ok {
			return configurationSettingError(field, "Use a duration such as 2h, 90m, or 30s.")
		}
		if _, err := automationspec.ParseDuration(duration); err != nil {
			return configurationSettingError(field, err.Error())
		}
	default:
		text, ok := value.(string)
		if !ok {
			return &factoryConfigurationSettingError{Field: field, Message: "Use an explicit string value."}
		}
		if field == "thinking_level" {
			normalize := runner.NormalizeThinkingLevel
			if node.ComponentName() == "runnerCodex" {
				normalize = runner.NormalizeCodexThinkingLevel
			}
			if _, err := normalize(text); err != nil {
				return &factoryConfigurationSettingError{Field: field, Message: err.Error()}
			}
		}
	}
	return nil
}

func validateFactoryConfigurationBuiltInNodes(tx *gorm.DB, deps IntakeDependencies, factory *models.Factory, nodes []models.Node) error {
	for _, node := range nodes {
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		if err := validateBuiltInResetNode(yaml.Node{
			ID: node.ID, Name: node.Name, Type: yaml.NodeTypeAction,
			Component: node.ComponentName(), Configuration: node.Configuration,
		}, deps.Registry); err != nil {
			return &factoryConfigurationSettingError{Field: "agent", Message: err.Error()}
		}
		config := map[string]any{"credentials": node.Configuration["credentials"], "model": node.Configuration["model"]}
		if node.Configuration["llmProvider"] == models.UsageProviderCustom {
			config["llmProvider"] = models.UsageProviderCustom
		}
		policy := "byok-runner-model"
		if node.ComponentName() == models.SuperPlaneRunnerComponent {
			policy = "hosted-runner-model"
			config["hostedProvider"] = node.Configuration["hostedProvider"]
		}
		_, err := factoryAutomationPolicy(tx, factory, policy, []any{node.ComponentName(), config}, func() (string, error) {
			if node.ComponentName() != models.SuperPlaneRunnerComponent {
				return models.ResolveBYOKRunnerModel(tx, factory.OrganizationID, &factory.ID, node.ComponentName(), config)
			}
			model := strings.TrimSpace(configString(config, "model"))
			if model == "" {
				_, err := models.ResolveRunnableHostedLLMModel(tx, factory.OrganizationID, &factory.ID)
				return "", err
			}
			allowed, err := models.RunnerAcceptsModel(tx, factory.OrganizationID, &factory.ID, node.ComponentName(), config, model)
			if err == nil && !allowed {
				return "", models.ErrSelectableLLMModelNotAllowed
			}
			return model, err
		})
		if errors.Is(err, models.ErrSelectableLLMModelNotAllowed) || errors.Is(err, models.ErrSuperPlaneRunnerNoModel) || errors.Is(err, models.ErrSuperPlaneRunnerModelNotAllowed) {
			return &factoryConfigurationSettingError{Field: "agent.model", Message: "Select an available model for the configured agent."}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func prefixFactoryConfigurationSettingError(prefix string, err error) error {
	var settingError *factoryConfigurationSettingError
	if errors.As(err, &settingError) {
		return &factoryConfigurationSettingError{Field: prefix + settingError.Field, Message: settingError.Message}
	}
	return err
}

func validFactoryConfigurationTimeout(value any) bool {
	var seconds float64
	switch typed := value.(type) {
	case int:
		seconds = float64(typed)
	case int64:
		seconds = float64(typed)
	case uint64:
		seconds = float64(typed)
	case float64:
		seconds = typed
	default:
		return false
	}
	return seconds >= 0 && seconds <= runner.MaxExecutionTimeoutSecondsRequest && math.Trunc(seconds) == seconds
}

func factoryConfigurationTimeoutSeconds(value any) int {
	switch seconds := value.(type) {
	case int:
		return seconds
	case int64:
		return int(seconds)
	case uint64:
		return int(seconds)
	case float64:
		return int(seconds)
	default:
		return 0
	}
}

func factoryConfigurationAgentZero(field string) any {
	switch field {
	case "skip_setup_commands", "skip_repository_checkout", "include_visual_evidence":
		return false
	case "timeout":
		return 0
	default:
		return ""
	}
}

func factoryConfigurationOverrides(settings, defaults map[string]any) map[string]any {
	overrides := map[string]any{}
	for field, value := range settings {
		baselineValue, present := defaults[field]
		if _, agentField := factoryConfigurationAgentFields[field]; agentField && !present {
			baselineValue = factoryConfigurationAgentZero(field)
			if field == "timeout" {
				baselineValue = "0s"
			}
		}
		if equalCanvasJSON(value, baselineValue) {
			continue
		}
		if fields, ok := value.(map[string]any); ok {
			baseline, _ := defaults[field].(map[string]any)
			nested := factoryConfigurationOverrides(fields, baseline)
			if len(nested) > 0 {
				overrides[field] = nested
			}
			continue
		}
		overrides[field] = value
	}
	return overrides
}
