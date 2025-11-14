package telemetry

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/database"
)

func StartDatabaseLocksReporter(ctx context.Context) {
	if !dbLocksCountHistogramReady.Load() {
		return
	}

	if !dbLocksReporterInitializedFlag.CompareAndSwap(false, true) {
		return
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reportDatabaseLocks(ctx, database.Conn())
			}
		}
	}()
}

func reportDatabaseLocks(ctx context.Context, db *gorm.DB) {
	if !dbLocksCountHistogramReady.Load() {
		return
	}

	var count int64

	if err := db.
		Raw("select count(*) from pg_locks").
		Scan(&count).Error; err != nil {
		return
	}

	dbLocksCountHistogram.Record(ctx, count)
}
