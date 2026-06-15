package configurationfields

import (
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/configuration"
)

func TestIsCamelCaseFieldName(t *testing.T) {
	tests := map[string]bool{
		"myField":         true,
		"durationSeconds": true,
		"commands":        true,
		"my_field":        false,
		"ExecutionMode":   false,
		"docker_image":    false,
		"SCREAMING_SNAKE": false,
		"":                false,
	}

	for name, want := range tests {
		if got := isCamelCaseFieldName(name); got != want {
			t.Errorf("isCamelCaseFieldName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestCheckFieldsFlagsSnakeCaseNames(t *testing.T) {
	fields := []configuration.Field{
		{
			Name: "execution_mode",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemDefinition: &configuration.ListItemDefinition{
						Schema: []configuration.Field{
							{Name: "setup_commands", Type: configuration.FieldTypeString},
						},
					},
				},
			},
			VisibilityConditions: []configuration.VisibilityCondition{
				{Field: "docker_image"},
			},
		},
		{Name: "commands", Type: configuration.FieldTypeText},
	}

	var issues []Issue
	checkFields("action", "runner", "configuration", fields, &issues)

	if len(issues) != 3 {
		t.Fatalf("expected 3 issues, got %d: %#v", len(issues), issues)
	}

	names := make([]string, 0, len(issues))
	for _, issue := range issues {
		names = append(names, issue.Field)
	}

	for _, want := range []string{"execution_mode", "setup_commands", "docker_image"} {
		found := false
		for _, name := range names {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing expected field %q in %#v", want, names)
		}
	}
}

func TestRunFindsKnownSnakeCaseFields(t *testing.T) {
	issues, err := Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	found := false
	for _, issue := range issues {
		if issue.OwnerName == "runner" && issue.Field == "execution_mode" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected runner execution_mode issue, got: %#v", issues)
	}
}

func TestIssueString(t *testing.T) {
	issue := Issue{
		OwnerKind: "action",
		OwnerName: "runner",
		Path:      "configuration[0]",
		Field:     "execution_mode",
	}

	text := issue.String()
	if !strings.Contains(text, "execution_mode") {
		t.Fatalf("issue string missing field: %q", text)
	}
	if !strings.Contains(text, "executionMode") {
		t.Fatalf("issue string missing suggestion: %q", text)
	}
}
