package canvasstorage

import (
	"context"
	"errors"
	"io"
	"os"
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
	initialHead, err := provider.CurrentHead(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, repo.DefaultBranch)
	if err != nil {
		t.Fatalf("current head: %v", err)
	}
	if initialHead == "" {
		t.Fatal("expected repository to be initialized")
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

	head, err := provider.CurrentHead(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, repo.DefaultBranch)
	if err != nil {
		t.Fatalf("current head: %v", err)
	}
	if head != result.NewSHA {
		t.Fatalf("expected current head %q, got %q", result.NewSHA, head)
	}

	files, err := provider.ListFiles(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, ListFilesOptions{})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(files.Paths) != 3 ||
		files.Paths[0] != "README.md" ||
		files.Paths[1] != "docs/readme.md" ||
		files.Paths[2] != "notes.txt" {
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

func TestLocalGitProviderDoesNotReinitializeExistingRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	cfg := config.CanvasStorageConfig{
		LocalRoot:      t.TempDir(),
		DefaultBranch:  "main",
		MaxFileBytes:   1024,
		MaxCommitBytes: 4096,
	}

	provider, err := NewLocalGitProvider(cfg)
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	ctx := context.Background()
	spec := RepositorySpec{OrganizationID: uuid.New(), CanvasID: uuid.New()}
	repo, err := provider.EnsureRepository(ctx, spec)
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}

	if _, err := provider.CommitFiles(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, CommitFilesOptions{
		Message: "Update readme",
		Author:  CommitAuthor{Name: "SuperPlane", Email: "bot@superplane.local"},
		Operations: []FileOperation{
			{Path: "README.md", Content: strings.NewReader("custom"), SizeBytes: 6},
		},
	}); err != nil {
		t.Fatalf("commit files: %v", err)
	}

	repo, err = provider.EnsureRepository(ctx, RepositorySpec{RepoID: repo.RepoID})
	if err != nil {
		t.Fatalf("ensure existing repository: %v", err)
	}

	reader, err := provider.GetFile(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch}, GetFileOptions{
		Path: "README.md",
	})
	if err != nil {
		t.Fatalf("get file: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(body) != "custom" {
		t.Fatalf("unexpected readme body %q", string(body))
	}
}

func TestLocalGitProviderDeleteRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}

	provider, err := NewLocalGitProvider(config.CanvasStorageConfig{
		LocalRoot:     t.TempDir(),
		DefaultBranch: "main",
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

	repoPath, err := provider.repoPath(repo.RepoID)
	if err != nil {
		t.Fatalf("repo path: %v", err)
	}
	if _, err := os.Stat(repoPath); err != nil {
		t.Fatalf("expected repository path to exist: %v", err)
	}

	err = provider.DeleteRepository(ctx, RepositoryRef{RepoID: repo.RepoID, DefaultBranch: repo.DefaultBranch})
	if err != nil {
		t.Fatalf("delete repository: %v", err)
	}
	if _, err := os.Stat(repoPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected repository path to be removed, got %v", err)
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

func TestLocalGitProviderGitAccessUnsupported(t *testing.T) {
	provider, err := NewLocalGitProvider(config.CanvasStorageConfig{
		LocalRoot:     t.TempDir(),
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("provider error: %v", err)
	}

	_, err = provider.GitURL(context.Background(), RepositoryRef{})
	if !errors.Is(err, ErrRemoteURLUnsupported) {
		t.Fatalf("expected remote URL unsupported, got %v", err)
	}

	_, err = provider.GenerateGitCredentials(context.Background(), RepositoryRef{}, GitCredentialsOptions{})
	if !errors.Is(err, ErrRemoteURLUnsupported) {
		t.Fatalf("expected remote URL unsupported, got %v", err)
	}
}
