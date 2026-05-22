package canvasstorage

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestCanvasRepoID(t *testing.T) {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	canvasID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	got := CanvasRepoID(orgID, canvasID)
	want := "orgs/11111111-1111-1111-1111-111111111111/canvases/22222222-2222-2222-2222-222222222222"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestValidateRepositoryID(t *testing.T) {
	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	canvasID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	got, err := ValidateRepositoryID(CanvasRepoID(orgID, canvasID))
	if err != nil {
		t.Fatalf("expected valid repository id, got %v", err)
	}

	want := "orgs/11111111-1111-1111-1111-111111111111/canvases/22222222-2222-2222-2222-222222222222"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	invalid := []string{
		"",
		"/orgs/11111111-1111-1111-1111-111111111111/canvases/22222222-2222-2222-2222-222222222222",
		"../orgs/11111111-1111-1111-1111-111111111111/canvases/22222222-2222-2222-2222-222222222222",
		"orgs/not-a-uuid/canvases/22222222-2222-2222-2222-222222222222",
		"orgs/11111111-1111-1111-1111-111111111111/canvases/not-a-uuid",
		"orgs/11111111-1111-1111-1111-111111111111/projects/22222222-2222-2222-2222-222222222222",
	}
	for _, value := range invalid {
		t.Run(value, func(t *testing.T) {
			_, err := ValidateRepositoryID(value)
			if !errors.Is(err, ErrInvalidRepositoryID) {
				t.Fatalf("expected invalid repository id, got %v", err)
			}
		})
	}
}

func TestValidateUserPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		err  error
	}{
		{name: "valid", path: "docs/readme.md"},
		{name: "cleans", path: "docs/./readme.md"},
		{name: "leading slash", path: "/docs/readme.md"},
		{name: "empty", path: "", err: ErrInvalidPath},
		{name: "only slash", path: "/", err: ErrInvalidPath},
		{name: "traversal", path: "../secret", err: ErrInvalidPath},
		{name: "nested traversal", path: "docs/../../secret", err: ErrInvalidPath},
		{name: "git dir", path: ".git/config", err: ErrInvalidPath},
		{name: "reserved", path: ".superplane/canvas.yaml", err: ErrReservedPath},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateUserPath(tt.path)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}

func TestValidateCommitOperationsLimits(t *testing.T) {
	_, err := validateCommitOperations(nil, Limits{MaxFileBytes: 3, MaxCommitBytes: 10})
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected invalid commit, got %v", err)
	}

	_, err = validateCommitOperations([]FileOperation{
		{Path: "big.txt", Content: strings.NewReader("abcd"), SizeBytes: 4},
	}, Limits{MaxFileBytes: 3, MaxCommitBytes: 10})
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected file too large, got %v", err)
	}

	_, err = validateCommitOperations([]FileOperation{
		{Path: "a.txt", Content: strings.NewReader("abc"), SizeBytes: 3},
		{Path: "b.txt", Content: strings.NewReader("abc"), SizeBytes: 3},
	}, Limits{MaxFileBytes: 3, MaxCommitBytes: 5})
	if !errors.Is(err, ErrCommitTooLarge) {
		t.Fatalf("expected commit too large, got %v", err)
	}
}

func TestValidateCommitMetadata(t *testing.T) {
	err := validateCommitMetadata("", CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"})
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected invalid commit, got %v", err)
	}

	err = validateCommitMetadata("Update files", CommitAuthor{Name: "SuperPlane"})
	if !errors.Is(err, ErrInvalidCommit) {
		t.Fatalf("expected invalid commit, got %v", err)
	}

	err = validateCommitMetadata("Update files", CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"})
	if err != nil {
		t.Fatalf("expected valid commit metadata, got %v", err)
	}
}
