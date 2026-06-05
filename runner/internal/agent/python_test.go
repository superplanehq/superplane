package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPythonProgram(t *testing.T) {
	t.Parallel()
	chain := json.RawMessage(`{"GitHub PR":{"data":{"number":42}}}`)
	prog, err := buildPythonProgram(`
def main(payload):
    return {"pr": payload["GitHub PR"]["data"]["number"]}
`, chain)
	if err != nil {
		t.Fatal(err)
	}
	s := string(prog)
	for _, want := range []string{
		`payload = json.loads(`,
		`def main(payload):`,
		`main_fn(payload)`,
		"SUPERPLANE_RESULT_FILE",
		"main(payload) is required",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("program missing %q:\n%s", want, s)
		}
	}
}

func TestBuildPythonProgramRejectsInvalidChain(t *testing.T) {
	t.Parallel()
	_, err := buildPythonProgram(`def main(payload): return 1`, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildPythonProgramRejectsEmptyScript(t *testing.T) {
	t.Parallel()
	_, err := buildPythonProgram("  ", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}
