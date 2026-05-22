package canvasstorage

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/config"
)

func TestLocalGitProviderCommitListAndRead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	provider, err := NewLocalGitProvider(config.CanvasStorageConfig{
		LocalRoot:      t.TempDir(),
		DefaultBranch:  "main",
		MaxFileBytes:   1024,
		MaxCommitBytes: 4096,
	})
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	ctx := context.Background()
	repo, err := provider.EnsureRepository(ctx, RepositorySpec{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}

	result, err := provider.CommitFiles(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, CommitFilesOptions{
		Message: "Add files",
		Author:  CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"},
		Operations: []FileOperation{
			{Path: "docs/readme.md", Content: strings.NewReader("hello"), SizeBytes: 5},
			{Path: "notes.txt", Content: strings.NewReader("note"), SizeBytes: 4},
		},
	})
	if err != nil {
		t.Fatalf("commit files: %v", err)
	}
	if result.NewSHA == "" || result.NewSHA == result.OldSHA {
		t.Fatalf("unexpected commit result: %+v", result)
	}

	files, err := provider.ListFiles(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, ListFilesOptions{})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files.Paths) != 2 || files.Paths[0] != "docs/readme.md" || files.Paths[1] != "notes.txt" {
		t.Fatalf("unexpected files: %#v", files.Paths)
	}

	reader, err := provider.GetFile(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, GetFileOptions{
		Path: "docs/readme.md",
	})
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(body) != "hello" {
		t.Fatalf("unexpected body %q", string(body))
	}
}

func TestLocalGitProviderExpectedHeadMismatch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	provider, err := NewLocalGitProvider(config.CanvasStorageConfig{
		LocalRoot:      t.TempDir(),
		DefaultBranch:  "main",
		MaxFileBytes:   1024,
		MaxCommitBytes: 4096,
	})
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	ctx := context.Background()
	repo, err := provider.EnsureRepository(ctx, RepositorySpec{
		OrganizationID: uuid.New(),
		CanvasID:       uuid.New(),
	})
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}

	_, err = provider.CommitFiles(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, CommitFilesOptions{
		ExpectedHeadSHA: "not-the-current-head",
		Message:         "Add file",
		Author:          CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"},
		Operations: []FileOperation{
			{Path: "readme.md", Content: strings.NewReader("hello"), SizeBytes: 5},
		},
	})
	if !errors.Is(err, ErrExpectedHeadMismatch) {
		t.Fatalf("expected head mismatch, got %v", err)
	}
}

func TestLocalGitProviderRemoteURLUnsupported(t *testing.T) {
	provider, err := NewLocalGitProvider(config.CanvasStorageConfig{
		LocalRoot:     t.TempDir(),
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	_, err = provider.RemoteURL(context.Background(), RepositoryRef{}, RemoteURLOptions{})
	if !errors.Is(err, ErrRemoteURLUnsupported) {
		t.Fatalf("expected remote URL unsupported, got %v", err)
	}
}
