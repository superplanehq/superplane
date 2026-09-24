package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildJavaScriptProgram(t *testing.T) {
	t.Parallel()
	chain := json.RawMessage(`{"GitHub PR":{"data":{"number":42}}}`)
	prog, err := buildJavaScriptProgram(`
function main() {
  return { pr: $['GitHub PR'].data.number };
}
`, chain)
	if err != nil {
		t.Fatal(err)
	}
	s := string(prog)
	for _, want := range []string{
		"globalThis.$ = ",
		`"GitHub PR"`,
		"function main()",
		"SUPERPLANE_RESULT_FILE",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("program missing %q:\n%s", want, s)
		}
	}
}

func TestBuildJavaScriptProgramDoesNotRedeclareFs(t *testing.T) {
	t.Parallel()
	prog, err := buildJavaScriptProgram(`
const fs = require("fs");

function main() {
  return { exists: fs.existsSync(".") };
}
`, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	s := string(prog)
	if strings.Count(s, "const fs") != 1 {
		t.Fatalf("expected exactly one top-level `const fs` declaration (the user's), got %d:\n%s",
			strings.Count(s, "const fs"), s)
	}
	if strings.Contains(s, "const fs = require('fs')") {
		t.Fatalf("wrapper must not declare its own top-level `const fs`, it collides with user scripts:\n%s", s)
	}
}

func TestBuildJavaScriptProgramRejectsInvalidChain(t *testing.T) {
	t.Parallel()
	_, err := buildJavaScriptProgram(`function main() { return 1; }`, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildJavaScriptProgramRejectsEmptyScript(t *testing.T) {
	t.Parallel()
	_, err := buildJavaScriptProgram("  ", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
