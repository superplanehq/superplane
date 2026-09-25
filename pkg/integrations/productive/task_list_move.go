package productive

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// TaskListMove is a Productive.io task changing task lists. From and To are
// task list ids. An empty From means the task had no list.
type TaskListMove struct {
	From string
	To   string
}

// latestTaskUpdateChangeset reads the newest task-update changeset.
// Productive.io webhooks carry the task after the change, not the fields
// that changed, so the activity feed is what shows a list move.
func (c *Client) latestTaskUpdateChangeset(taskID string) (any, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("filter[task_id]", taskID)
	params.Set("filter[event]", "update")
	params.Set("filter[item_type]", "task")
	params.Set("filter[type]", "2")
	params.Set("sort", "-created_at")
	params.Set("page[size]", "10")

	body, err := c.execRequest(http.MethodGet, fmt.Sprintf("%s/activities?%s", c.BaseURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	response := resourceListResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing task activities: %v", err)
	}

	newest, ok := newestActivity(response.Data)
	if !ok {
		return nil, nil
	}
	return newest.Attributes["changeset"], nil
}

func newestActivity(activities []resourceDocument) (resourceDocument, bool) {
	if len(activities) == 0 {
		return resourceDocument{}, false
	}

	newest := activities[0]
	newestAt := stringAttribute(newest.Attributes["created_at"])
	for _, activity := range activities[1:] {
		createdAt := stringAttribute(activity.Attributes["created_at"])
		if createdAt > newestAt {
			newest = activity
			newestAt = createdAt
		}
	}
	return newest, true
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
