package datadog

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func parseErrorSample(decoded any, source string) *ErrorSample {
	best := (*ErrorSample)(nil)
	for _, event := range eventList(decoded) {
		sample := sampleFromValue(event, source)
		if sample == nil {
			continue
		}
		if best == nil || len(sample.Stack) > len(best.Stack) {
			best = sample
		}
	}
	return best
}

func parseRelatedLogs(decoded any) []LogLine {
	events := eventList(decoded)
	logs := make([]LogLine, 0, len(events))
	for _, event := range events {
		line := logLineFromValue(event)
		if line.Message == "" && line.ErrorKind == "" && line.HTTPPath == "" {
			continue
		}
		logs = append(logs, line)
		if len(logs) >= maxRelatedLogs {
			break
		}
	}
	return logs
}

func eventList(decoded any) []any {
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil
	}
	data, ok := root["data"].([]any)
	if !ok {
		return nil
	}
	return data
}

func sampleFromValue(value any, source string) *ErrorSample {
	sample := &ErrorSample{
		Source:      source,
		Timestamp:   lookupTime(value, "start_timestamp", "timestamp"),
		TraceID:     lookupString(value, "otel.trace_id", "dd.trace_id", "trace_id"),
		SpanID:      lookupString(value, "otel.span_id", "dd.span_id", "span_id"),
		Env:         lookupString(value, "deployment.environment.name", "env"),
		Version:     lookupString(value, "version", "git.version"),
		Host:        lookupString(value, "host", "hostname"),
		Resource:    lookupString(value, "resource_name", "resource"),
		HTTPMethod:  lookupString(value, "http.method", "http.request.method"),
		HTTPPath:    lookupString(value, "http.path", "http.route", "http.url_details.path"),
		HTTPStatus:  lookupString(value, "http.status_code", "http.status"),
		UserID:      lookupString(value, "usr.id", "user.id"),
		RequestID:   lookupString(value, "request_id", "http.request_id"),
		Fingerprint: lookupString(value, "error.fingerprint"),
		Origin:      lookupString(value, "source", "error.source"),
		Route:       lookupString(value, "fe.route", "view.url_path", "view.url"),
		Action:      lookupString(value, "fe.action", "action.target.name"),
		Breadcrumbs: breadcrumbs(lookupValue(value, "breadcrumb"), lookupValue(value, "breadcrumbs")),
		Stack:       limitRunes(longestStack(value), maxErrorSampleStackRunes),
	}
	if sample.isEmpty() {
		return nil
	}
	return sample
}

func (s *ErrorSample) isEmpty() bool {
	if s == nil {
		return true
	}
	return s.Stack == "" &&
		s.TraceID == "" &&
		s.SpanID == "" &&
		s.Resource == "" &&
		s.HTTPPath == "" &&
		s.RequestID == "" &&
		s.UserID == "" &&
		s.Route == "" &&
		s.Action == ""
}

// breadcrumbs reads user steps before the error. Datadog stores them as a
// list or as a map keyed by position ("00", "01", ...).
func breadcrumbs(values ...any) []string {
	for _, value := range values {
		if steps := breadcrumbSteps(value); len(steps) > 0 {
			return steps
		}
	}
	return nil
}

func breadcrumbSteps(value any) []string {
	var items []any
	switch typed := value.(type) {
	case []any:
		items = typed
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			items = append(items, typed[key])
		}
	default:
		return nil
	}

	steps := make([]string, 0, len(items))
	for _, item := range items {
		step := breadcrumbText(item)
		if step == "" {
			continue
		}
		steps = append(steps, limitRunes(step, maxRelatedLogMessageRunes))
		if len(steps) >= maxBreadcrumbs {
			break
		}
	}
	return steps
}

func breadcrumbText(item any) string {
	if text := scalarString(item); text != "" {
		return text
	}
	fields, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	parts := []string{}
	for _, key := range []string{"category", "type", "message", "url"} {
		if text := scalarString(fields[key]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, " ")
}

func logLineFromValue(value any) LogLine {
	message := lookupString(value, "message")
	return LogLine{
		Timestamp:  lookupTime(value, "timestamp"),
		Status:     lookupString(value, "status", "severity_text"),
		Service:    lookupString(value, "service"),
		Message:    limitRunes(message, maxRelatedLogMessageRunes),
		HTTPMethod: lookupString(value, "http.method", "http.request.method"),
		HTTPPath:   lookupString(value, "http.path", "http.route"),
		HTTPStatus: lookupString(value, "http.status_code", "http.status"),
		ErrorKind:  lookupString(value, "error.kind", "error.type", "exception.type"),
	}
}

func lookupString(value any, keys ...string) string {
	for _, key := range keys {
		if found := lookup(value, key); found != "" {
			return found
		}
	}
	return ""
}

func lookupTime(value any, keys ...string) time.Time {
	for _, key := range keys {
		if stamp := parseDatadogTimeValue(lookupValue(value, key)); !stamp.IsZero() {
			return stamp
		}
	}
	return time.Time{}
}

func lookup(value any, key string) string {
	return scalarString(lookupValue(value, key))
}

func lookupValue(value any, key string) any {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return nil
	}
	return lookupAt(value, strings.Split(key, "."))
}

func lookupAt(value any, parts []string) any {
	switch typed := value.(type) {
	case map[string]any:
		if got := mapLookup(typed, parts); got != nil {
			return got
		}
		names := make([]string, 0, len(typed))
		for name := range typed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if got := lookupAt(typed[name], parts); got != nil {
				return got
			}
		}
	case []any:
		for _, child := range typed {
			if got := lookupAt(child, parts); got != nil {
				return got
			}
		}
	}
	return nil
}

func mapLookup(m map[string]any, parts []string) any {
	current := any(m)
	for _, part := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		next, ok := mapKeyCI(obj, part)
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

func mapKeyCI(m map[string]any, key string) (any, bool) {
	if value, ok := m[key]; ok {
		return value, true
	}
	for name, value := range m {
		if strings.EqualFold(name, key) {
			return value, true
		}
	}
	return nil, false
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	default:
		return ""
	}
}

func parseDatadogTimeValue(value any) time.Time {
	switch typed := value.(type) {
	case string:
		return parseDatadogTime(typed)
	case json.Number:
		if millis, err := typed.Int64(); err == nil {
			return unixMilliTime(millis)
		}
	case float64:
		return unixMilliTime(int64(typed))
	case int64:
		return unixMilliTime(typed)
	}
	return time.Time{}
}

func longestStack(value any) string {
	best := ""
	walkStack(value, &best)
	return strings.TrimSpace(best)
}

func walkStack(value any, best *string) {
	switch typed := value.(type) {
	case map[string]any:
		names := make([]string, 0, len(typed))
		for name := range typed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			child := typed[name]
			if text := scalarString(child); isStackKey(name) && looksLikeStack(text) && len(text) > len(*best) {
				*best = text
			}
			walkStack(child, best)
		}
	case []any:
		for _, child := range typed {
			walkStack(child, best)
		}
	}
}

func isStackKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "stack", "stacktrace", "stack_trace", "handling_stack", "details":
		return true
	}
	return strings.HasSuffix(key, ".stack") ||
		strings.HasSuffix(key, ".stacktrace") ||
		strings.HasSuffix(key, ".handling_stack")
}

func looksLikeStack(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) < 20 {
		return false
	}
	return strings.Contains(text, "\n") || strings.Contains(text, " at ")
}

func telemetryIDs(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}

	add(value)
	if isDecimalID(value) {
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			add(strconv.FormatUint(parsed, 16))
		}
		return ids
	}

	hex := strings.TrimPrefix(strings.ToLower(value), "0x")
	if !isHexID(hex) {
		return ids
	}
	add(hex)
	if len(hex) > 16 {
		add(hex[len(hex)-16:])
	}
	if decimal := decimalLower64(hex); decimal != "" {
		add(decimal)
	}
	return ids
}

func decimalLower64(value string) string {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if value == "" {
		return ""
	}
	if isDecimalID(value) {
		return value
	}
	if !isHexID(value) {
		return ""
	}
	if len(value) > 16 {
		value = value[len(value)-16:]
	}
	parsed, err := strconv.ParseUint(value, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(parsed, 10)
}

func isDecimalID(value string) bool {
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

func isHexID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		isDigit := r >= '0' && r <= '9'
		isLower := r >= 'a' && r <= 'f'
		isUpper := r >= 'A' && r <= 'F'
		if !isDigit && !isLower && !isUpper {
			return false
		}
	}
	return true
}
