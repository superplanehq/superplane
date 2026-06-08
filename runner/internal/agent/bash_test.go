package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildBashProgram(t *testing.T) {
	t.Parallel()
	chain := json.RawMessage(`{"GitHub PR":{"data":{"number":42}}}`)
	prog, err := buildBashProgram(`#!/usr/bin/env bash
main() {
  local payload="$1"
  echo "{\"pr\": 42}"
}
`, chain)
	if err != nil {
		t.Fatal(err)
	}
	s := string(prog)
	for _, want := range []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"payload='",
		`"GitHub PR"`,
		"main() {",
		`main "$payload"`,
		"SUPERPLANE_RESULT_FILE",
		"main(payload) is required",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("program missing %q:\n%s", want, s)
		}
	}
	if strings.Count(s, "#!/usr/bin/env bash") != 1 {
		t.Fatalf("expected exactly one shebang:\n%s", s)
	}
}

func TestBuildBashProgramRejectsInvalidChain(t *testing.T) {
	t.Parallel()
	_, err := buildBashProgram(`main() { echo ok; }`, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildBashProgramRejectsEmptyScript(t *testing.T) {
	t.Parallel()
	_, err := buildBashProgram("  ", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStripUserShebang(t *testing.T) {
	t.Parallel()
	got := stripUserShebang("#!/bin/bash\nmain() { true; }\n")
	if got != "main() { true; }" {
		t.Fatalf("got %q", got)
	}
}

func TestBashSingleQuoted(t *testing.T) {
	t.Parallel()
	if got := bashSingleQuoted(`it's`); got != `'it'\''s'` {
		t.Fatalf("got %q", got)
	}
}
