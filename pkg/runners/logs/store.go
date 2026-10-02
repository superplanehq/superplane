package logs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric"
	"gorm.io/gorm"
)

const (
	StoreFS                = "fs"
	StorePostgres          = "postgres"
	MaxRetainedBytes int64 = 10 * 1024 * 1024
	SafetyExpiration       = 7 * 24 * time.Hour

	TruncationRecord = `{"type":"line","text":"SuperPlane stopped retaining logs because this task reached the 10 MiB log limit."}` + "\n"

	HeaderCursor = "X-SuperPlane-Log-Cursor"
	HeaderState  = "X-SuperPlane-Log-State"
	HeaderReset  = "X-SuperPlane-Log-Reset"
)

var (
	ErrNotFound         = errors.New("active runner log not found")
	ErrSequenceConflict = errors.New("active runner log sequence conflict")
	ErrInvalidCursor    = errors.New("invalid active runner log cursor")
)

type AppendResult struct {
	Truncated bool
}

type ReadResult struct {
	Content   io.ReadCloser
	Cursor    string
	Truncated bool
}

type SetupContext struct {
	Context       context.Context
	Database      *gorm.DB
	MeterProvider metric.MeterProvider
}

/*
 * Store owns chunks and all append-frequency metadata needed to order,
 * deduplicate, limit, and read them.
 *
 * An implementation backed by an external system must keep that metadata in
 * the external system. Accepting a chunk must not write sequence, byte-count,
 * or truncation metadata to the application PostgreSQL database. This rule
 * keeps PostgreSQL out of the active log write path when another store is
 * selected.
 */
type Store interface {

	/*
	 * Name returns the stable identifier persisted in RunnerTaskLogLifecycle.
	 * Changing it would make active logs from older tasks unavailable.
	 */
	Name() string

	/*
	 * Setup prepares and validates the selected store during application startup.
	 * It must be idempotent and safe for concurrent application replicas.
	 * The application does not serve log requests if Setup fails.
	 */
	Setup(SetupContext) error

	/*
	 * Initialize creates the active-store metadata for a task before the runner
	 * receives it. It is idempotent. Implementations with multiple storage
	 * locations use this call to pin the task to the current primary location.
	 */
	Initialize(context.Context, uuid.UUID) error

	/*
	 * Append durably stores one runner chunk and its sequence metadata before returning success.
	 * Initialize must complete before the first append.
	 * It accepts only the next sequence, treats an already accepted sequence as an idempotent duplicate,
	 * and returns ErrSequenceConflict for a future sequence.
	 * It also enforces the retained log limit and reports when truncation occurs.
	 */
	Append(context.Context, uuid.UUID, int64, []byte) (AppendResult, error)

	/*
	 * ReadAfter returns an ordered snapshot of retained NDJSON records after an opaque cursor.
	 * An empty cursor reads from the beginning. Callers pass the returned cursor to a later call
	 * to receive only newer records. ErrNotFound means that the task has no active-store record.
	 *
	 * The cursor is independent from the runner chunk sequence passed to Append.
	 * Callers must not create a cursor from a chunk sequence or interpret its value.
	 * PostgreSQL currently uses the next chunk sequence, while FS uses a committed
	 * byte offset. A client only stores and returns the latest cursor from ReadAfter.
	 */
	ReadAfter(context.Context, uuid.UUID, string) (*ReadResult, error)

	/*
	 * Delete idempotently removes all active chunks and append metadata for one task.
	 * The archiver calls it after the final blob is available and the live-reader grace period expires.
	 */
	Delete(context.Context, uuid.UUID) error

	/*
	 * DeleteExpired removes abandoned active logs whose last activity is older than the supplied time.
	 * It is the safety cleanup path for tasks that did not complete normal archival and returns the number of task logs removed.
	 */
	DeleteExpired(context.Context, time.Time) (int64, error)
}

var (
	currentMu sync.RWMutex
	current   Store
)

func SetCurrent(store Store) {
	currentMu.Lock()
	defer currentMu.Unlock()
	current = store
}

func Current() Store {
	currentMu.RLock()
	defer currentMu.RUnlock()
	return current
}

func FinalKey(installationID string, organizationID, taskID uuid.UUID) string {
	return fmt.Sprintf(
		"%s/orgs/%s/runner-tasks/%s/logs/v1/logs.ndjson.gz",
		installationID,
		organizationID,
		taskID,
	)
}
