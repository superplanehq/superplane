package factories

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
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
	"additional_instructions":   "additionalInstructions",
	"model":                     "model",
	"thinking_level":            "thinkingLevel",
	"execution_timeout_seconds": "executionTimeoutSeconds",
	"skip_setup_commands":       "skipSetupCommands",
	"skip_repository_checkout":  "skipRepositoryCheckout",
}

func exportFactoryConfigurationBuiltIn(version *models.CanvasVersion, templateID string) (map[string]any, error) {
	if _, ok := factoryConfigurationBuiltInAlias(templateID); !ok {
		return nil, fmt.Errorf("unknown built-in automation %q", templateID)
	}
	settings := map[string]any{}
	agents := map[string]any{}
	for i := range version.Nodes {
		node := &version.Nodes[i]
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		fields := map[string]any{}
		for field, configurationKey := range factoryConfigurationAgentFields {
			value, present := node.Configuration[configurationKey]
			if !present {
				continue
			}
			if err := validateFactoryConfigurationAgentField(node, field, value); err != nil {
				return nil, prefixFactoryConfigurationSettingError("agents."+node.ID+".", err)
			}
			if templateID == lineImplementationTemplateID && node.ID == implementationAgentNodeID && field == "additional_instructions" {
				settings[field] = value
				continue
			}
			fields[field] = value
		}
		if len(fields) > 0 {
			agents[node.ID] = fields
		}
	}
	if templateID == lineImplementationTemplateID {
		agent := findIntakeNode(version.Nodes, implementationAgentNodeID)
		if agent == nil || !isFactoryAgentHarness(agent.ComponentName()) {
			return nil, fmt.Errorf("the Implement agent is missing or unsupported")
		}
		if _, present := settings["additional_instructions"]; !present {
			settings["additional_instructions"] = ""
		}
	}
	if len(agents) > 0 {
		settings["agents"] = agents
	}
	return settings, nil
}

func copyFactoryConfigurationBuiltIn(version *models.CanvasVersion, templateID string, settings map[string]any) ([]models.Node, []models.Edge, error) {
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
	for field := range settings {
		if field == "agents" || (field == "additional_instructions" && templateID == lineImplementationTemplateID) {
			continue
		}
		return nil, nil, &factoryConfigurationSettingError{Field: field, Message: "This built-in setting is not supported."}
	}
	if templateID == lineImplementationTemplateID {
		instructions, present := settings["additional_instructions"]
		if !present {
			return nil, nil, &factoryConfigurationSettingError{Field: "additional_instructions", Message: "This field is required for Implement."}
		}
		agent := findIntakeNode(snapshot.Nodes, implementationAgentNodeID)
		if agent == nil || !isFactoryAgentHarness(agent.ComponentName()) {
			return nil, nil, fmt.Errorf("the Implement agent is missing or unsupported")
		}
		if err := applyFactoryConfigurationAgentField(agent, "additional_instructions", instructions); err != nil {
			return nil, nil, err
		}
	}
	if rawAgents, present := settings["agents"]; present {
		agents, ok := rawAgents.(map[string]any)
		if !ok {
			return nil, nil, &factoryConfigurationSettingError{Field: "agents", Message: "Use a mapping of agent node IDs to settings."}
		}
		for id, rawFields := range agents {
			agent := findIntakeNode(snapshot.Nodes, id)
			if agent == nil || !isFactoryAgentHarness(agent.ComponentName()) {
				return nil, nil, &factoryConfigurationSettingError{Field: "agents." + id, Message: "This agent does not exist in the live automation."}
			}
			fields, ok := rawFields.(map[string]any)
			if !ok {
				return nil, nil, &factoryConfigurationSettingError{Field: "agents." + id, Message: "Use a mapping of supported agent settings."}
			}
			for field, value := range fields {
				if field == "additional_instructions" && templateID == lineImplementationTemplateID && id == implementationAgentNodeID {
					return nil, nil, &factoryConfigurationSettingError{Field: "agents." + id + "." + field, Message: "Use with.additional_instructions for Implement."}
				}
				if err := applyFactoryConfigurationAgentField(agent, field, value); err != nil {
					return nil, nil, prefixFactoryConfigurationSettingError("agents."+id+".", err)
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
	if node.Configuration == nil {
		node.Configuration = map[string]any{}
	}
	node.Configuration[factoryConfigurationAgentFields[field]] = value
	return nil
}

func validateFactoryConfigurationAgentField(node *models.Node, field string, value any) error {
	if _, ok := factoryConfigurationAgentFields[field]; !ok {
		return &factoryConfigurationSettingError{Field: field, Message: "This agent setting is not supported."}
	}
	switch field {
	case "skip_setup_commands", "skip_repository_checkout":
		if _, ok := value.(bool); !ok {
			return &factoryConfigurationSettingError{Field: field, Message: "Use a boolean value."}
		}
	case "execution_timeout_seconds":
		if !validFactoryConfigurationTimeout(value) {
			return &factoryConfigurationSettingError{Field: field, Message: fmt.Sprintf("Use an integer from 0 through %d.", runner.MaxExecutionTimeoutSecondsRequest)}
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

func validateFactoryConfigurationBuiltInNodes(deps IntakeDependencies, nodes []models.Node) error {
	for _, node := range nodes {
		if !isFactoryAgentHarness(node.ComponentName()) {
			continue
		}
		if err := validateBuiltInResetNode(yaml.Node{
			ID: node.ID, Name: node.Name, Type: yaml.NodeTypeAction,
			Component: node.ComponentName(), Configuration: node.Configuration,
		}, deps.Registry); err != nil {
			return &factoryConfigurationSettingError{Field: "agents." + node.ID, Message: err.Error()}
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
