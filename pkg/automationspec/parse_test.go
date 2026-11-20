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
  - id: morning
    use: schedule
    config:
      cron: "0 9 * * 1-5"
      utc_offset: 5.5
  - id: afternoon
    use: schedule
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
	if len(document.Triggers) != 2 || document.Triggers[0].ID != "morning" ||
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
		{"missing schedule ID", definitionHeader + schedule + "  - {id: later, use: schedule, config: {cron: '0 12 * * *'}}\n" + bashStep, "triggers[0].id", 5, 5},
		{"invalid cron", definitionHeader + strings.Replace(schedule, "0 9 * * *", "0 99 * * *", 1) + bashStep, "triggers[0].config.cron", 7, 13},
		{"out of range offset", definitionHeader + schedule + "      utc_offset: 14.25\n" + bashStep, "triggers[0].config.utc_offset", 8, 19},
		{"nonfinite offset", definitionHeader + schedule + "      utc_offset: .nan\n" + bashStep, "triggers[0].config.utc_offset", 8, 19},
		{"blank bash", definitionHeader + manualTrigger + strings.Replace(bashStep, "git status --short", "\"  \"", 1), "steps[0].bash", 8, 11},
		{"two step actions", definitionHeader + manualTrigger + bashStep + "    prompt: Summarize changes\n", "steps[0].prompt", 9, 13},
		{"secrets on prompt", definitionHeader + manualTrigger + "steps:\n  - name: Inspect\n    prompt: Inspect\n    secrets: [TOKEN]\n", "steps[0].secrets", 9, 14},
		{"duplicate secrets", definitionHeader + manualTrigger + bashStep + "    secrets: [TOKEN, TOKEN]\n", "steps[0].secrets[1]", 9, 22},
		{"removed working directory", definitionHeader + manualTrigger + bashStep + "    working_directory: repo\n", "steps[0].working_directory", 9, 5},
		{"non-hosted agent", definitionHeader + manualTrigger + bashStep + "agent: {harness: codex}\n", "agent.harness", 9, 18},
		{"agent credentials", definitionHeader + manualTrigger + bashStep + "agent: {harness: superplane, credentials: {source: subscription}}\n", "agent.credentials", 9, 30},
		{"extra document", definitionHeader + manualTrigger + bashStep + "---\nname: extra\n", "$", 9, 1},
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
