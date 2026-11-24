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
	APIVersion             string    `yaml:"apiVersion" json:"apiVersion"`
	Kind                   string    `yaml:"kind" json:"kind"`
	Name                   string    `yaml:"name" json:"name"`
	Icon                   string    `yaml:"icon,omitempty" json:"icon"`
	TimeoutSeconds         int       `yaml:"timeout_seconds,omitempty" json:"timeout_seconds"`
	RunsOnTask             *bool     `yaml:"runs_on_task,omitempty" json:"runs_on_task,omitempty"`
	SkipRepositoryCheckout *bool     `yaml:"skip_repository_checkout,omitempty" json:"skip_repository_checkout,omitempty"`
	SkipSetupCommands      *bool     `yaml:"skip_setup_commands,omitempty" json:"skip_setup_commands,omitempty"`
	Agent                  *Agent    `yaml:"agent,omitempty" json:"agent,omitempty"`
	Triggers               []Trigger `yaml:"triggers" json:"triggers"`
	Steps                  []Step    `yaml:"steps" json:"steps"`
}

type Trigger struct {
	ID     string     `yaml:"id,omitempty" json:"id,omitempty"`
	Use    TriggerUse `yaml:"use" json:"use"`
	Config *Schedule  `yaml:"config,omitempty" json:"config,omitempty"`
}

type TriggerUse string

const (
	Manual             TriggerUse = "manual"
	TaskEntersStage    TriggerUse = "task_enters_stage"
	TaskLeavesStage    TriggerUse = "task_leaves_stage"
	PullRequestUpdated TriggerUse = "pull_request_updated"
	Scheduled          TriggerUse = "schedule"
)

type Schedule struct {
	Cron      string  `yaml:"cron" json:"cron"`
	UTCOffset float64 `yaml:"utc_offset,omitempty" json:"utc_offset"`
}

type Step struct {
	Chat             string   `yaml:"chat,omitempty" json:"chat,omitempty"`
	Name             string   `yaml:"name" json:"name"`
	WorkingDirectory string   `yaml:"working_directory,omitempty" json:"working_directory,omitempty"`
	Bash             string   `yaml:"bash,omitempty" json:"bash,omitempty"`
	Prompt           string   `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Secrets          []string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type Agent struct {
	Harness     Harness      `yaml:"harness,omitempty" json:"harness,omitempty"`
	Model       string       `yaml:"model,omitempty" json:"model,omitempty"`
	Credentials *Credentials `yaml:"credentials,omitempty" json:"credentials,omitempty"`
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
	return encodeYAML(definition)
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
	if previous.Harness == "" && previous.Credentials == nil {
		// Keep a model-only selection independent of the workspace's source.
		if agent.Model == "" {
			i := slices.Index(document.Content, field.key)
			document.Content = slices.Delete(document.Content, i, i+2)
			return encodeYAML(&root)
		}
		agent = &Agent{Model: agent.Model}
	}
	if reflect.DeepEqual(&previous, agent) {
		return source, nil
	}
	if err := field.value.Encode(agent); err != nil {
		return "", err
	}
	return encodeYAML(&root)
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

func ValidateStage(document *Document, stage Stage) error {
	switch stage {
	case Backlog, Implement, Verify, Done:
	default:
		if !strings.HasPrefix(string(stage), "custom-") || len(stage) <= len("custom-") {
			return document.Diagnostic("$", "Select a stage in this factory.")
		}
	}
	for index, trigger := range document.Triggers {
		triggerPath := itemPath("triggers", index) + ".use"
		if trigger.Use == PullRequestUpdated && stage != Verify {
			return document.Diagnostic(triggerPath, "The pull_request_updated trigger requires Verify.")
		}
		if trigger.Use == TaskEntersStage && stage == Verify {
			return document.Diagnostic(triggerPath, "Use pull_request_updated in Verify, or select Backlog, Implement, or Done.")
		}
	}
	return nil
}
