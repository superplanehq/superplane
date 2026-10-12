package agent

import (
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/runners/api"
)

func TestWithHomeEnvReplacesExistingHome(t *testing.T) {
	got := withHomeEnv([]string{"PATH=/bin", "HOME=/old", "USER=node"}, "/new/home")
	if !envContains(got, "HOME=/new/home") {
		t.Fatalf("missing new HOME: %#v", got)
	}
	if envContains(got, "HOME=/old") {
		t.Fatalf("old HOME still present: %#v", got)
	}
	if !envContains(got, "PATH=/bin") || !envContains(got, "USER=node") {
		t.Fatalf("lost other vars: %#v", got)
	}
}

func TestWithHomeEnvNilUsesProcessEnv(t *testing.T) {
	t.Setenv("HOME", "/process/home")
	got := withHomeEnv(nil, "/task/home")
	if !envContains(got, "HOME=/task/home") {
		t.Fatalf("missing task HOME: %#v", got)
	}
	if envContains(got, "HOME=/process/home") {
		t.Fatalf("process HOME still present: %#v", got)
	}
	foundPath := false
	for _, pair := range got {
		if strings.HasPrefix(pair, "PATH=") {
			foundPath = true
			break
		}
	}
	if !foundPath {
		t.Fatalf("expected process PATH in env: %#v", got)
	}
}

func TestUseRunnerBaseURLReplacesPlanningSessionURL(t *testing.T) {
	environment := []api.EnvironmentVariable{
		{Name: "SUPERPLANE_BASE_URL", Value: "https://example.ngrok.app"},
		{Name: "OTHER", Value: "value"},
	}

	got := useRunnerBaseURL(environment, "http://app:8000/")

	if got[0].Value != "http://app:8000" {
		t.Fatalf("SUPERPLANE_BASE_URL = %q", got[0].Value)
	}
	if got[1] != environment[1] {
		t.Fatalf("other environment changed: %#v", got)
	}
}

func TestUseRunnerBaseURLDoesNotAddPlanningSessionURL(t *testing.T) {
	environment := []api.EnvironmentVariable{{Name: "OTHER", Value: "value"}}

	got := useRunnerBaseURL(environment, "http://app:8000")

	if len(got) != 1 || got[0] != environment[0] {
		t.Fatalf("environment = %#v", got)
	}
}
