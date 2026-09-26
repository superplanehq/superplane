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
		Label: originLabelFromIntakePayload(originURL, payload),
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

func originLabelFromIntakePayload(originURL string, payload map[string]any) string {
	if label := sentryOriginLabel(originURL, payload); label != "" {
		return label
	}

	return OriginLabelFromURL(originURL)
}

func sentryOriginLabel(originURL string, payload map[string]any) string {
	if !isSentryOriginURL(originURL) {
		return ""
	}

	issue := sentryIssueForOriginURL(originURL, payload)
	if issue == nil {
		return ""
	}

	title, _ := issue["title"].(string)
	return normalizeOriginLabel(title)
}

func isSentryOriginURL(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Hostname())
	return host == "sentry.io" || strings.HasSuffix(host, ".sentry.io")
}

func sentryIssueForOriginURL(originURL string, payload map[string]any) map[string]any {
	if issue := findMapWithOriginURL(payload, originURL, 0); issue != nil {
		return issue
	}
	if data, ok := payload["data"].(map[string]any); ok {
		if issue, ok := data["issue"].(map[string]any); ok {
			return issue
		}
	}
	issue, _ := payload["issue"].(map[string]any)
	return issue
}

func findMapWithOriginURL(value any, originURL string, depth int) map[string]any {
	if depth > 6 || value == nil {
		return nil
	}

	switch current := value.(type) {
	case map[string]any:
		if originURLMatches(current["permalink"], originURL) || originURLMatches(current["web_url"], originURL) {
			return current
		}

		keys := make([]string, 0, len(current))
		for key := range current {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			if found := findMapWithOriginURL(current[key], originURL, depth+1); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range current {
			if found := findMapWithOriginURL(child, originURL, depth+1); found != nil {
				return found
			}
		}
	}

	return nil
}

func originURLMatches(value any, originURL string) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(raw) == originURL
}

func normalizeOriginLabel(title string) string {
	return strings.Join(strings.Fields(title), " ")
}

func githubOriginLabel(parsed *url.URL) string {
	if parsed.Hostname() != "github.com" {
		return ""
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 {
		return ""
	}

	owner, repo := parts[0], parts[1]
	if owner == "" || repo == "" {
		return ""
	}
	if len(parts) >= 5 && parts[2] == "security" && parts[3] == "dependabot" && parts[4] != "" {
		return owner + "/" + repo + " dependabot #" + parts[4]
	}
	if len(parts) == 4 && parts[2] == "security" && parts[3] == "dependabot" {
		return dependabotPackageOriginLabel(parsed)
	}

	kind, number := parts[2], parts[3]
	if number == "" {
		return ""
	}
	if kind != "issues" && kind != "pull" {
		return ""
	}

	return owner + "/" + repo + "#" + number
}

// dependabotPackageOriginLabel names the package of an alerts page filtered
// with a `package:` qualifier, such as
// https://github.com/acme/payments/security/dependabot?q=is%3Aopen+package%3Alodash.
func dependabotPackageOriginLabel(parsed *url.URL) string {
	for _, qualifier := range strings.Fields(parsed.Query().Get("q")) {
		if name, ok := strings.CutPrefix(qualifier, "package:"); ok && name != "" {
			return "Dependabot: " + name
		}
	}
	return ""
}

func lastPathSegment(parsed *url.URL) string {
	parts := strings.FieldsFunc(parsed.Path, func(r rune) bool { return r == '/' })
	if len(parts) == 0 {
		return strings.TrimSpace(parsed.String())
	}
	return parts[len(parts)-1]
}
