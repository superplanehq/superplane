// Package automationspec defines the public v2 custom automation language.
package automationspec

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	APIVersion = "v2"
	Kind       = "Automation"

	DefaultIcon           = "sparkles"
	DefaultTimeoutSeconds = 3600
)

type Definition struct {
	APIVersion     string    `yaml:"apiVersion" json:"apiVersion"`
	Kind           string    `yaml:"kind" json:"kind"`
	Name           string    `yaml:"name" json:"name"`
	Icon           string    `yaml:"icon,omitempty" json:"icon,omitempty"`
	Timeout        string    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	RunContext     string    `yaml:"run_context,omitempty" json:"run_context,omitempty"`
	Agent          *Agent    `yaml:"agent,omitempty" json:"agent,omitempty"`
	Triggers       []Trigger `yaml:"triggers" json:"triggers"`
	Steps          []Step    `yaml:"steps" json:"steps"`
	TimeoutSeconds int       `yaml:"-" json:"-"`
	RunsOnTask     *bool     `yaml:"-" json:"-"`
}

type Trigger struct {
	ID     string         `yaml:"id,omitempty" json:"id,omitempty"`
	Uses   TriggerUse     `yaml:"uses" json:"uses"`
	Config *TriggerConfig `yaml:"config,omitempty" json:"config,omitempty"`
}

type TriggerUse string

const (
	Manual             TriggerUse = "manual"
	TaskEntersStage    TriggerUse = "task_enters_stage"
	TaskLeavesStage    TriggerUse = "task_leaves_stage"
	PullRequestUpdated TriggerUse = "pull_request_updated"
	Scheduled          TriggerUse = "schedule"
	Webhook            TriggerUse = "webhook"
)

// TriggerConfig holds the schedule fields or the webhook ID. The schema
// allows only the fields for the trigger type.
type TriggerConfig struct {
	Cron      string `yaml:"cron,omitempty" json:"cron,omitempty"`
	Timezone  string `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	WebhookID string `yaml:"webhook_id,omitempty" json:"webhook_id,omitempty"`
}

type Step struct {
	Name             string   `yaml:"name" json:"name"`
	WorkingDirectory string   `yaml:"working_directory,omitempty" json:"working_directory,omitempty"`
	Bash             string   `yaml:"bash,omitempty" json:"bash,omitempty"`
	Prompt           string   `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Secrets          []string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type Agent struct {
	AdditionalInstructions *string      `yaml:"additional_instructions,omitempty" json:"additional_instructions,omitempty"`
	ThinkingLevel          string       `yaml:"thinking_level,omitempty" json:"thinking_level,omitempty"`
	Timeout                string       `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Harness                Harness      `yaml:"harness,omitempty" json:"harness,omitempty"`
	Model                  string       `yaml:"model,omitempty" json:"model,omitempty"`
	Credentials            *Credentials `yaml:"credentials,omitempty" json:"credentials,omitempty"`
	SkipRepositoryCheckout *bool        `yaml:"skip_repository_checkout,omitempty" json:"skip_repository_checkout,omitempty"`
	SkipSetupCommands      *bool        `yaml:"skip_setup_commands,omitempty" json:"skip_setup_commands,omitempty"`
}

type Harness string

const (
	SuperPlane Harness = "superplane"
	ClaudeCode Harness = "claude-code"
	Codex      Harness = "codex"
	OpenRouter Harness = "openrouter"
)

type Credentials struct {
	Source CredentialSource `yaml:"source" json:"source"`
	Name   string           `yaml:"name,omitempty" json:"name,omitempty"`
}

type CredentialSource string

const (
	Integration  CredentialSource = "integration"
	Subscription CredentialSource = "subscription"
	Hosted       CredentialSource = "hosted"
)

type Stage string

const (
	Backlog   Stage = "backlog"
	Implement Stage = "implement"
	Verify    Stage = "verify"
	Done      Stage = "done"
)

// Encode renders normalized YAML for a definition without an authored source.
func Encode(definition Definition) (string, error) {
	if definition.APIVersion == "" {
		definition.APIVersion = APIVersion
	}
	if definition.Kind == "" {
		definition.Kind = Kind
	}
	if definition.Icon == DefaultIcon {
		definition.Icon = ""
	}
	if definition.Timeout == "" && (definition.Agent == nil || definition.Agent.Timeout == "") && definition.TimeoutSeconds != 0 && definition.TimeoutSeconds != DefaultTimeoutSeconds {
		definition.Timeout = FormatDuration(definition.TimeoutSeconds)
	}
	return encodeYAML(definition)
}

// ReplaceName updates the display name while preserving the authored definition.
func ReplaceName(source, name string) (string, error) {
	document, err := Parse(source)
	if err != nil {
		return "", err
	}
	if document.Name == name {
		return source, nil
	}
	mappingProperty(document.root, "name").value.Value = name
	return encodeYAML(document.root)
}

// ReplaceExplicitAgent updates a stored override after the factory switches providers.
// A definition that inherits its agent keeps its exact authored source.
func ReplaceExplicitAgent(source string, agent *Agent) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(source), &root); err != nil {
		return "", err
	}
	if len(root.Content) == 0 {
		return "", fmt.Errorf("empty automation definition")
	}
	document := root.Content[0]
	field := mappingProperty(document, "agent")
	if field.value == nil {
		return source, nil
	}
	var previous Agent
	if err := field.value.Decode(&previous); err != nil {
		return "", err
	}
	if previous.Harness == "" && previous.Model == "" && previous.Credentials == nil {
		return source, nil
	}
	next := *agent
	next.AdditionalInstructions = previous.AdditionalInstructions
	next.ThinkingLevel = previous.ThinkingLevel
	next.Timeout = previous.Timeout
	next.SkipRepositoryCheckout = previous.SkipRepositoryCheckout
	next.SkipSetupCommands = previous.SkipSetupCommands
	agent = &next
	if previous.Harness == "" && previous.Credentials == nil {
		// Keep a model-only selection independent of the workspace's source.
		if agent.Model == "" && previous.SkipRepositoryCheckout == nil && previous.SkipSetupCommands == nil && previous.AdditionalInstructions == nil && previous.ThinkingLevel == "" && previous.Timeout == "" {
			i := slices.Index(document.Content, field.key)
			document.Content = slices.Delete(document.Content, i, i+2)
			return encodeYAML(&root)
		}
		agent = &Agent{Model: agent.Model, AdditionalInstructions: previous.AdditionalInstructions, ThinkingLevel: previous.ThinkingLevel, Timeout: previous.Timeout, SkipRepositoryCheckout: previous.SkipRepositoryCheckout, SkipSetupCommands: previous.SkipSetupCommands}
	}
	if reflect.DeepEqual(&previous, agent) {
		return source, nil
	}
	if err := field.value.Encode(agent); err != nil {
		return "", err
	}
	return encodeYAML(&root)
}

// RestoreStoredAgentSelection preserves authored choices when stored state needs a fresh export.
// This reads only agent selection from storage; Parse still rejects replaced input fields.
func RestoreStoredAgentSelection(generatedSource, storedSource string) (string, error) {
	var stored yaml.Node
	if err := yaml.Unmarshal([]byte(storedSource), &stored); err != nil {
		return "", fmt.Errorf("read stored automation agent: %w", err)
	}
	if len(stored.Content) == 0 {
		return generatedSource, nil
	}
	field := mappingProperty(stored.Content[0], "agent")
	if field.value == nil {
		return generatedSource, nil
	}
	var authored Agent
	if err := field.value.Decode(&authored); err != nil {
		return "", fmt.Errorf("read stored automation agent: %w", err)
	}
	if authored.Harness == "" && authored.Model == "" && authored.Credentials == nil {
		return generatedSource, nil
	}
	document, err := Parse(generatedSource)
	if err != nil {
		return "", err
	}
	if document.Agent == nil {
		document.Agent = &Agent{}
	}
	if mappingProperty(field.value, "harness").value != nil {
		document.Agent.Harness = authored.Harness
	}
	if mappingProperty(field.value, "model").value != nil {
		document.Agent.Model = authored.Model
	}
	if mappingProperty(field.value, "credentials").value != nil {
		document.Agent.Credentials = authored.Credentials
	}
	return Encode(document.Definition)
}

func encodeYAML(value any) (string, error) {
	var source bytes.Buffer
	encoder := yaml.NewEncoder(&source)
	// The YAML emitter's four-space indentation loses leading spaces in
	// literal blocks inside sequence items.
	encoder.SetIndent(2)
	if err := encoder.Encode(value); err != nil {
		return "", fmt.Errorf("encode automation definition: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("close automation definition: %w", err)
	}
	return source.String(), nil
}

// ValidateDispatch checks an automation bound to a line's execution plan.
func ValidateDispatch(document *Document) error {
	if len(document.Triggers) != 1 || document.Triggers[0].Uses != Manual {
		return document.Diagnostic("triggers", "A phase requires exactly one manual trigger.")
	}
	if document.RunContext == "workspace" {
		return document.Diagnostic("run_context", "A phase runs on a task.")
	}
	return nil
}

func ValidateStage(document *Document, stage Stage) error {
	switch stage {
	case Backlog, Implement, Verify, Done:
	default:
		if !strings.HasPrefix(string(stage), "custom-") || len(stage) <= len("custom-") {
			return document.Diagnostic("$", "Select a stage in this factory.")
		}
	}
	for index, trigger := range document.Triggers {
		triggerPath := itemPath("triggers", index) + ".uses"
		if trigger.Uses == PullRequestUpdated && stage != Verify {
			return document.Diagnostic(triggerPath, "The pull_request_updated trigger requires Verify.")
		}
		if trigger.Uses == TaskEntersStage && stage == Verify {
			return document.Diagnostic(triggerPath, "Use pull_request_updated in Verify, or select Backlog, Implement, or Done.")
		}
	}
	return nil
}
