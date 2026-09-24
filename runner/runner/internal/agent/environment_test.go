package agent

import (
	"strings"
	"testing"
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
