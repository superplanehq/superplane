package agent

import (
	"strings"
	"testing"
)

func TestPtyExitAliasBootstrap(t *testing.T) {
	t.Parallel()

	if !strings.Contains(ptyExitAliasBootstrap, "shopt -s expand_aliases") {
		t.Fatalf("bootstrap missing expand_aliases: %q", ptyExitAliasBootstrap)
	}
	if !strings.Contains(ptyExitAliasBootstrap, "alias exit=return") {
		t.Fatalf("bootstrap missing exit alias: %q", ptyExitAliasBootstrap)
	}
	if strings.Contains(ptyExitAliasBootstrap, "\n") {
		t.Fatalf("bootstrap should be a single shell line: %q", ptyExitAliasBootstrap)
	}
}
