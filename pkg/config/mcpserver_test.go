package config

import "testing"

func TestMCPServerEnabled(t *testing.T) {
	t.Setenv(EnvMCPServer, "")
	if MCPServerEnabled() {
		t.Fatal("empty value must leave the server off")
	}

	t.Setenv(EnvMCPServer, "no")
	if MCPServerEnabled() {
		t.Fatal("no must leave the server off")
	}

	t.Setenv(EnvMCPServer, "yes")
	if !MCPServerEnabled() {
		t.Fatal("yes must turn the server on")
	}
}
