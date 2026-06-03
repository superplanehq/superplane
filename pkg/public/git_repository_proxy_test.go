package public

import "testing"

func Test__parseGitRepositoryPath(t *testing.T) {
	t.Parallel()

	canvasID := "550e8400-e29b-41d4-a716-446655440000"

	id, suffix, ok := parseGitRepositoryPath("/git/" + canvasID + ".git/info/refs")
	if !ok || id != canvasID || suffix != "info/refs" {
		t.Fatalf("got id=%q suffix=%q ok=%v", id, suffix, ok)
	}

	id, suffix, ok = parseGitRepositoryPath("/git/" + canvasID + ".git")
	if !ok || id != canvasID || suffix != "" {
		t.Fatalf("got id=%q suffix=%q ok=%v", id, suffix, ok)
	}

	_, _, ok = parseGitRepositoryPath("/git/my-app.git")
	if ok {
		t.Fatal("expected slug path to be rejected when not a uuid segment")
	}

	_, _, ok = parseGitRepositoryPath("/git/")
	if ok {
		t.Fatal("expected invalid path")
	}
}
