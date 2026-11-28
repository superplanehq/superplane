package automationspec_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/automationspec"
)

const definitionHeader = `apiVersion: v2
kind: Automation
name: Inspect a task
`

const manualTrigger = `triggers:
  - use: manual
`

const bashStep = `steps:
  - name: Inspect
    bash: git status --short
`

func TestParseKeepsStepOrderAndText(t *testing.T) {
	source := definitionHeader + `# Comments are allowed in the source.
triggers:
  - use: schedule
    config:
      cron: "0 9 ? * 1-5"
      utc_offset: 5.5
  - use: schedule
    config:
      cron: "0 30 16 * * 5"
steps:
  - name: "Inspect"
    bash: |
      git status --short
        # Indentation and the final newline matter.
    secrets: [REPORT_TOKEN]
  - name: Inspect
    prompt: |-
        Summarize the change.
          Include test results.
`
	document, err := automationspec.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if document.Icon != "sparkles" || document.TimeoutSeconds != 3600 || document.Agent != nil {
		t.Fatalf("unexpected defaults: %+v", document.Definition)
	}
	if len(document.Triggers) != 2 || document.Triggers[0].ID != "" || document.Triggers[1].ID != "" ||
		document.Triggers[0].Config.UTCOffset != 5.5 || document.Triggers[1].Config.UTCOffset != 0 {
		t.Fatalf("unexpected schedules: %+v", document.Triggers)
	}
	if len(document.Steps) != 2 || document.Steps[0].Bash != "git status --short\n  # Indentation and the final newline matter.\n" ||
		document.Steps[1].Prompt != "Summarize the change.\n  Include test results." {
		t.Fatalf("unexpected steps: %+v", document.Steps)
	}
	normalized, err := automationspec.Encode(document.Definition)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := automationspec.Parse(normalized)
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.Steps[0].Bash != document.Steps[0].Bash || reparsed.Steps[1].Prompt != document.Steps[1].Prompt {
		t.Fatalf("normalized source changed step text: %s", normalized)
	}
}

func TestParseKeepsWebhookTriggers(t *testing.T) {
	id := "1f0c3c55-4a8f-4a8e-9a43-2a7d8f6b6f10"
	document, err := automationspec.Parse(definitionHeader + "triggers:\n  - use: webhook\n    config:\n      webhook_id: " + id + "\n" + bashStep)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := automationspec.Encode(document.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(normalized, "webhook_id: "+id) || strings.Contains(normalized, "cron") {
		t.Fatalf("unexpected webhook trigger: %s", normalized)
	}
	if _, err := automationspec.Parse(normalized); err != nil {
		t.Fatal(err)
	}
}

func TestParseKeepsIntegrationTriggers(t *testing.T) {
	id := "1f0c3c55-4a8f-4a8e-9a43-2a7d8f6b6f10"
	source := definitionHeader + "triggers:\n  - use: integration\n    config:\n      integration_id: " + id +
		"\n      trigger: github.onPullRequest\n      configuration:\n        repository: acme/app\n        actions: [closed]\n" + bashStep
	document, err := automationspec.Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	config := document.Triggers[0].Config
	if config.IntegrationID != id || config.Trigger != "github.onPullRequest" || config.Configuration["repository"] != "acme/app" {
		t.Fatalf("unexpected integration trigger: %+v", config)
	}
	normalized, err := automationspec.Encode(document.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := automationspec.Parse(normalized); err != nil || !strings.Contains(normalized, "- closed") {
		t.Fatalf("normalized integration trigger did not parse: %v\n%s", err, normalized)
	}
}

func TestParseAcceptsIntegralTimeout(t *testing.T) {
	document, err := automationspec.Parse(definitionHeader + "timeout_seconds: 900.0\n" + manualTrigger + bashStep)
	if err != nil {
		t.Fatal(err)
	}
	if document.TimeoutSeconds != 900 {
		t.Fatalf("expected a 900-second timeout, got %d", document.TimeoutSeconds)
	}
}

func TestParseRejectsInvalidDefinitionAtItsSource(t *testing.T) {
	schedule := "triggers:\n  - use: schedule\n    config:\n      cron: \"0 9 * * *\"\n"
	cases := []struct {
		name   string
		source string
		path   string
		line   int
		column int
	}{
		{"unknown nested key", definitionHeader + manualTrigger + bashStep + "    command: pwd\n", "steps[0].command", 9, 5},
		{"duplicate nested key", definitionHeader + manualTrigger + bashStep + "    name: Again\n", "steps[0].name", 9, 5},
		{"missing step name", definitionHeader + manualTrigger + "steps:\n  - bash: pwd\n", "steps[0].name", 7, 5},
		{"null icon", definitionHeader + "icon: null\n" + manualTrigger + bashStep, "icon", 4, 7},
		{"wrong version case", strings.Replace(definitionHeader, "v2", "V2", 1) + manualTrigger + bashStep, "apiVersion", 1, 13},
		{"quoted number", definitionHeader + "timeout_seconds: '900'\n" + manualTrigger + bashStep, "timeout_seconds", 4, 18},
		{"timeout over integer range", definitionHeader + "timeout_seconds: 18446744073709551615\n" + manualTrigger + bashStep, "timeout_seconds", 4, 18},
		{"out of range timeout", definitionHeader + "timeout_seconds: 86401\n" + manualTrigger + bashStep, "timeout_seconds", 4, 18},
		{"generic automatic trigger", definitionHeader + "triggers: [{use: automatic}]\n" + bashStep, "triggers[0].use", 4, 18},
		{"event configuration", definitionHeader + manualTrigger + "    config: {}\n" + bashStep, "triggers[0].config", 6, 13},
		{"duplicate event", definitionHeader + manualTrigger + "  - use: manual\n" + bashStep, "triggers[1].use", 6, 10},
		{"duplicate trigger ID", definitionHeader + "triggers:\n  - {id: same, use: manual}\n  - {id: same, use: task_enters_stage}\n" + bashStep, "triggers[1].id", 6, 10},
		{"invalid cron", definitionHeader + strings.Replace(schedule, "0 9 * * *", "0 99 * * *", 1) + bashStep, "triggers[0].config.cron", 7, 13},
		{"out of range offset", definitionHeader + schedule + "      utc_offset: 14.25\n" + bashStep, "triggers[0].config.utc_offset", 8, 19},
		{"nonfinite offset", definitionHeader + schedule + "      utc_offset: .nan\n" + bashStep, "triggers[0].config.utc_offset", 8, 19},
		{"blank bash", definitionHeader + manualTrigger + strings.Replace(bashStep, "git status --short", "\"  \"", 1), "steps[0].bash", 8, 11},
		{"two step actions", definitionHeader + manualTrigger + bashStep + "    prompt: Summarize changes\n", "steps[0].prompt", 9, 13},
		{"secrets on prompt", definitionHeader + manualTrigger + "steps:\n  - name: Inspect\n    prompt: Inspect\n    secrets: [TOKEN]\n", "steps[0].secrets", 9, 14},
		{"duplicate secrets", definitionHeader + manualTrigger + bashStep + "    secrets: [TOKEN, TOKEN]\n", "steps[0].secrets[1]", 9, 22},
		{"invalid working directory type", definitionHeader + manualTrigger + bashStep + "    working_directory: false\n", "steps[0].working_directory", 9, 24},
		{"name on subscription", definitionHeader + manualTrigger + bashStep + "agent: {harness: codex, credentials: {source: subscription, name: shared}}\n", "agent.credentials.name", 9, 67},
		{"missing integration name", definitionHeader + manualTrigger + bashStep + "agent: {harness: codex, credentials: {source: integration}}\n", "agent.credentials.name", 9, 38},
		{"hosted credentials override", definitionHeader + manualTrigger + bashStep + "agent: {harness: superplane, credentials: {source: hosted}}\n", "agent.credentials", 9, 43},
		{"extra document", definitionHeader + manualTrigger + bashStep + "---\nname: extra\n", "$", 9, 1},
		{"webhook without ID", definitionHeader + "triggers:\n  - use: webhook\n    config: {cron: \"0 9 * * *\"}\n" + bashStep, "triggers[0].config.webhook_id", 6, 13},
		{"invalid webhook ID", definitionHeader + "triggers:\n  - use: webhook\n    config: {webhook_id: jira}\n" + bashStep, "triggers[0].config.webhook_id", 6, 26},
		{"integration without trigger", definitionHeader + "triggers:\n  - use: integration\n    config: {integration_id: 1f0c3c55-4a8f-4a8e-9a43-2a7d8f6b6f10}\n" + bashStep, "triggers[0].config.trigger", 6, 13},
		{"invalid integration ID", definitionHeader + "triggers:\n  - use: integration\n    config: {integration_id: github, trigger: github.onPullRequest}\n" + bashStep, "triggers[0].config.integration_id", 6, 30},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document, err := automationspec.Parse(test.source)
			var diagnostic *automationspec.Diagnostic
			if document != nil || !errors.As(err, &diagnostic) {
				t.Fatalf("expected a diagnostic, got document=%+v, err=%v", document, err)
			}
			if diagnostic.Path != test.path || diagnostic.Line != test.line || diagnostic.Column != test.column ||
				diagnostic.Message == "" {
				t.Fatalf("expected %s at %d:%d, got %+v", test.path, test.line, test.column, diagnostic)
			}
		})
	}
}

func TestMalformedSourceReturnsADocumentDiagnostic(t *testing.T) {
	source := definitionHeader + manualTrigger + bashStep + "  - name: [unclosed\n"
	_, err := automationspec.Parse(source)
	var diagnostic *automationspec.Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Path != "$" || diagnostic.Line < 1 ||
		diagnostic.Column < 1 || !strings.Contains(diagnostic.Message, "YAML syntax") {
		t.Fatalf("unexpected YAML syntax diagnostic: %v", err)
	}
}

func TestValidateStagePointsAtTheTriggerType(t *testing.T) {
	for _, test := range []struct {
		use          string
		validStage   automationspec.Stage
		invalidStage automationspec.Stage
	}{
		{"pull_request_updated", automationspec.Verify, automationspec.Backlog},
		{"task_enters_stage", automationspec.Backlog, automationspec.Verify},
	} {
		t.Run(test.use, func(t *testing.T) {
			source := definitionHeader + "triggers:\n  - use: " + test.use + "\n" + bashStep
			document, err := automationspec.Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			if err := automationspec.ValidateStage(document, test.validStage); err != nil {
				t.Fatal(err)
			}
			err = automationspec.ValidateStage(document, test.invalidStage)
			var diagnostic *automationspec.Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Path != "triggers[0].use" ||
				diagnostic.Line != 5 || diagnostic.Column != 10 || !strings.Contains(diagnostic.Message, "Verify") {
				t.Fatalf("unexpected diagnostic: %v", err)
			}
		})
	}
}
