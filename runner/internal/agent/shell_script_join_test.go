package agent

import (
	"strings"
	"testing"
)

func TestCombineShellDirectives(t *testing.T) {
	t.Parallel()
	s, ok := combineShellDirectives([]string{"echo a", "", "  echo b  "})
	if !ok {
		t.Fatal("expected ok")
	}
	if strings.Contains(s, "set -e") {
		t.Fatalf("should not prepend set -e; got %q", s)
	}
	if want := "echo a && echo b"; s != want {
		t.Fatalf("want %q, got %q", want, s)
	}

	if _, ok := combineShellDirectives(nil); ok {
		t.Fatal("nil scripts should be invalid")
	}
	if _, ok := combineShellDirectives([]string{"", "  "}); ok {
		t.Fatal("all empty should be invalid")
	}
}
