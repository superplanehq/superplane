package agent

import (
	"strings"
	"testing"
)

func TestWrapSourcedDirectiveAliasesExit(t *testing.T) {
	t.Parallel()

	got := wrapSourcedDirective("echo hello\nexit 1\necho there\n")
	for _, want := range []string{
		"shopt -s expand_aliases\n",
		"alias exit=return\n",
		"echo hello\n",
		"exit 1\n",
		"echo there\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("wrapSourcedDirective missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "mktemp") || strings.Contains(got, "(\n") {
		t.Fatalf("wrap should stay in-process (no subshell/mktemp): %q", got)
	}
}
