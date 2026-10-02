package fs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	api "github.com/superplanehq/superplane/pkg/runners/logs"
)

const (
	dataFilename     = "logs.ndjson"
	manifestFilename = "manifest.json"
	lockDirectory    = ".locks"
	lockShardCount   = 4096
)

/**
 * Store keeps each active runner task log in one directory on a local or
 * shared POSIX-compatible filesystem. The task directory contains an
 * append-only logs.ndjson file and a manifest.json file. The manifest stores
 * the next sequence, committed byte count, truncation state, and last update
 * time. The manifest is the commit record: an append synchronizes log data
 * first, then replaces and synchronizes the manifest. If a process stops
 * between these steps, the next operation truncates data beyond the manifest's
 * committed byte count.
 *
 * Ordering and deduplication use the manifest's next sequence, not its byte
 * count. Append compares the incoming runner chunk sequence with NextSequence:
 *
 *     incoming sequence < NextSequence: duplicate; return success without write
 *     incoming sequence = NextSequence: append and increment NextSequence
 *     incoming sequence > NextSequence: missing earlier chunk; return conflict
 *
 * For example:
 *
 *     before: file="abc", NextSequence=1, TotalBytes=3
 *     append: sequence=1, content="hello"
 *     data:   file="abchello", manifest still says sequence=1 and bytes=3
 *     commit: NextSequence=2, TotalBytes=8
 *
 * If the process stops before the manifest commit, a retry sees sequence 1 and
 * TotalBytes 3. It truncates the uncommitted "hello" bytes and appends the
 * chunk again. If the manifest commit succeeded but the response failed, a
 * retry sees sequence 1 below NextSequence 2 and returns success without
 * another write. The task locks prevent concurrent replicas from accepting the
 * same next sequence. Duplicate detection uses the sequence only and assumes
 * the runner's durable spool sends the same content for a retried sequence.
 *
 * Initialize creates a task in the primary path before SuperPlane sends the
 * task to a runner. Existing operations locate the task manifest across the
 * primary path and all fallback paths. This lets a deployment direct new tasks
 * to a new volume while running tasks continue to use an old volume. A task
 * manifest in more than one configured path is an error.
 *
 * Each task operation uses both an in-process mutex and a filesystem advisory
 * lock. The mutex coordinates goroutines in one application process. The
 * filesystem lock also coordinates separate processes or pods when they mount
 * the same shared filesystem. Local development uses both locks on its local
 * Docker volume, so it exercises the same operation path without requiring
 * shared storage.
 *
 * Setup creates a .locks directory under each configured root. The filesystem
 * locks use 4,096 fixed shards in that directory. A shard file is created
 * lazily when an operation first selects it and remains for later operations.
 * The task UUID selects one shard. One lock file per task would accumulate
 * indefinitely. Deleting a per-task lock file is unsafe because a process can
 * still hold the deleted inode while another process creates and locks a new
 * file at the same path. The two processes would then hold different locks for
 * the same task. Fixed shards avoid deletion and keep the number of lock files
 * bounded. Tasks that select the same shard serialize briefly, but their data
 * remains separate. The shard count balances this contention against the
 * number of permanent lock files.
 *
 * The shard selection is:
 *
 *     task UUID
 *         |
 *         v
 *     first two UUID bytes as a 16-bit number
 *         |
 *         v
 *     value modulo 4,096
 *         |
 *         v
 *     shard 0-4,095
 *         |
 *         v
 *     .locks/xxxx.lock
 *
 * For example, a UUID that starts with f2f7 uses 0xf2f7, or 62,199.
 * 62,199 modulo 4,096 is 759, which is 0x02f7. The lock path is therefore
 * .locks/02f7.lock. Prefixes 02f7, 12f7, through f2f7 select the same shard.
 * Because random UUIDs have random first bytes, tasks distribute evenly across
 * the shards.
 *
 * Runner chunk sequences and read cursors have different purposes. Append uses
 * chunk sequences to order uploads and reject missing or duplicate chunks.
 * ReadAfter uses committed byte offsets as opaque cursors and does not preserve
 * runner chunk boundaries.
 *
 * ReadAfter opens the data file while holding the task locks. It reads the
 * committed end from the manifest and creates an io.SectionReader bounded by
 * the requested offset and that committed end:
 *
 *     committed file: [0--------------13)
 *     requested cursor:       6
 *     returned section:       [6------13)
 *     returned cursor:                 13
 *
 * The locks can be released after the bounded reader is open. A later append
 * can extend the file to byte 20, but the existing reader still ends at byte
 * 13. The next request returns cursor 13 and receives bytes [13,20). This works
 * because the store never modifies committed bytes in place. Callers must
 * treat the cursor as opaque and return it unchanged; they must not convert an
 * upload chunk sequence into a cursor.
 *
 * Delete removes one task directory.
 */
type Store struct {
	primaryPath string
	paths       []string
	setupMu     sync.Mutex
	locks       [lockShardCount]sync.Mutex
	metrics     storeMetrics
}

type manifest struct {
	NextSequence int64     `json:"next_sequence"`
	TotalBytes   int64     `json:"total_bytes"`
	Truncated    bool      `json:"truncated"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type fileSectionReadCloser struct {
	io.Reader
	dataFile *os.File
}

func (r *fileSectionReadCloser) Close() error {
	return r.dataFile.Close()
}

func NewProvider() (*Store, error) {
	primaryPath := strings.TrimSpace(os.Getenv("RUNNER_ACTIVE_LOG_FS_PATH"))
	if primaryPath == "" {
		return nil, errors.New("RUNNER_ACTIVE_LOG_FS_PATH is not set")
	}
	var fallbackPaths []string
	for path := range strings.SplitSeq(
		os.Getenv("RUNNER_ACTIVE_LOG_FS_FALLBACK_PATHS"),
		",",
	) {
		if path = strings.TrimSpace(path); path != "" {
			fallbackPaths = append(fallbackPaths, path)
		}
	}
	return New(primaryPath, fallbackPaths...)
}

// New creates an active-log store that writes new tasks to primaryPath and
// searches fallbackPaths for tasks created before a storage migration.
func New(primaryPath string, fallbackPaths ...string) (*Store, error) {
	paths := make([]string, 0, len(fallbackPaths)+1)
	seen := map[string]bool{}
	for index, path := range append([]string{primaryPath}, fallbackPaths...) {
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, errors.New("runner active log FS path is required")
		}
		path = filepath.Clean(path)
		if seen[path] {
			return nil, fmt.Errorf("runner active log FS path %q is configured more than once", path)
		}
		seen[path] = true
		paths = append(paths, path)
		if index == 0 {
			primaryPath = path
		}
	}
	return &Store{
		primaryPath: primaryPath,
		paths:       paths,
	}, nil
}

func (s *Store) Name() string {
	return api.StoreFS
}

func (s *Store) Setup(ctx api.SetupContext) error {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if ctx.Context == nil {
		return errors.New("FS active log store requires a context")
	}
	if ctx.MeterProvider == nil {
		return errors.New("FS active log store requires a meter provider")
	}
	metrics, err := newStoreMetrics(ctx.MeterProvider)
	if err != nil {
		return fmt.Errorf("create FS active log store metrics: %w", err)
	}
	for _, path := range s.paths {
		if err := os.MkdirAll(filepath.Join(path, lockDirectory), 0o750); err != nil {
			return fmt.Errorf("create FS active log path %q: %w", path, err)
		}
	}
	s.metrics = metrics
	return nil
}

func (s *Store) Initialize(ctx context.Context, taskID uuid.UUID) (err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "initialize", err)
	}()

	if err := ctx.Err(); err != nil {
		return err
	}
	processLock := s.lockFor(taskID)
	processLock.Lock()
	defer processLock.Unlock()

	if _, found, err := s.locateTask(taskID); err != nil || found {
		return err
	}

	lockFile, err := s.acquireFileLock(ctx, s.primaryPath, taskID)
	if err != nil {
		return err
	}
	defer releaseFileLock(lockFile)

	if _, found, err := s.locateTask(taskID); err != nil || found {
		return err
	}

	taskDir := s.taskDir(s.primaryPath, taskID)
	if err := os.MkdirAll(taskDir, 0o750); err != nil {
		return fmt.Errorf("create active runner log directory: %w", err)
	}
	if err := appendAndSync(taskDir, 0, nil); err != nil {
		return err
	}
	return writeManifest(taskDir, manifest{
		UpdatedAt: time.Now().UTC(),
	})
}

func (s *Store) Append(
	ctx context.Context,
	taskID uuid.UUID,
	sequence int64,
	content []byte,
) (result api.AppendResult, err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "append", err)
	}()

	if sequence < 0 {
		return api.AppendResult{}, api.ErrSequenceConflict
	}
	if err := ctx.Err(); err != nil {
		return api.AppendResult{}, err
	}

	processLock := s.lockFor(taskID)
	processLock.Lock()
	defer processLock.Unlock()

	root, found, err := s.locateTask(taskID)
	if err != nil {
		return api.AppendResult{}, err
	}
	if !found {
		return api.AppendResult{}, api.ErrNotFound
	}
	lockFile, err := s.acquireFileLock(ctx, root, taskID)
	if err != nil {
		return api.AppendResult{}, err
	}
	defer releaseFileLock(lockFile)

	taskDir := s.taskDir(root, taskID)
	current, found, err := readManifest(taskDir)
	if err != nil {
		return api.AppendResult{}, err
	}
	if !found {
		return api.AppendResult{}, api.ErrNotFound
	}
	if err := repairDataFile(taskDir, current.TotalBytes); err != nil {
		return api.AppendResult{}, err
	}

	if sequence < current.NextSequence {
		return api.AppendResult{Truncated: current.Truncated}, nil
	}
	if sequence > current.NextSequence {
		return api.AppendResult{}, api.ErrSequenceConflict
	}
	if current.Truncated {
		return api.AppendResult{Truncated: true}, nil
	}

	stored, truncated := api.RetainContent(current.TotalBytes, content)
	if err := appendAndSync(taskDir, current.TotalBytes, stored); err != nil {
		return api.AppendResult{}, err
	}

	current.NextSequence++
	current.TotalBytes += int64(len(stored))
	current.Truncated = truncated
	current.UpdatedAt = time.Now().UTC()
	if err := writeManifest(taskDir, current); err != nil {
		return api.AppendResult{}, err
	}
	return api.AppendResult{Truncated: truncated}, nil
}

func (s *Store) ReadAfter(
	ctx context.Context,
	taskID uuid.UUID,
	cursor string,
) (result *api.ReadResult, err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "read", err)
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	after, err := parseCursor(cursor)
	if err != nil {
		return nil, err
	}

	processLock := s.lockFor(taskID)
	processLock.Lock()
	defer processLock.Unlock()

	root, found, err := s.locateTask(taskID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, api.ErrNotFound
	}
	lockFile, err := s.acquireFileLock(ctx, root, taskID)
	if err != nil {
		return nil, err
	}
	defer releaseFileLock(lockFile)

	taskDir := s.taskDir(root, taskID)
	current, found, err := readManifest(taskDir)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, api.ErrNotFound
	}
	if after > current.TotalBytes {
		return nil, api.ErrInvalidCursor
	}
	if err := repairDataFile(taskDir, current.TotalBytes); err != nil {
		return nil, err
	}

	file, err := os.Open(filepath.Join(taskDir, dataFilename))
	if err != nil {
		return nil, fmt.Errorf("open active runner log data: %w", err)
	}
	content := io.NewSectionReader(file, after, current.TotalBytes-after)
	return &api.ReadResult{
		Content: &fileSectionReadCloser{
			Reader:   content,
			dataFile: file,
		},
		Cursor:    formatCursor(current.TotalBytes),
		Truncated: current.Truncated,
	}, nil
}

func (s *Store) Delete(ctx context.Context, taskID uuid.UUID) (err error) {
	startedAt := time.Now()
	defer func() {
		s.metrics.recordOperation(ctx, startedAt, "delete", err)
	}()

	if err := ctx.Err(); err != nil {
		return err
	}
	processLock := s.lockFor(taskID)
	processLock.Lock()
	defer processLock.Unlock()

	root, found, err := s.locateTask(taskID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	lockFile, err := s.acquireFileLock(ctx, root, taskID)
	if err != nil {
		return err
	}
	defer releaseFileLock(lockFile)

	if err := os.RemoveAll(s.taskDir(root, taskID)); err != nil {
		return fmt.Errorf("delete active runner log directory: %w", err)
	}
	return nil
}

func (s *Store) taskDir(root string, taskID uuid.UUID) string {
	return filepath.Join(root, taskID.String())
}

func (s *Store) lockFor(taskID uuid.UUID) *sync.Mutex {
	return &s.locks[s.lockShard(taskID)]
}

func (s *Store) lockShard(taskID uuid.UUID) int {
	return (int(taskID[0])<<8 | int(taskID[1])) % lockShardCount
}

func (s *Store) acquireFileLock(
	ctx context.Context,
	root string,
	taskID uuid.UUID,
) (*os.File, error) {
	path := filepath.Join(
		root,
		lockDirectory,
		fmt.Sprintf("%04x.lock", s.lockShard(taskID)),
	)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open active runner log lock: %w", err)
	}

	retry := time.NewTicker(10 * time.Millisecond)
	defer retry.Stop()
	for {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return file, nil
		}
		if err != syscall.EWOULDBLOCK && err != syscall.EAGAIN {
			_ = file.Close()
			return nil, fmt.Errorf("lock active runner log: %w", err)
		}
		select {
		case <-ctx.Done():
			_ = file.Close()
			return nil, ctx.Err()
		case <-retry.C:
		}
	}
}

func releaseFileLock(file *os.File) error {
	return errors.Join(
		syscall.Flock(int(file.Fd()), syscall.LOCK_UN),
		file.Close(),
	)
}

func (s *Store) locateTask(taskID uuid.UUID) (string, bool, error) {
	var found string
	for _, root := range s.paths {
		_, err := os.Stat(filepath.Join(s.taskDir(root, taskID), manifestFilename))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", false, fmt.Errorf("locate active runner log: %w", err)
		}
		if found != "" {
			return "", false, fmt.Errorf(
				"active runner log %s exists in multiple FS paths",
				taskID,
			)
		}
		found = root
	}
	return found, found != "", nil
}

func readManifest(taskDir string) (manifest, bool, error) {
	data, err := os.ReadFile(filepath.Join(taskDir, manifestFilename))
	if errors.Is(err, os.ErrNotExist) {
		return manifest{}, false, nil
	}
	if err != nil {
		return manifest{}, false, fmt.Errorf("read active runner log manifest: %w", err)
	}

	var current manifest
	if err := json.Unmarshal(data, &current); err != nil {
		return manifest{}, false, fmt.Errorf("decode active runner log manifest: %w", err)
	}
	if current.NextSequence < 0 || current.TotalBytes < 0 || current.UpdatedAt.IsZero() {
		return manifest{}, false, errors.New("active runner log manifest is invalid")
	}
	return current, true, nil
}

func writeManifest(taskDir string, current manifest) error {
	data, err := json.Marshal(current)
	if err != nil {
		return fmt.Errorf("encode active runner log manifest: %w", err)
	}
	data = append(data, '\n')

	file, err := os.CreateTemp(taskDir, ".manifest-*.tmp")
	if err != nil {
		return fmt.Errorf("create active runner log manifest: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := writeAll(file, data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write active runner log manifest: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync active runner log manifest: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close active runner log manifest: %w", err)
	}
	if err := os.Rename(tempPath, filepath.Join(taskDir, manifestFilename)); err != nil {
		return fmt.Errorf("replace active runner log manifest: %w", err)
	}

	directory, err := os.Open(taskDir)
	if err != nil {
		return fmt.Errorf("open active runner log directory: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync active runner log directory: %w", err)
	}
	return nil
}

func appendAndSync(taskDir string, committedBytes int64, content []byte) error {
	file, err := os.OpenFile(
		filepath.Join(taskDir, dataFilename),
		os.O_CREATE|os.O_RDWR,
		0o600,
	)
	if err != nil {
		return fmt.Errorf("open active runner log data: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat active runner log data: %w", err)
	}
	if info.Size() < committedBytes {
		return errors.New("active runner log data is shorter than its manifest")
	}
	if info.Size() > committedBytes {
		if err := file.Truncate(committedBytes); err != nil {
			return fmt.Errorf("discard uncommitted active runner log data: %w", err)
		}
	}
	if _, err := file.Seek(committedBytes, io.SeekStart); err != nil {
		return fmt.Errorf("seek active runner log data: %w", err)
	}
	if err := writeAll(file, content); err != nil {
		return fmt.Errorf("append active runner log data: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync active runner log data: %w", err)
	}
	return nil
}

func repairDataFile(taskDir string, committedBytes int64) error {
	path := filepath.Join(taskDir, dataFilename)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open active runner log data: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat active runner log data: %w", err)
	}
	if info.Size() < committedBytes {
		return errors.New("active runner log data is shorter than its manifest")
	}
	if info.Size() == committedBytes {
		return nil
	}
	if err := file.Truncate(committedBytes); err != nil {
		return fmt.Errorf("discard uncommitted active runner log data: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync repaired active runner log data: %w", err)
	}
	return nil
}

func writeAll(writer io.Writer, content []byte) error {
	for len(content) > 0 {
		written, err := writer.Write(content)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		content = content[written:]
	}
	return nil
}

func parseCursor(cursor string) (int64, error) {
	if cursor == "" {
		return 0, nil
	}
	offset, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || offset < 0 {
		return 0, api.ErrInvalidCursor
	}
	return offset, nil
}

func formatCursor(offset int64) string {
	return strconv.FormatInt(offset, 10)
}

var _ api.Store = (*Store)(nil)
