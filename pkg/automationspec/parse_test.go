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
  - uses: manual
`

const bashStep = `steps:
  - name: Inspect
    bash: git status --short
`

func TestParseKeepsStepOrderAndText(t *testing.T) {
	source := definitionHeader + `# Comments are allowed in the source.
triggers:
  - uses: schedule
    config:
      cron: "0 9 ? * 1-5"
      timezone: "5.5"
  - uses: schedule
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
		document.Triggers[0].Config.Timezone != "5.5" || document.Triggers[1].Config.Timezone != "0" {
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
	document, err := automationspec.Parse(definitionHeader + "triggers:\n  - uses: webhook\n    config:\n      webhook_id: " + id + "\n" + bashStep)
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

func TestParseAcceptsDurationTimeout(t *testing.T) {
	document, err := automationspec.Parse(definitionHeader + "timeout: 15m\n" + manualTrigger + bashStep)
	if err != nil {
		t.Fatal(err)
	}
	if document.TimeoutSeconds != 900 {
		t.Fatalf("expected a 900-second timeout, got %d", document.TimeoutSeconds)
	}
}

func TestParseRejectsInvalidDefinitionAtItsSource(t *testing.T) {
	schedule := "triggers:\n  - uses: schedule\n    config:\n      cron: \"0 9 * * *\"\n"
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
		{"quoted number", definitionHeader + "timeout: '900'\n" + manualTrigger + bashStep, "timeout", 4, 10},
		{"timeout over integer range", definitionHeader + "timeout: 18446744073709551615\n" + manualTrigger + bashStep, "timeout", 4, 10},
		{"out of range timeout", definitionHeader + "timeout: 86401s\n" + manualTrigger + bashStep, "timeout", 4, 10},
		{"generic automatic trigger", definitionHeader + "triggers: [{uses: automatic}]\n" + bashStep, "triggers[0].uses", 4, 19},
		{"event configuration", definitionHeader + manualTrigger + "    config: {}\n" + bashStep, "triggers[0].config", 6, 13},
		{"duplicate event", definitionHeader + manualTrigger + "  - uses: manual\n" + bashStep, "triggers[1].uses", 6, 11},
		{"duplicate trigger ID", definitionHeader + "triggers:\n  - {id: same, uses: manual}\n  - {id: same, uses: task_enters_stage}\n" + bashStep, "triggers[1].id", 6, 10},
		{"invalid cron", definitionHeader + strings.Replace(schedule, "0 9 * * *", "0 99 * * *", 1) + bashStep, "triggers[0].config.cron", 7, 13},
		{"out of range offset", definitionHeader + schedule + "      timezone: \"14.25\"\n" + bashStep, "triggers[0].config.timezone", 8, 17},
		{"nonfinite offset", definitionHeader + schedule + "      timezone: .nan\n" + bashStep, "triggers[0].config.timezone", 8, 17},
		{"blank bash", definitionHeader + manualTrigger + strings.Replace(bashStep, "git status --short", "\"  \"", 1), "steps[0].bash", 8, 11},
		{"two step actions", definitionHeader + manualTrigger + bashStep + "    prompt: Summarize changes\n", "steps[0].prompt", 9, 13},
		{"secrets on prompt", definitionHeader + manualTrigger + "steps:\n  - name: Inspect\n    prompt: Inspect\n    secrets: [TOKEN]\n", "steps[0].secrets", 9, 14},
		{"duplicate secrets", definitionHeader + manualTrigger + bashStep + "    secrets: [TOKEN, TOKEN]\n", "steps[0].secrets[1]", 9, 22},
		{"invalid working directory type", definitionHeader + manualTrigger + bashStep + "    working_directory: false\n", "steps[0].working_directory", 9, 24},
		{"name on subscription", definitionHeader + manualTrigger + bashStep + "agent: {harness: codex, credentials: {source: subscription, name: shared}}\n", "agent.credentials.name", 9, 67},
		{"missing integration name", definitionHeader + manualTrigger + bashStep + "agent: {harness: codex, credentials: {source: integration}}\n", "agent.credentials.name", 9, 38},
		{"hosted credentials override", definitionHeader + manualTrigger + bashStep + "agent: {harness: superplane, credentials: {source: hosted}}\n", "agent.credentials", 9, 43},
		{"extra document", definitionHeader + manualTrigger + bashStep + "---\nname: extra\n", "$", 9, 1},
		{"webhook without ID", definitionHeader + "triggers:\n  - uses: webhook\n    config: {cron: \"0 9 * * *\"}\n" + bashStep, "triggers[0].config.webhook_id", 6, 13},
		{"invalid webhook ID", definitionHeader + "triggers:\n  - uses: webhook\n    config: {webhook_id: jira}\n" + bashStep, "triggers[0].config.webhook_id", 6, 26},
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
			source := definitionHeader + "triggers:\n  - uses: " + test.use + "\n" + bashStep
			document, err := automationspec.Parse(source)
			if err != nil {
				t.Fatal(err)
			}
			if err := automationspec.ValidateStage(document, test.validStage); err != nil {
				t.Fatal(err)
			}
			err = automationspec.ValidateStage(document, test.invalidStage)
			var diagnostic *automationspec.Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Path != "triggers[0].uses" ||
				diagnostic.Line != 5 || diagnostic.Column != 11 || !strings.Contains(diagnostic.Message, "Verify") {
				t.Fatalf("unexpected diagnostic: %v", err)
			}
		})
	}
}

func TestParsePortableExecutionSettings(t *testing.T) {
	for _, value := range []struct {
		text    string
		seconds int
	}{{"2h", 7200}, {"90m", 5400}, {"30s", 30}, {"0s", 3600}} {
		t.Run(value.text, func(t *testing.T) {
			document, err := automationspec.Parse(definitionHeader + "timeout: " + value.text + "\nagent: {skip_setup_commands: false, skip_repository_checkout: true}\n" + manualTrigger + bashStep)
			if err != nil {
				t.Fatal(err)
			}
			if document.TimeoutSeconds != value.seconds || document.Agent.SkipSetupCommands == nil || *document.Agent.SkipSetupCommands || document.Agent.SkipRepositoryCheckout == nil || !*document.Agent.SkipRepositoryCheckout {
				t.Fatalf("execution settings changed: %+v", document.Definition)
			}
			encoded, err := automationspec.Encode(document.Definition)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(encoded, "skip_setup_commands: false") || strings.Contains(encoded, "icon:") {
				t.Fatalf("unexpected generated defaults: %s", encoded)
			}
		})
	}
	t.Run("agent timeout scope", func(t *testing.T) {
		document, err := automationspec.Parse(definitionHeader + "agent: {timeout: 120m}\n" + manualTrigger + bashStep)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := automationspec.Encode(document.Definition)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(encoded, "timeout:") != 1 {
			t.Fatalf("agent timeout duplicated into automation scope: %s", encoded)
		}
		reparsed, err := automationspec.Parse(encoded)
		if err != nil {
			t.Fatal(err)
		}
		if reparsed.Timeout != "" || reparsed.Agent.Timeout != "2h" || reparsed.TimeoutSeconds != 7200 {
			t.Fatalf("timeout scope changed: %+v", reparsed.Definition)
		}
	})
	for _, value := range []string{"-1s", "0.5s", "1ns", "25h", "900", "999999999999999999999h"} {
		t.Run(value, func(t *testing.T) {
			_, err := automationspec.Parse(definitionHeader + "timeout: " + value + "\n" + manualTrigger + bashStep)
			var diagnostic *automationspec.Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Path != "timeout" {
				t.Fatalf("expected timeout diagnostic, got %v", err)
			}
		})
	}
}

func TestParseRejectsReplacedFields(t *testing.T) {
	for _, test := range []struct{ source, path, replacement string }{
		{definitionHeader + "timeout_seconds: 30\n" + manualTrigger + bashStep, "timeout_seconds", "timeout"},
		{definitionHeader + "runs_on_task: true\n" + manualTrigger + bashStep, "runs_on_task", "run_context"},
		{definitionHeader + "skip_setup_commands: false\n" + manualTrigger + bashStep, "skip_setup_commands", "agent.skip_setup_commands"},
		{definitionHeader + "triggers: [{use: manual}]\n" + bashStep, "triggers[0].use", "uses"},
		{definitionHeader + "triggers: [{uses: schedule, config: {cron: '0 9 * * *', utc_offset: 1}}]\n" + bashStep, "triggers[0].config.utc_offset", "timezone"},
	} {
		t.Run(test.path, func(t *testing.T) {
			_, err := automationspec.Parse(test.source)
			var diagnostic *automationspec.Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Path != test.path || !strings.Contains(diagnostic.Message, test.replacement) || diagnostic.Line < 1 || diagnostic.Column < 1 {
				t.Fatalf("expected replacement at %s, got %v", test.path, err)
			}
		})
	}
}

func TestParseResolvesExecutionContext(t *testing.T) {
	webhook := "{uses: webhook, config: {webhook_id: 1f0c3c55-4a8f-4a8e-9a43-2a7d8f6b6f10}}"
	for _, test := range []struct {
		name, context, triggers, resolved string
		valid                             bool
	}{
		{"schedule and manual", "", "[{uses: schedule, config: {cron: '0 9 * * *', timezone: '-3.5'}}, {uses: manual}]", "workspace", true},
		{"task event", "", "[{uses: task_enters_stage}]", "dynamic", true},
		{"webhook", "", "[" + webhook + "]", "workspace", true},
		{"webhook and task event", "", "[" + webhook + ", {uses: task_enters_stage}]", "dynamic", true},
		{"webhook and manual", "", "[" + webhook + ", {uses: manual}]", "dynamic", true},
		{"webhook and workspace manual", "workspace", "[" + webhook + ", {uses: manual}]", "workspace", true},
		{"manual task", "task", "[{uses: manual}]", "task", true},
		{"manual workspace", "workspace", "[{uses: manual}]", "workspace", true},
		{"task schedule", "task", "[{uses: schedule, config: {cron: '0 9 * * *'}}]", "", false},
		{"task webhook", "task", "[" + webhook + "]", "", false},
		{"workspace task event", "workspace", "[{uses: task_enters_stage}]", "", false},
		{"mixed schedule contexts", "", "[{uses: task_enters_stage}, {uses: schedule, config: {cron: '0 9 * * *'}}]", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := definitionHeader
			if test.context != "" {
				source += "run_context: " + test.context + "\n"
			}
			document, err := automationspec.Parse(source + "triggers: " + test.triggers + "\n" + bashStep)
			if !test.valid {
				if err == nil {
					t.Fatal("expected incompatible context diagnostic")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			resolved := "dynamic"
			if document.RunsOnTask != nil {
				resolved = "workspace"
				if *document.RunsOnTask {
					resolved = "task"
				}
			}
			if resolved != test.resolved {
				t.Fatalf("execution context = %s, want %s", resolved, test.resolved)
			}
		})
	}
}
