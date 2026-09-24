package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/superplane/runner/shared/api"
)

func TestMaterializeTaskFiles(t *testing.T) {
	root := t.TempDir()
	got, err := materializeTaskFiles(root, []api.TaskFile{
		{Path: "format.js", Content: "console.log(1)\n"},
		{Path: "steps/01.sh", Content: "#!/bin/bash\necho hi\n", Mode: "0755"},
	})
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if got != root {
		// materialize returns abs path
		abs, _ := filepath.Abs(root)
		if got != abs {
			t.Fatalf("root = %q, want %q", got, abs)
		}
	}

	body, err := os.ReadFile(filepath.Join(got, "format.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "console.log(1)\n" {
		t.Fatalf("format.js = %q", body)
	}

	st, err := os.Stat(filepath.Join(got, "steps", "01.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&0o111 == 0 {
		t.Fatalf("expected executable mode, got %v", st.Mode())
	}
}

func TestMaterializeTaskFilesEmpty(t *testing.T) {
	got, err := materializeTaskFiles(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestWithTaskDirEnv(t *testing.T) {
	env := withTaskDirEnv([]string{"FOO=bar"}, "/tmp/task")
	found := false
	for _, pair := range env {
		if pair == envSuperplaneTaskDir+"=/tmp/task" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing task dir in %#v", env)
	}
}
