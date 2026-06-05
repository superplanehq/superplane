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
