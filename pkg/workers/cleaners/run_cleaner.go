package cleaners

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type RunCleanerMode int

const (
	RunCleanerModeRetention RunCleanerMode = iota
	RunCleanerModeCanvasTeardown
)

type RunCleanerOptions struct {
	Mode          RunCleanerMode
	ReferenceTime time.Time
	Canvas        *models.Canvas
}

func (o RunCleanerOptions) Validate() error {
	switch o.Mode {
	case RunCleanerModeRetention:
		if o.ReferenceTime.IsZero() {
			return fmt.Errorf("reference time is required for retention cleanup")
		}
	case RunCleanerModeCanvasTeardown:
		if o.Canvas == nil {
			return fmt.Errorf("canvas is required for canvas teardown cleanup")
		}
	default:
		return fmt.Errorf("unknown run cleaner mode")
	}

	return nil
}

type RunCleaner struct {
	tx      *gorm.DB
	options RunCleanerOptions
}

func NewRunCleaner(tx *gorm.DB, options RunCleanerOptions) (*RunCleaner, error) {
	if err := options.Validate(); err != nil {
		return nil, err
	}

	return &RunCleaner{
		tx:      tx,
		options: options,
	}, nil
}

func (c *RunCleaner) CleanBatch(limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	runs, err := c.lockRuns(limit)
	if err != nil {
		return 0, err
	}

	if len(runs) == 0 {
		return 0, nil
	}

	runIDs := make([]uuid.UUID, len(runs))
	for i, run := range runs {
		runIDs[i] = run.ID
	}

	if err := models.DeleteCanvasRunChains(c.tx, runIDs); err != nil {
		return 0, fmt.Errorf("delete run chains: %w", err)
	}

	return len(runs), nil
}

func (c *RunCleaner) lockRuns(limit int) ([]models.CanvasRun, error) {
	switch c.options.Mode {
	case RunCleanerModeRetention:
		return models.LockRetainedFinishedRuns(c.tx, c.options.ReferenceTime, limit)
	case RunCleanerModeCanvasTeardown:
		return c.options.Canvas.LockRunsForCleanup(c.tx, limit)
	default:
		return nil, fmt.Errorf("unknown run cleaner mode")
	}
}
