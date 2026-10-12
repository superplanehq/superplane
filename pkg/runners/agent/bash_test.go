package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBashTaskFilesPreservesUserScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	userScript := "#!/usr/bin/env bash\nset -e\nnum=$(jq -r '.pr' \"$SUPERPLANE_PAYLOAD_FILE\")\nprintf '{\"pr\":%s}\\n' \"$num\" > \"$SUPERPLANE_RESULT_FILE\"\n"
	chain := json.RawMessage(`{"pr":42}`)
	files, err := writeBashTaskFiles(dir, userScript, chain)
	if err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile(files.scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(script) != userScript {
		t.Fatalf("script modified:\n%s", script)
	}
	payload, err := os.ReadFile(files.payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"pr":42}` {
		t.Fatalf("payload = %s", payload)
	}
	if filepath.Base(files.payloadPath) != bashPayloadName {
		t.Fatalf("payload path = %s", files.payloadPath)
	}
}

func TestWriteBashTaskFilesRejectsInvalidChain(t *testing.T) {
	t.Parallel()
	_, err := writeBashTaskFiles(t.TempDir(), `echo ok`, json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteBashTaskFilesRejectsEmptyScript(t *testing.T) {
	t.Parallel()
	_, err := writeBashTaskFiles(t.TempDir(), "  ", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteBashTaskFilesDefaultEmptyChain(t *testing.T) {
	t.Parallel()
	files, err := writeBashTaskFiles(t.TempDir(), `echo ok`, nil)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(files.payloadPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(payload)) != "{}" {
		t.Fatalf("payload = %s", payload)
	}
}
