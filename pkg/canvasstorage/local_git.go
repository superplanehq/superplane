package canvasstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/superplanehq/superplane/pkg/config"
)

type LocalGitProvider struct {
	root          string
	defaultBranch string
	limits        Limits
	locks         sync.Map
}

func NewLocalGitProvider(cfg config.CanvasStorageConfig) (*LocalGitProvider, error) {
	root := strings.TrimSpace(cfg.LocalRoot)
	if root == "" {
		return nil, errors.New("local canvas storage root is required")
	}

	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("local canvas storage requires git in PATH: %w", err)
	}

	return &LocalGitProvider{
		root:          root,
		defaultBranch: defaultBranch(cfg.DefaultBranch),
		limits: Limits{
			MaxFileBytes:   cfg.MaxFileBytes,
			MaxCommitBytes: cfg.MaxCommitBytes,
		},
	}, nil
}

func (p *LocalGitProvider) EnsureRepository(ctx context.Context, spec RepositorySpec) (*Repository, error) {
	repoID := strings.TrimSpace(spec.RepoID)
	if repoID == "" {
		repoID = CanvasRepoID(spec.OrganizationID, spec.CanvasID)
	}

	branch := defaultBranch(spec.DefaultBranch)
	if branch == "main" {
		branch = p.defaultBranch
	}

	repoPath, err := p.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(repoPath); errors.Is(err, os.ErrNotExist) {
		if _, err := runGit(ctx, "", "init", "--bare", "--initial-branch", branch, repoPath); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	head, _ := p.currentHead(ctx, repoID, branch)
	return &Repository{RepoID: repoID, DefaultBranch: branch, HeadSHA: head}, nil
}

func (p *LocalGitProvider) ListFiles(ctx context.Context, ref RepositoryRef, options ListFilesOptions) (*ListFilesResult, error) {
	repoID := strings.TrimSpace(ref.RepoID)
	repoPath, err := p.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	gitRef := refOrDefault(options.Ref, ref.DefaultBranch)
	out, err := runGit(ctx, "", "--git-dir", repoPath, "ls-tree", "-r", "--name-only", gitRef)
	if err != nil {
		if isUnknownRevision(err) {
			return &ListFilesResult{Paths: []string{}, Ref: gitRef}, nil
		}
		return nil, err
	}

	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			paths = append(paths, line)
		}
	}

	return &ListFilesResult{Paths: paths, Ref: gitRef}, nil
}

func (p *LocalGitProvider) GetFile(ctx context.Context, ref RepositoryRef, options GetFileOptions) (io.ReadCloser, error) {
	filePath, err := NormalizePath(options.Path)
	if err != nil {
		return nil, err
	}

	repoPath, err := p.repoPath(ref.RepoID)
	if err != nil {
		return nil, err
	}

	gitRef := refOrDefault(options.Ref, ref.DefaultBranch)
	out, err := runGit(ctx, "", "--git-dir", repoPath, "show", gitRef+":"+filePath)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(out)), nil
}

func (p *LocalGitProvider) CommitFiles(ctx context.Context, ref RepositoryRef, options CommitFilesOptions) (*CommitResult, error) {
	if err := validateCommitMetadata(options.Message, options.Author); err != nil {
		return nil, err
	}

	operations, err := validateCommitOperations(options.Operations, p.limits)
	if err != nil {
		return nil, err
	}

	repoID := strings.TrimSpace(ref.RepoID)
	repoPath, err := p.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	unlock := p.lock(repoID)
	defer unlock()

	branch := refOrDefault(options.Branch, ref.DefaultBranch)
	oldSHA, _ := p.currentHead(ctx, repoID, branch)
	if expected := strings.TrimSpace(options.ExpectedHeadSHA); expected != "" && expected != oldSHA {
		return nil, ErrExpectedHeadMismatch
	}

	worktree, err := os.MkdirTemp("", "superplane-canvas-repo-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(worktree)

	if _, err := runGit(ctx, "", "clone", repoPath, worktree); err != nil {
		return nil, err
	}

	if _, err := runGit(ctx, worktree, "checkout", branch); err != nil {
		if base := strings.TrimSpace(options.BaseBranch); base != "" {
			if _, err := runGit(ctx, worktree, "checkout", "-b", branch, "origin/"+base); err != nil {
				return nil, err
			}
		} else if _, err := runGit(ctx, worktree, "checkout", "-B", branch); err != nil {
			return nil, err
		}
	}

	for _, operation := range operations {
		destination := filepath.Join(worktree, filepath.FromSlash(operation.Path))
		if !strings.HasPrefix(destination, worktree+string(os.PathSeparator)) {
			return nil, ErrInvalidPath
		}

		if operation.Delete {
			if err := os.RemoveAll(destination); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			return nil, err
		}

		if err := writeLimitedFile(destination, operation.Content, p.limits.MaxFileBytes); err != nil {
			return nil, err
		}
	}

	if _, err := runGit(ctx, worktree, "add", "-A"); err != nil {
		return nil, err
	}

	if _, err := runGit(ctx, worktree, "diff", "--cached", "--quiet"); err == nil {
		return &CommitResult{OldSHA: oldSHA, NewSHA: oldSHA, Branch: branch}, nil
	}

	if _, err := runGit(ctx, worktree,
		"-c", "user.name="+strings.TrimSpace(options.Author.Name),
		"-c", "user.email="+strings.TrimSpace(options.Author.Email),
		"-c", "commit.gpgsign=false",
		"commit", "-m", strings.TrimSpace(options.Message),
	); err != nil {
		return nil, err
	}

	if _, err := runGit(ctx, worktree, "push", "origin", "HEAD:refs/heads/"+branch); err != nil {
		return nil, err
	}

	newSHA, err := p.currentHead(ctx, repoID, branch)
	if err != nil {
		return nil, err
	}

	return &CommitResult{CommitSHA: newSHA, OldSHA: oldSHA, NewSHA: newSHA, Branch: branch}, nil
}

func (p *LocalGitProvider) RemoteURL(ctx context.Context, ref RepositoryRef, options RemoteURLOptions) (string, error) {
	return "", ErrRemoteURLUnsupported
}

func (p *LocalGitProvider) repoPath(repoID string) (string, error) {
	repoID, err := ValidateRepositoryID(repoID)
	if err != nil {
		return "", err
	}

	repoPath := filepath.Join(p.root, filepath.FromSlash(repoID)+".git")
	cleanRoot, err := filepath.Abs(p.root)
	if err != nil {
		return "", err
	}
	cleanRepo, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	if cleanRepo != cleanRoot && !strings.HasPrefix(cleanRepo, cleanRoot+string(os.PathSeparator)) {
		return "", ErrInvalidRepositoryID
	}

	return repoPath, nil
}

func (p *LocalGitProvider) currentHead(ctx context.Context, repoID, branch string) (string, error) {
	repoPath, err := p.repoPath(repoID)
	if err != nil {
		return "", err
	}

	out, err := runGit(ctx, "", "--git-dir", repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
	if err != nil {
		if isUnknownRevision(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (p *LocalGitProvider) lock(repoID string) func() {
	value, _ := p.locks.LoadOrStore(repoID, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func writeLimitedFile(destination string, content io.Reader, limit int64) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := content
	if limit > 0 {
		reader = io.LimitReader(content, limit+1)
	}

	written, err := io.Copy(file, reader)
	if err != nil {
		return err
	}
	if limit > 0 && written > limit {
		return ErrFileTooLarge
	}

	return nil
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func isUnknownRevision(err error) bool {
	message := err.Error()
	return strings.Contains(message, "unknown revision") ||
		strings.Contains(message, "Needed a single revision") ||
		strings.Contains(message, "Not a valid object name") ||
		strings.Contains(message, "does not exist")
}
