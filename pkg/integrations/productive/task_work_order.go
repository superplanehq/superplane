package productive

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// TaskRef identifies one Productive.io task in one Productive organization.
// Task ids are unique inside an organization, not across organizations.
type TaskRef struct {
	OrganizationID string
	TaskID         string
}

// LockTaskWorkOrder serializes work-order creation for one Productive.io
// task in this factory. The lock is held until the caller commits tx.
func LockTaskWorkOrder(tx *gorm.DB, factory *models.Factory, ref TaskRef) error {
	if tx == nil || factory == nil {
		return nil
	}
	ref.TaskID = strings.TrimSpace(ref.TaskID)
	if ref.TaskID == "" {
		return nil
	}
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", taskWorkOrderLockKey(factory.ID, ref)).Error
}

func taskWorkOrderLockKey(factoryID uuid.UUID, ref TaskRef) int64 {
	sum := sha256.New()
	sum.Write([]byte("productive-task-work-order:"))
	sum.Write(factoryID[:])
	sum.Write([]byte(productiveOrganizationKey(ref.OrganizationID)))
	sum.Write([]byte{0})
	sum.Write([]byte(strings.TrimSpace(ref.TaskID)))
	digest := sum.Sum(nil)
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

// TaskHasWorkOrder reports whether this factory already has a work order
// for this Productive.io task. Matching covers every work-order state, both
// by origin URL and by an origin-less work order whose source run is this
// task (webhook bodies used to omit the page URL).
func TaskHasWorkOrder(tx *gorm.DB, factory *models.Factory, ref TaskRef) (bool, error) {
	ref.TaskID = strings.TrimSpace(ref.TaskID)
	ref.OrganizationID = strings.TrimSpace(ref.OrganizationID)
	if factory == nil || ref.TaskID == "" {
		return false, nil
	}

	urls, err := factory.ListWorkOrderOriginURLsContaining(tx, TaskURLFragment(ref.TaskID))
	if err != nil {
		return false, err
	}
	for _, rawURL := range urls {
		found, ok := TaskRefFromURL(rawURL)
		if ok && taskRefMatches(ref, found) {
			return true, nil
		}
	}

	return originlessEventHasTask(tx, factory, ref)
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

// TaskRefFromEventData reads the Productive.io task id and organization from
// a canvas root event. The organization comes from the task page URL. Older
// events omit that URL, so OrganizationID can be empty.
func TaskRefFromEventData(eventData any) (TaskRef, bool) {
	taskID, ok := TaskIDFromEventData(eventData)
	if !ok {
		return TaskRef{}, false
	}

	ref := TaskRef{TaskID: taskID}
	envelope, _ := eventData.(map[string]any)
	outer, _ := envelope["data"].(map[string]any)
	rawURL, _ := outer["url"].(string)
	if fromURL, ok := TaskRefFromURL(rawURL); ok {
		ref.OrganizationID = fromURL.OrganizationID
	}
	return ref, true
}

// TaskRefFromURL reads the Productive.io organization and task id from a
// task page URL.
func TaskRefFromURL(rawURL string) (TaskRef, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || !strings.EqualFold(parsed.Hostname(), "app.productive.io") {
		return TaskRef{}, false
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "tasks" || parts[2] == "" {
		return TaskRef{}, false
	}
	return TaskRef{OrganizationID: parts[0], TaskID: parts[2]}, true
}

// TaskIDFromURL reads the Productive.io task id from a task page URL.
func TaskIDFromURL(rawURL string) (string, bool) {
	ref, ok := TaskRefFromURL(rawURL)
	return ref.TaskID, ok
}

func taskRefMatches(want, found TaskRef) bool {
	if want.TaskID == "" || found.TaskID != want.TaskID {
		return false
	}
	if want.OrganizationID == "" || found.OrganizationID == "" {
		return true
	}
	return sameProductiveOrganization(want.OrganizationID, found.OrganizationID)
}

// productiveOrganizationKey collapses a numeric id and its page slug
// (12345 and 12345-acme) to the same lock and match key.
func productiveOrganizationKey(organizationID string) string {
	organizationID = strings.ToLower(strings.TrimSpace(organizationID))
	number, _, _ := strings.Cut(organizationID, "-")
	if number == "" || !isDigits(number) {
		return organizationID
	}
	return number
}

func sameProductiveOrganization(left, right string) bool {
	return productiveOrganizationKey(left) == productiveOrganizationKey(right)
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// originlessEventHasTask matches one task id in SQL, then confirms the
// organization from the page URL. An event that never stored a URL can only
// be matched by task id.
func originlessEventHasTask(tx *gorm.DB, factory *models.Factory, ref TaskRef) (bool, error) {
	if tx == nil {
		return false, nil
	}

	query := fmt.Sprintf(`
SELECT DISTINCT events.data #>> '{data,url}' AS page_url
FROM %s AS orders
INNER JOIN %s AS executions
	ON executions.run_id = orders.source_run_id
INNER JOIN %s AS events
	ON events.id = executions.root_event_id
WHERE orders.organization_id = ?
	AND orders.factory_id = ?
	AND orders.origin_url IS NULL
	AND orders.source_run_id IS NOT NULL
	AND events.data ->> 'type' = ?
	AND events.data #>> '{data,data,id}' = ?
`,
		(models.FactoryWorkOrder{}).TableName(),
		(&models.CanvasNodeExecution{}).TableName(),
		(&models.CanvasEvent{}).TableName(),
	)

	var rows []struct {
		PageURL *string `gorm:"column:page_url"`
	}
	err := tx.Raw(query,
		factory.OrganizationID,
		factory.ID,
		TaskPayloadType,
		ref.TaskID,
	).Scan(&rows).Error
	if err != nil {
		return false, err
	}

	for _, row := range rows {
		if row.PageURL == nil || strings.TrimSpace(*row.PageURL) == "" {
			return true, nil
		}
		found, ok := TaskRefFromURL(*row.PageURL)
		if ok && taskRefMatches(ref, found) {
			return true, nil
		}
	}
	return false, nil
}
