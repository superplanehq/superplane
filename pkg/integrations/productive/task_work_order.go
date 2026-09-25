package productive

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// LockTaskWorkOrder serializes work-order creation for one Productive.io
// task in this factory. The lock is held until the caller commits tx.
func LockTaskWorkOrder(tx *gorm.DB, factory *models.Factory, taskID string) error {
	if tx == nil || factory == nil {
		return nil
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", taskWorkOrderLockKey(factory.ID, taskID)).Error
}

func taskWorkOrderLockKey(factoryID uuid.UUID, taskID string) int64 {
	sum := sha256.New()
	sum.Write([]byte("productive-task-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(taskID))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// TaskHasWorkOrder reports whether this factory already has a work order
// for the Productive.io task. Matching covers every work-order state, both
// by origin URL and by an origin-less work order whose source run is this
// task (webhook bodies used to omit the page URL).
func TaskHasWorkOrder(tx *gorm.DB, factory *models.Factory, taskID string) (bool, error) {
	taskID = strings.TrimSpace(taskID)
	if factory == nil || taskID == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, TaskURLFragment(taskID))
	if err != nil {
		return false, err
	}
	for _, rawURL := range urls {
		found, ok := TaskIDFromURL(rawURL)
		if ok && found == taskID {
			return true, nil
		}
	}

	return sourceRunHasTask(tx, factory, taskID)
}

// TaskURLFragment is the origin-URL substring that identifies a Productive.io
// task. Callers use it for a coarse lookup, then confirm with TaskIDFromURL
// so `/tasks/9` cannot match `/tasks/91`.
func TaskURLFragment(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ""
	}
	return "/tasks/" + taskID
}

// TaskIDFromURL reads the Productive.io task id from a task page URL.
func TaskIDFromURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "app.productive.io") {
		return "", false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "tasks" || parts[2] == "" {
		return "", false
	}
	return parts[2], true
}

func sourceRunHasTask(tx *gorm.DB, factory *models.Factory, taskID string) (bool, error) {
	var runIDs []uuid.UUID
	err := tx.Model(&models.FactoryWorkOrder{}).
		Where("organization_id = ? AND factory_id = ?", factory.OrganizationID, factory.ID).
		Where("source_run_id IS NOT NULL").
		Where("origin_url IS NULL").
		Pluck("source_run_id", &runIDs).
		Error
	if err != nil {
		return false, err
	}

	seen := map[uuid.UUID]bool{}
	for _, runID := range runIDs {
		if seen[runID] {
			continue
		}
		seen[runID] = true

		event, err := models.FindRootEventForRun(tx, runID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return false, err
		}
		found, ok := TaskIDFromEventData(event.Data.Data())
		if ok && found == taskID {
			return true, nil
		}
	}

	return false, nil
}
