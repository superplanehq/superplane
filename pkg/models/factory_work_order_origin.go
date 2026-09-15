package models

import (
	"net/url"
	"slices"
	"strings"
)

var preferredIntakeOriginURLKeys = []string{"html_url", "permalink", "web_url", "url"}

// WorkOrderOrigin is the external ticket a work order was created from.
type WorkOrderOrigin struct {
	URL   string
	Label string
}

func OriginFromIntakePayload(payload map[string]any) *WorkOrderOrigin {
	originURL := firstHTTPURL(payload, 0)
	if originURL == "" {
		return nil
	}

	return &WorkOrderOrigin{
		URL:   originURL,
		Label: OriginLabelFromURL(originURL),
	}
}

func OriginFromIntakeRootEvent(event *CanvasEvent) *WorkOrderOrigin {
	if event == nil {
		return nil
	}

	payload, ok := RootEventSourcePayload(event.Data.Data()).(map[string]any)
	if !ok {
		return nil
	}

	return OriginFromIntakePayload(payload)
}

func OriginLabelFromURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return strings.TrimSpace(rawURL)
	}

	if label := githubOriginLabel(parsed); label != "" {
		return label
	}

	return lastPathSegment(parsed)
}

func applyWorkOrderOrigin(order *FactoryWorkOrder, origin *WorkOrderOrigin) {
	if order == nil || origin == nil {
		return
	}

	originURL := strings.TrimSpace(origin.URL)
	if originURL == "" {
		return
	}

	order.OriginURL = &originURL
	label := strings.TrimSpace(origin.Label)
	if label == "" {
		label = OriginLabelFromURL(originURL)
	}
	if label != "" {
		order.OriginLabel = &label
	}
}

func firstHTTPURL(value any, depth int) string {
	if depth > 6 || value == nil {
		return ""
	}

	switch current := value.(type) {
	case string:
		if isHTTPURL(current) {
			return strings.TrimSpace(current)
		}
	case map[string]any:
		for _, key := range preferredIntakeOriginURLKeys {
			if found := firstHTTPURL(current[key], depth+1); found != "" {
				return found
			}
		}

		keys := make([]string, 0, len(current))
		for key := range current {
			if slices.Contains(preferredIntakeOriginURLKeys, key) {
				continue
			}
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			if found := firstHTTPURL(current[key], depth+1); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range current {
			if found := firstHTTPURL(child, depth+1); found != "" {
				return found
			}
		}
	}

	return ""
}

func isHTTPURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func githubOriginLabel(parsed *url.URL) string {
	if parsed.Hostname() != "github.com" {
		return ""
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 {
		return ""
	}

	owner, repo, kind, number := parts[0], parts[1], parts[2], parts[3]
	if owner == "" || repo == "" || number == "" {
		return ""
	}
	if kind != "issues" && kind != "pull" {
		return ""
	}

	return owner + "/" + repo + "#" + number
}

func lastPathSegment(parsed *url.URL) string {
	parts := strings.FieldsFunc(parsed.Path, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return strings.TrimSpace(parsed.String())
	}
	return parts[len(parts)-1]
}
