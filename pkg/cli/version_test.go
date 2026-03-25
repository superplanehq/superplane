package cli

import "testing"

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"minor bump", "v0.13.0", "v0.14.0", true},
		{"patch bump", "v0.13.0", "v0.13.1", true},
		{"major bump", "v0.13.0", "v1.0.0", true},
		{"same version", "v0.13.0", "v0.13.0", false},
		{"current is newer minor", "v0.14.0", "v0.13.0", false},
		{"current is newer major", "v1.0.0", "v0.99.99", false},
		{"no v prefix", "0.13.0", "0.14.0", true},
		{"mixed prefixes", "v0.13.0", "0.14.0", true},
		{"multi-digit versions", "v0.9.0", "v0.10.0", true},
		{"current is newer patch", "v0.13.2", "v0.13.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNewerVersion(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestIsDevBuild(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "dev"
	if !isDevBuild() {
		t.Error("expected dev build when Version is 'dev'")
	}

	Version = ""
	if !isDevBuild() {
		t.Error("expected dev build when Version is empty")
	}

	Version = "v0.13.0"
	if isDevBuild() {
		t.Error("expected non-dev build when Version is 'v0.13.0'")
	}
}

func TestShouldStartUpdateCheck(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "v0.13.0"

	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{name: "no args", args: nil, expected: false},
		{name: "root help when no args", args: []string{}, expected: false},
		{name: "help subcommand", args: []string{"help"}, expected: false},
		{name: "help flag", args: []string{"--help"}, expected: false},
		{name: "short help flag", args: []string{"-h"}, expected: false},
		{name: "version subcommand", args: []string{"version"}, expected: false},
		{name: "version flag", args: []string{"--version"}, expected: false},
		{name: "completion command", args: []string{"completion", "zsh"}, expected: false},
		{name: "networked command", args: []string{"whoami"}, expected: true},
		{name: "networked subcommand with flags", args: []string{"canvases", "list", "-o", "json"}, expected: true},
		{name: "flag before local subcommand", args: []string{"-o", "json", "version"}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldStartUpdateCheck(tt.args)
			if result != tt.expected {
				t.Fatalf("ShouldStartUpdateCheck(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}
