package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

const (
	ReservedSuperPlanePath = ".superplane"
)

var (
	ErrInvalidPath          = errors.New("invalid file path")
	ErrInvalidRepositoryID  = errors.New("invalid repository path")
	ErrReservedPath         = errors.New("path is reserved for SuperPlane")
	ErrFileTooLarge         = errors.New("file exceeds configured size limit")
	ErrCommitTooLarge       = errors.New("commit exceeds configured size limit")
	ErrInvalidCommit        = errors.New("invalid commit")
	ErrExpectedHeadMismatch = errors.New("expected head sha does not match current branch head")
	ErrRepositoryNotFound   = errors.New("repository not found")
)

type Limits struct {
	MaxFileBytes   int64
	MaxCommitBytes int64
}

type RepositorySpec struct {
	ID            string
	DefaultBranch string
}

type Repository struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"default_branch"`
}

type RepositoryRef struct {
	ID            string
	DefaultBranch string
}

type ListFilesOptions struct {
	Ref string
}

type ListFilesResult struct {
	Paths []string `json:"paths"`
	Ref   string   `json:"ref"`
}

type GetFileOptions struct {
	Path string
	Ref  string
}

type FileOperation struct {
	Path      string
	Content   io.Reader
	SizeBytes int64
	Delete    bool
}

type CommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type CommitOptions struct {
	Branch          string
	BaseBranch      string
	ExpectedHeadSHA string
	Message         string
	Author          CommitAuthor
	Operations      []FileOperation
}

type CommitResult struct {
	CommitSHA string
	OldSHA    string
}

type CommitInfo struct {
	CommitSHA string       `json:"commit_sha"`
	TreeSHA   string       `json:"tree_sha"`
	Message   string       `json:"message"`
	Author    CommitAuthor `json:"author"`
}

type Store struct {
	root          string
	defaultBranch string
	limits        Limits
	locks         sync.Map
}

func NewStore(root, initialBranch string, limits Limits) (*Store, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("storage root is required")
	}

	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("supergit requires git in PATH: %w", err)
	}

	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	return &Store{
		root:          root,
		defaultBranch: defaultBranch(initialBranch),
		limits:        limits,
	}, nil
}

func (s *Store) CreateRepository(ctx context.Context, spec RepositorySpec) (*Repository, error) {
	repoID, err := ValidateRepositoryID(spec.ID)
	if err != nil {
		return nil, err
	}

	branch := defaultBranch(spec.DefaultBranch)
	if branch == "main" {
		branch = s.defaultBranch
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(repoPath); err == nil {
		return &Repository{ID: repoID, DefaultBranch: branch}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if _, err := runGit(ctx, "", "init", "--bare", "--initial-branch", branch, repoPath); err != nil {
		return nil, err
	}

	return &Repository{ID: repoID, DefaultBranch: branch}, nil
}

func (s *Store) ListRepositories(ctx context.Context) ([]Repository, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Repository{}, nil
		}
		return nil, err
	}

	repos := make([]Repository, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".git") {
			continue
		}

		repoID := strings.TrimSuffix(entry.Name(), ".git")
		repoID = strings.ReplaceAll(repoID, string(os.PathSeparator), "/")
		if _, err := ValidateRepositoryID(repoID); err != nil {
			continue
		}

		branch := s.defaultBranch
		if headBranch, err := s.defaultBranchForRepo(ctx, repoID); err == nil && headBranch != "" {
			branch = headBranch
		}

		repos = append(repos, Repository{
			ID:            repoID,
			DefaultBranch: branch,
		})
	}

	return repos, nil
}

func (s *Store) GetRepository(ctx context.Context, repoID string) (*Repository, error) {
	repoID, err := ValidateRepositoryID(repoID)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(repoPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}

	branch := s.defaultBranch
	if headBranch, err := s.defaultBranchForRepo(ctx, repoID); err == nil && headBranch != "" {
		branch = headBranch
	}

	return &Repository{ID: repoID, DefaultBranch: branch}, nil
}

func (s *Store) DeleteRepository(ctx context.Context, ref RepositoryRef) error {
	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return err
	}

	unlock := s.lock(repoID)
	defer unlock()

	if _, err := os.Stat(repoPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return os.RemoveAll(repoPath)
}

func (s *Store) ListFiles(ctx context.Context, ref RepositoryRef, options ListFilesOptions) (*ListFilesResult, error) {
	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(repoID)
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

func (s *Store) GetFile(ctx context.Context, ref RepositoryRef, options GetFileOptions) (io.ReadCloser, error) {
	filePath, err := NormalizePath(options.Path)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(ref.ID)
	if err != nil {
		return nil, err
	}

	gitRef := refOrDefault(options.Ref, ref.DefaultBranch)
	cmd := exec.CommandContext(ctx, "git", "--git-dir", repoPath, "show", gitRef+":"+filePath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &gitCommandReadCloser{
		reader: stdout,
		cmd:    cmd,
		stderr: &stderr,
	}, nil
}

func (s *Store) Commit(ctx context.Context, ref RepositoryRef, options CommitOptions) (*CommitResult, error) {
	if err := validateCommitMetadata(options.Message, options.Author); err != nil {
		return nil, err
	}

	operations, err := validateCommitOperations(options.Operations, s.limits)
	if err != nil {
		return nil, err
	}

	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	unlock := s.lock(repoID)
	defer unlock()

	branch := refOrDefault(options.Branch, ref.DefaultBranch)
	oldSHA, _ := s.Head(ctx, ref, branch)
	if expected := strings.TrimSpace(options.ExpectedHeadSHA); expected != "" && expected != oldSHA {
		return nil, ErrExpectedHeadMismatch
	}

	worktree, err := os.MkdirTemp("", "supergit-repo-*")
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

		if err := writeLimitedFile(destination, operation.Content, s.limits.MaxFileBytes); err != nil {
			return nil, err
		}
	}

	if _, err := runGit(ctx, worktree, "add", "-A"); err != nil {
		return nil, err
	}

	if _, err := runGit(ctx, worktree, "diff", "--cached", "--quiet"); err == nil {
		return &CommitResult{CommitSHA: oldSHA, OldSHA: oldSHA}, nil
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

	newSHA, err := s.Head(ctx, ref, branch)
	if err != nil {
		return nil, err
	}

	return &CommitResult{CommitSHA: newSHA, OldSHA: oldSHA}, nil
}

func (s *Store) Head(ctx context.Context, ref RepositoryRef, branch string) (string, error) {
	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return "", err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return "", err
	}

	branch = refOrDefault(branch, ref.DefaultBranch)
	out, err := runGit(ctx, "", "--git-dir", repoPath, "rev-parse", "--verify", "refs/heads/"+branch)
	if err != nil {
		if isUnknownRevision(err) {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (s *Store) GetCommit(ctx context.Context, ref RepositoryRef, sha string) (*CommitInfo, error) {
	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	sha = strings.TrimSpace(sha)
	if sha == "" {
		sha = refOrDefault("", ref.DefaultBranch)
	}

	commitSHA, err := runGit(ctx, "", "--git-dir", repoPath, "rev-parse", "--verify", sha+"^{commit}")
	if err != nil {
		if isUnknownRevision(err) {
			return nil, ErrRepositoryNotFound
		}
		return nil, err
	}

	treeSHA, err := runGit(ctx, "", "--git-dir", repoPath, "show", "-s", "--format=%T", strings.TrimSpace(string(commitSHA)))
	if err != nil {
		return nil, err
	}

	message, err := runGit(ctx, "", "--git-dir", repoPath, "show", "-s", "--format=%B", strings.TrimSpace(string(commitSHA)))
	if err != nil {
		return nil, err
	}

	authorName, err := runGit(ctx, "", "--git-dir", repoPath, "show", "-s", "--format=%an", strings.TrimSpace(string(commitSHA)))
	if err != nil {
		return nil, err
	}

	authorEmail, err := runGit(ctx, "", "--git-dir", repoPath, "show", "-s", "--format=%ae", strings.TrimSpace(string(commitSHA)))
	if err != nil {
		return nil, err
	}

	return &CommitInfo{
		CommitSHA: strings.TrimSpace(string(commitSHA)),
		TreeSHA:   strings.TrimSpace(string(treeSHA)),
		Message:   strings.TrimSpace(string(message)),
		Author: CommitAuthor{
			Name:  strings.TrimSpace(string(authorName)),
			Email: strings.TrimSpace(string(authorEmail)),
		},
	}, nil
}

func (s *Store) ListCommits(ctx context.Context, ref RepositoryRef, branch string, limit int) ([]CommitInfo, error) {
	repoID, err := ValidateRepositoryID(ref.ID)
	if err != nil {
		return nil, err
	}

	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}

	gitRef := refOrDefault(branch, ref.DefaultBranch)
	out, err := runGit(ctx, "", "--git-dir", repoPath, "log", gitRef, fmt.Sprintf("-%d", limit), "--format=%H")
	if err != nil {
		if isUnknownRevision(err) {
			return []CommitInfo{}, nil
		}
		return nil, err
	}

	commits := make([]CommitInfo, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		info, err := s.GetCommit(ctx, ref, line)
		if err != nil {
			return nil, err
		}
		commits = append(commits, *info)
	}

	return commits, nil
}

func (s *Store) defaultBranchForRepo(ctx context.Context, repoID string) (string, error) {
	repoPath, err := s.repoPath(repoID)
	if err != nil {
		return "", err
	}

	out, err := runGit(ctx, "", "--git-dir", repoPath, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func (s *Store) repoPath(repoID string) (string, error) {
	repoID, err := ValidateRepositoryID(repoID)
	if err != nil {
		return "", err
	}

	repoPath := filepath.Join(s.root, filepath.FromSlash(repoID)+".git")
	cleanRoot, err := filepath.Abs(s.root)
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

func (s *Store) lock(repoID string) func() {
	value, _ := s.locks.LoadOrStore(repoID, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

type validatedOperation struct {
	Path      string
	Content   io.Reader
	SizeBytes int64
	Delete    bool
}

func ValidateRepositoryID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ReplaceAll(value, "\\", "/"), "/") {
		return "", ErrInvalidRepositoryID
	}

	normalized, err := NormalizePath(value)
	if err != nil {
		return "", ErrInvalidRepositoryID
	}

	segments := strings.Split(normalized, "/")
	if len(segments) != 4 || segments[0] != "orgs" || segments[2] != "canvases" {
		return "", ErrInvalidRepositoryID
	}
	if _, err := uuid.Parse(segments[1]); err != nil {
		return "", ErrInvalidRepositoryID
	}
	if _, err := uuid.Parse(segments[3]); err != nil {
		return "", ErrInvalidRepositoryID
	}

	return normalized, nil
}

func ValidateUserPath(value string) (string, error) {
	normalized, err := NormalizePath(value)
	if err != nil {
		return "", err
	}

	if normalized == ReservedSuperPlanePath || strings.HasPrefix(normalized, ReservedSuperPlanePath+"/") {
		return "", ErrReservedPath
	}

	return normalized, nil
}

func NormalizePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsRune(value, '\x00') {
		return "", ErrInvalidPath
	}

	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimLeft(value, "/")
	if value == "" {
		return "", ErrInvalidPath
	}

	normalized := path.Clean(value)
	if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
		return "", ErrInvalidPath
	}

	for _, segment := range strings.Split(normalized, "/") {
		if segment == "" || segment == "." || segment == ".." || segment == ".git" {
			return "", ErrInvalidPath
		}
	}

	return normalized, nil
}

func validateCommitOperations(operations []FileOperation, limits Limits) ([]validatedOperation, error) {
	if len(operations) == 0 {
		return nil, fmt.Errorf("%w: at least one file operation is required", ErrInvalidCommit)
	}

	validated := make([]validatedOperation, 0, len(operations))
	var totalBytes int64

	for _, operation := range operations {
		path, err := ValidateUserPath(operation.Path)
		if err != nil {
			return nil, err
		}

		if !operation.Delete {
			if operation.Content == nil {
				return nil, fmt.Errorf("%w: content is required for %q", ErrInvalidPath, path)
			}
			if operation.SizeBytes < 0 {
				return nil, fmt.Errorf("%w: size is required for %q", ErrInvalidPath, path)
			}
			if limits.MaxFileBytes > 0 && operation.SizeBytes > limits.MaxFileBytes {
				return nil, fmt.Errorf("%w: %q", ErrFileTooLarge, path)
			}
			totalBytes += operation.SizeBytes
		}

		validated = append(validated, validatedOperation{
			Path:      path,
			Content:   operation.Content,
			SizeBytes: operation.SizeBytes,
			Delete:    operation.Delete,
		})
	}

	if limits.MaxCommitBytes > 0 && totalBytes > limits.MaxCommitBytes {
		return nil, ErrCommitTooLarge
	}

	return validated, nil
}

func validateCommitMetadata(message string, author CommitAuthor) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("%w: commit message is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Name) == "" {
		return fmt.Errorf("%w: author name is required", ErrInvalidCommit)
	}
	if strings.TrimSpace(author.Email) == "" {
		return fmt.Errorf("%w: author email is required", ErrInvalidCommit)
	}
	return nil
}

func defaultBranch(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return "main"
	}
	return branch
}

func refOrDefault(ref, branch string) string {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return ref
	}
	return defaultBranch(branch)
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

type gitCommandReadCloser struct {
	reader io.ReadCloser
	cmd    *exec.Cmd
	stderr *bytes.Buffer
	once   sync.Once
	err    error
}

func (r *gitCommandReadCloser) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if errors.Is(err, io.EOF) {
		if waitErr := r.wait(); waitErr != nil {
			return n, waitErr
		}
	}

	return n, err
}

func (r *gitCommandReadCloser) Close() error {
	err := r.reader.Close()
	if r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
	if waitErr := r.wait(); err == nil {
		err = waitErr
	}

	return err
}

func (r *gitCommandReadCloser) wait() error {
	r.once.Do(func() {
		if err := r.cmd.Wait(); err != nil {
			r.err = fmt.Errorf("git %s failed: %w: %s", strings.Join(r.cmd.Args[1:], " "), err, strings.TrimSpace(r.stderr.String()))
		}
	})

	return r.err
}
