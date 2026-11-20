// Package automationspec defines the public v2 custom automation language.
package automationspec

import (
	"bytes"
	"fmt"

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
	Icon           string    `yaml:"icon,omitempty" json:"icon"`
	TimeoutSeconds int       `yaml:"timeout_seconds,omitempty" json:"timeout_seconds"`
	RunsOnTask     *bool     `yaml:"runs_on_task,omitempty" json:"runs_on_task,omitempty"`
	Agent          *Agent    `yaml:"agent,omitempty" json:"agent,omitempty"`
	Triggers       []Trigger `yaml:"triggers" json:"triggers"`
	Steps          []Step    `yaml:"steps" json:"steps"`
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
	Name    string   `yaml:"name" json:"name"`
	Bash    string   `yaml:"bash,omitempty" json:"bash,omitempty"`
	Prompt  string   `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Secrets []string `yaml:"secrets,omitempty" json:"secrets,omitempty"`
}

type Agent struct {
	Harness     Harness      `yaml:"harness" json:"harness"`
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
	var source bytes.Buffer
	encoder := yaml.NewEncoder(&source)
	// The YAML emitter's four-space indentation loses leading spaces in
	// literal blocks inside sequence items.
	encoder.SetIndent(2)
	if err := encoder.Encode(definition); err != nil {
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
		return document.Diagnostic("$", "Select Backlog, Implement, Verify, or Done.")
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
