package automationspec

import (
	"fmt"
	"math"
	"path"
	"slices"
	"strings"

	"github.com/robfig/cron/v3"
)

var icons = []string{
	"sparkles", "shield-check", "search", "bug", "file-code", "code", "terminal",
	"eye", "gauge", "git-pull-request", "cpu", "circle-check", "activity", "alarm-clock",
	"archive", "bell", "bolt", "book", "box", "boxes", "braces", "calendar", "camera",
	"chart-line", "check", "clock", "cloud", "cog", "component", "container", "database",
	"factory", "file-json", "file-text", "fingerprint", "flag", "folder", "git-branch",
	"git-commit", "git-merge", "globe", "key", "layers", "lock", "message-square",
	"network", "package", "rocket",
}

func (d *Document) validateDefinition() error {
	switch {
	case d.APIVersion != APIVersion:
		return d.Diagnostic("apiVersion", "Set apiVersion to v2.")
	case d.Kind != Kind:
		return d.Diagnostic("kind", "Set kind to Automation.")
	case blank(d.Name):
		return d.Diagnostic("name", "Enter a name for the automation.")
	case d.has("icon") && !slices.Contains(icons, d.Icon):
		return d.Diagnostic("icon", "Enter an icon key from the automation icon picker.")
	case d.has("timeout_seconds") && (d.TimeoutSeconds < 1 || d.TimeoutSeconds > 86400):
		return d.Diagnostic("timeout_seconds", "Enter a timeout from 1 through 86400 seconds.")
	case len(d.Triggers) == 0:
		return d.Diagnostic("triggers", "Add at least one trigger.")
	case len(d.Steps) == 0:
		return d.Diagnostic("steps", "Add at least one step.")
	}
	if err := d.validateAgent(); err != nil {
		return err
	}
	if err := d.validateTriggers(); err != nil {
		return err
	}
	return d.validateSteps()
}

func (d *Document) validateAgent() error {
	agent := d.Agent
	if agent == nil {
		return nil
	}
	switch agent.Harness {
	case SuperPlane, ClaudeCode, Codex, OpenRouter:
	default:
		return d.Diagnostic("agent.harness", "Enter superplane, claude-code, codex, or openrouter.")
	}
	if d.has("agent.model") && blank(agent.Model) {
		return d.Diagnostic("agent.model", "Enter a model ID or omit model to use the default.")
	}
	if agent.Credentials == nil {
		return nil
	}
	if agent.Harness == SuperPlane {
		return d.Diagnostic("agent.credentials", "Remove credentials to use the workspace configuration for the superplane harness.")
	}
	switch agent.Credentials.Source {
	case Integration:
		if blank(agent.Credentials.Name) {
			return d.Diagnostic("agent.credentials.name", "Enter the name of the integration connection.")
		}
	case Subscription:
		if agent.Harness != Codex {
			return d.Diagnostic("agent.credentials.source", "Use subscription with the codex harness.")
		}
		if d.has("agent.credentials.name") {
			return d.Diagnostic("agent.credentials.name", "Remove name to use the configured subscription.")
		}
		if d.has("agent.model") {
			return d.Diagnostic("agent.model", "Remove model to use the default model for the subscription.")
		}
	default:
		return d.Diagnostic("agent.credentials.source", "Enter integration or subscription.")
	}
	return nil
}

func (d *Document) validateTriggers() error {
	scheduleCount := 0
	for _, trigger := range d.Triggers {
		if trigger.Use == Scheduled {
			scheduleCount++
		}
	}
	eventTypes := make(map[TriggerUse]bool)
	ids := make(map[string]bool)
	for index, trigger := range d.Triggers {
		triggerPath := itemPath("triggers", index)
		idPath := triggerPath + ".id"
		if d.has(idPath) || (trigger.Use == Scheduled && scheduleCount > 1) {
			if blank(trigger.ID) {
				return d.Diagnostic(idPath, "Enter a nonblank ID. Give each schedule an ID when you use multiple schedules.")
			}
			if ids[trigger.ID] {
				return d.Diagnostic(idPath, "Give each trigger a unique ID.")
			}
			ids[trigger.ID] = true
		}

		switch trigger.Use {
		case Manual, TaskEntersStage, PullRequestUpdated:
			if eventTypes[trigger.Use] {
				return d.Diagnostic(triggerPath+".use", "Use each event trigger once.")
			}
			eventTypes[trigger.Use] = true
			if trigger.Config != nil {
				return d.Diagnostic(triggerPath+".config", "Remove config from this event trigger. Schedule triggers accept config.")
			}
		case Scheduled:
			if err := d.validateSchedule(triggerPath+".config", trigger.Config); err != nil {
				return err
			}
		default:
			return d.Diagnostic(triggerPath+".use", "Enter manual, task_enters_stage, pull_request_updated, or schedule.")
		}
	}
	return nil
}

func (d *Document) validateSchedule(configPath string, config *Schedule) error {
	if config == nil {
		return d.Diagnostic(configPath, "Add config with a cron expression for this schedule.")
	}
	if err := validateCron(config.Cron); err != nil {
		return d.Diagnostic(configPath+".cron", err.Error())
	}
	if math.IsNaN(config.UTCOffset) || math.IsInf(config.UTCOffset, 0) || config.UTCOffset < -12 || config.UTCOffset > 14 {
		return d.Diagnostic(configPath+".utc_offset", "Enter a UTC offset from -12 through 14 hours.")
	}
	return nil
}

func validateCron(expression string) error {
	options := cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow
	switch len(strings.Fields(expression)) {
	case 5:
	case 6:
		options |= cron.Second
	default:
		return fmt.Errorf("enter a cron expression with five fields, or six with seconds first")
	}
	for _, char := range strings.TrimSpace(expression) {
		if !strings.ContainsRune("0123456789*,-/ abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", char) {
			return fmt.Errorf("remove %q from the cron expression", char)
		}
	}
	if _, err := cron.NewParser(options).Parse(expression); err != nil {
		return fmt.Errorf("correct the cron expression (%s)", err)
	}
	return nil
}

func (d *Document) validateSteps() error {
	for index, step := range d.Steps {
		stepPath := itemPath("steps", index)
		if blank(step.Name) {
			return d.Diagnostic(stepPath+".name", "Enter a name for this step.")
		}
		if err := d.validateStepAction(stepPath, step); err != nil {
			return err
		}
		if d.has(stepPath+".working_directory") && !validWorkingDirectory(step.WorkingDirectory) {
			return d.Diagnostic(stepPath+".working_directory", "Enter a path on one line. Keep a relative path within the runner's working directory.")
		}
		if err := d.validateStepSecrets(stepPath, step); err != nil {
			return err
		}
	}
	return nil
}

func (d *Document) validateStepAction(stepPath string, step Step) error {
	hasBash := d.has(stepPath + ".bash")
	hasPrompt := d.has(stepPath + ".prompt")
	switch {
	case hasBash && hasPrompt:
		return d.Diagnostic(stepPath+".prompt", "Keep one field for the step action. Remove bash or prompt.")
	case !hasBash && !hasPrompt:
		return d.Diagnostic(stepPath, "Add bash or prompt to this step.")
	case hasBash && blank(step.Bash):
		return d.Diagnostic(stepPath+".bash", "Enter a Bash script.")
	case hasPrompt && blank(step.Prompt):
		return d.Diagnostic(stepPath+".prompt", "Enter instructions for the agent.")
	default:
		return nil
	}
}

func (d *Document) validateStepSecrets(stepPath string, step Step) error {
	secretsPath := stepPath + ".secrets"
	if d.has(secretsPath) && !d.has(stepPath+".bash") {
		return d.Diagnostic(secretsPath, "Use secrets in Bash steps.")
	}
	names := make(map[string]bool)
	for index, name := range step.Secrets {
		namePath := itemPath(secretsPath, index)
		if blank(name) {
			return d.Diagnostic(namePath, "Enter an organization secret name.")
		}
		if names[name] {
			return d.Diagnostic(namePath, "Enter each organization secret name once per step.")
		}
		names[name] = true
	}
	return nil
}

func (d *Document) has(field string) bool {
	return d.locations[field] != nil
}

func blank(text string) bool {
	return strings.TrimSpace(text) == ""
}

func validWorkingDirectory(directory string) bool {
	if blank(directory) || strings.ContainsAny(directory, "\r\n\x00") {
		return false
	}
	cleaned := path.Clean(directory)
	return cleaned != ".." && !strings.HasPrefix(cleaned, "../")
}
