package productive

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TaskListMove is a Productive.io task changing task lists. From and To are
// task list ids. An empty From means the task had no list.
type TaskListMove struct {
	From string
	To   string
}

const (
	// taskActivityMatchWindow is how far an activity may sit from the webhook
	// timestamp and still belong to that delivery. A later edit must not replace
	// the change this delivery reported.
	taskActivityMatchWindow = 2 * time.Minute

	// taskActivityPageSize is how many activities each page requests.
	taskActivityPageSize = 20

	// maxTaskActivityPages bounds how many pages the lookup walks, so a
	// misbehaving pagination cursor cannot loop forever.
	maxTaskActivityPages = 20
)

// errTaskUpdateActivityUnavailable means this delivery's changeset is not
// available yet. The webhook handler returns an error so Productive.io retries.
var errTaskUpdateActivityUnavailable = errors.New("task update activity unavailable")

type taskActivity struct {
	at        time.Time
	changeset any
}

// taskUpdateChangesetAt reads the changeset for one delivered update.
// Productive.io webhooks carry the task after the change, not the fields
// that changed. The activity feed shows the change, matched to this delivery
// by time and by the task state in the webhook.
func (c *Client) taskUpdateChangesetAt(taskID string, deliveredAt time.Time, document map[string]any) (any, bool, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" || deliveredAt.IsZero() {
		return nil, false, nil
	}

	activities := []taskActivity{}
	var changeset any
	var bestDelta time.Duration
	found := false
	for page := 1; page <= maxTaskActivityPages; page++ {
		body, err := c.execRequest(http.MethodGet, c.taskActivityURL(taskID, deliveredAt, page), nil)
		if err != nil {
			// This page can still hold the closest change. Fail the lookup so
			// Productive.io retries, instead of emitting an older match.
			return nil, false, err
		}

		response := resourceListResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, false, fmt.Errorf("error parsing task activities: %v", err)
		}

		pageActivities := taskActivitiesFromDocuments(response.Data)
		activities = append(activities, pageActivities...)
		changeset, bestDelta, found = activityForDelivery(activities, deliveredAt, document)
		if len(response.Data) < taskActivityPageSize || activityPageBeforeWindow(pageActivities, deliveredAt) {
			break
		}
		// Later pages are older. Stop once none of them can sit closer to
		// this delivery than the activity already found.
		if found && !laterPageCanBeCloser(activities, deliveredAt, bestDelta) {
			break
		}
	}

	return changeset, found, nil
}

func (c *Client) taskActivityURL(taskID string, deliveredAt time.Time, page int) string {
	params := url.Values{}
	params.Set("filter[task_id]", taskID)
	params.Set("filter[event]", "update")
	params.Set("filter[item_type]", "task")
	params.Set("filter[type]", "2")
	params.Set("filter[after]", deliveredAt.Add(-taskActivityMatchWindow).Format(time.RFC3339Nano))
	params.Set("filter[before]", deliveredAt.Add(taskActivityMatchWindow).Format(time.RFC3339Nano))
	params.Set("sort", "-created_at")
	params.Set("page[number]", strconv.Itoa(page))
	params.Set("page[size]", strconv.Itoa(taskActivityPageSize))
	return fmt.Sprintf("%s/activities?%s", c.BaseURL, params.Encode())
}

func taskActivitiesFromDocuments(documents []resourceDocument) []taskActivity {
	activities := make([]taskActivity, 0, len(documents))
	for _, activity := range documents {
		at, ok := parseActivityTime(stringAttribute(activity.Attributes["created_at"]))
		if !ok {
			continue
		}
		activities = append(activities, taskActivity{at: at, changeset: activity.Attributes["changeset"]})
	}
	return activities
}

// activityPageBeforeWindow reports that this page is older than the match
// window. Later pages are older because the query sorts newest first.
func activityPageBeforeWindow(activities []taskActivity, deliveredAt time.Time) bool {
	if len(activities) == 0 {
		return false
	}

	newest := activities[0].at
	for _, activity := range activities[1:] {
		if activity.at.After(newest) {
			newest = activity.at
		}
	}
	return newest.Before(deliveredAt.Add(-taskActivityMatchWindow))
}

func activityForDelivery(activities []taskActivity, deliveredAt time.Time, document map[string]any) (any, time.Duration, bool) {
	if deliveredAt.IsZero() {
		return nil, 0, false
	}

	var changeset any
	var bestDelta time.Duration
	found := false
	for _, activity := range activities {
		if activity.at.IsZero() || activity.changeset == nil {
			continue
		}
		delta := activity.at.Sub(deliveredAt)
		if delta < 0 {
			delta = -delta
		}
		if delta > taskActivityMatchWindow {
			continue
		}
		if !changesetMatchesDocument(activity.changeset, document) {
			continue
		}
		if found && delta >= bestDelta {
			continue
		}
		changeset = activity.changeset
		bestDelta = delta
		found = true
	}
	return changeset, bestDelta, found
}

// laterPageCanBeCloser reports whether an older page can beat bestDelta.
// The query sorts newest first, so later pages are older than the oldest
// activity already read.
func laterPageCanBeCloser(activities []taskActivity, deliveredAt time.Time, bestDelta time.Duration) bool {
	if bestDelta == 0 {
		return false
	}

	oldest, ok := oldestActivityTime(activities)
	if !ok || oldest.After(deliveredAt) {
		return true
	}
	return deliveredAt.Sub(oldest) < bestDelta
}

func oldestActivityTime(activities []taskActivity) (time.Time, bool) {
	var oldest time.Time
	found := false
	for _, activity := range activities {
		if activity.at.IsZero() {
			continue
		}
		if !found || activity.at.Before(oldest) {
			oldest = activity.at
			found = true
		}
	}
	return oldest, found
}

func changesetMatchesDocument(changeset any, document map[string]any) bool {
	if listID, ok := changesetListAfter(changeset); ok && listID != taskListID(document) {
		return false
	}
	if title, ok := changesetTitleAfter(changeset); ok && title != taskAttribute(document, "title") {
		return false
	}
	return true
}

func changesetListAfter(changeset any) (string, bool) {
	fields := changesetFields(changeset)
	for _, key := range []string{"task_list_id", "milestone_id", "task_list"} {
		value, found := fields[key]
		if !found {
			continue
		}
		_, to, ok := changePair(value)
		if ok {
			return to, true
		}
	}
	return "", false
}

func changesetTitleAfter(changeset any) (string, bool) {
	_, after, ok := changePair(changesetFields(changeset)["title"])
	if !ok {
		return "", false
	}
	return after, true
}

func taskAttribute(document map[string]any, name string) string {
	attributes, _ := document["attributes"].(map[string]any)
	return strings.TrimSpace(changesetValue(attributes[name]))
}

func deliveryCreatedAt(body []byte) (time.Time, bool) {
	var payload struct {
		Created string `json:"created"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return time.Time{}, false
	}
	return parseActivityTime(payload.Created)
}

func parseActivityTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func taskListMoveFromChangeset(changeset any) (TaskListMove, bool) {
	fields := changesetFields(changeset)
	for _, key := range []string{"task_list_id", "milestone_id", "task_list"} {
		value, found := fields[key]
		if !found {
			continue
		}
		from, to, ok := changePair(value)
		if !ok || to == "" || from == to {
			return TaskListMove{}, false
		}
		return TaskListMove{From: from, To: to}, true
	}
	return TaskListMove{}, false
}

func changesetFields(changeset any) map[string]any {
	switch typed := changeset.(type) {
	case map[string]any:
		return lowerFieldKeys(typed)
	case []any:
		fields := map[string]any{}
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(changesetValue(firstField(entry, "attribute", "field", "name"))))
			if name == "" {
				for key, value := range lowerFieldKeys(entry) {
					fields[key] = value
				}
				continue
			}
			fields[name] = entry
		}
		return fields
	default:
		return nil
	}
}

func lowerFieldKeys(fields map[string]any) map[string]any {
	lowered := make(map[string]any, len(fields))
	for key, value := range fields {
		lowered[strings.ToLower(strings.TrimSpace(key))] = value
	}
	return lowered
}

func changePair(value any) (string, string, bool) {
	switch typed := value.(type) {
	case []any:
		if len(typed) < 2 {
			return "", "", false
		}
		return changesetValue(typed[0]), changesetValue(typed[1]), true
	case map[string]any:
		before, beforeOK := namedChangesetValue(typed, "old", "from", "previous")
		after, afterOK := namedChangesetValue(typed, "new", "to", "current")
		if !beforeOK || !afterOK {
			return "", "", false
		}
		return before, after, true
	default:
		return "", "", false
	}
}

func namedChangesetValue(fields map[string]any, names ...string) (string, bool) {
	lowered := lowerFieldKeys(fields)
	for _, name := range names {
		value, ok := lowered[name]
		if !ok {
			continue
		}
		return changesetValue(value), true
	}
	return "", false
}

func firstField(fields map[string]any, names ...string) any {
	lowered := lowerFieldKeys(fields)
	for _, name := range names {
		value, ok := lowered[name]
		if ok {
			return value
		}
	}
	return nil
}

func changesetValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}
