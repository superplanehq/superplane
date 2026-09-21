package sentry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	maxStackFrames = 40
	maxBreadcrumbs = 15
)

// IssueDescription renders a Sentry issue and optional full event as a task
// body. Empty sections are omitted. It does not dump JSON.
func IssueDescription(issue any, event *IssueEventDetail) string {
	issueMap := asStringMap(issue)
	if len(issueMap) == 0 && event == nil {
		return ""
	}

	var b strings.Builder
	writeHighlights(&b, issueMap, event)
	writeMessage(&b, event)
	writeStackTrace(&b, event)
	writeHTTPRequest(&b, event)
	writeUser(&b, event)
	writeTags(&b, issueMap, event)
	writeContexts(&b, event)
	writeBreadcrumbs(&b, event)
	writeAdditionalData(&b, event)
	writeSDK(&b, event)
	writeReplayAndTrace(&b, event, issueMap)

	return strings.TrimSpace(b.String())
}

func writeHighlights(b *strings.Builder, issue map[string]any, event *IssueEventDetail) {
	lines := []string{}
	addHighlight := func(label, value string) {
		if value == "" {
			return
		}
		lines = append(lines, fmt.Sprintf("- **%s:** %s", label, value))
	}

	title := firstNonEmpty(mapString(issue, "title"), eventString(event, func(e *IssueEventDetail) string { return e.Title }))
	addHighlight("Title", title)
	addHighlight("Culprit", firstNonEmpty(mapString(issue, "culprit"), eventString(event, func(e *IssueEventDetail) string { return e.Culprit })))
	addHighlight("Project", projectLabel(issue))
	addHighlight("Short ID", mapString(issue, "shortId"))
	addHighlight("Level", firstNonEmpty(mapString(issue, "level"), tagValue(issue, event, "level")))
	addHighlight("Environment", tagValue(issue, event, "environment"))
	status := mapString(issue, "status")
	substatus := mapString(issue, "substatus")
	switch {
	case status != "" && substatus != "" && substatus != status:
		addHighlight("Status", status+" ("+substatus+")")
	case status != "":
		addHighlight("Status", status)
	default:
		addHighlight("Status", substatus)
	}
	addHighlight("Priority", mapString(issue, "priority"))
	addHighlight("Event type", eventString(event, func(e *IssueEventDetail) string { return e.Type }))
	if message := eventMessage(event); message != "" && message != title {
		addHighlight("Message", message)
	}
	addHighlight("Release", releaseLabel(issue, event))
	addHighlight("First seen", mapString(issue, "firstSeen"))
	addHighlight("Last seen", mapString(issue, "lastSeen"))
	addHighlight("Count", mapString(issue, "count"))
	addHighlight("Users affected", nonzeroScalar(issue["userCount"]))
	if assigned := assignedLabel(issue); assigned != "" {
		addHighlight("Assigned to", assigned)
	}

	if len(lines) == 0 {
		return
	}

	b.WriteString("## Highlights\n\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
}

func writeMessage(b *strings.Builder, event *IssueEventDetail) {
	if event == nil || event.HasStack() {
		return
	}

	message := eventMessage(event)
	if message == "" {
		return
	}

	b.WriteString("## Message\n\n")
	fmt.Fprintf(b, "%s\n\n", message)
}

func writeStackTrace(b *strings.Builder, event *IssueEventDetail) {
	blocks := exceptionBlocks(event)
	if len(blocks) == 0 {
		return
	}

	b.WriteString("## Stack Trace\n\n")
	for i, block := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		if block.heading != "" {
			fmt.Fprintf(b, "%s\n\n", block.heading)
		}
		b.WriteString("```\n")
		b.WriteString(strings.Join(block.frames, "\n"))
		b.WriteString("\n```\n")
	}
	b.WriteString("\n")
}

func writeHTTPRequest(b *strings.Builder, event *IssueEventDetail) {
	request := entryData(event, "request")
	if len(request) == 0 {
		return
	}

	method := firstNonEmpty(mapString(request, "method"), mapString(request, "httpMethod"))
	urlValue := firstNonEmpty(mapString(request, "url"), mapString(request, "urlPath"))
	line := strings.TrimSpace(strings.Join([]string{method, urlValue}, " "))
	query := formatAny(request["query_string"])
	if query == "" {
		query = formatAny(request["queryString"])
	}
	headers := safeRequestHeaders(request["headers"])
	body := formatAny(request["data"])

	if line == "" && query == "" && len(headers) == 0 && body == "" {
		return
	}

	b.WriteString("## HTTP Request\n\n")
	if line != "" {
		fmt.Fprintf(b, "%s\n", line)
	}
	if query != "" {
		fmt.Fprintf(b, "\nQuery: %s\n", query)
	}
	if len(headers) > 0 {
		if line != "" || query != "" {
			b.WriteString("\n")
		}
		for _, header := range headers {
			fmt.Fprintf(b, "- **%s:** %s\n", header.name, header.value)
		}
	}
	if body != "" {
		b.WriteString("\n```\n")
		b.WriteString(body)
		b.WriteString("\n```\n")
	}
	b.WriteString("\n")
}

func writeUser(b *strings.Builder, event *IssueEventDetail) {
	if event == nil || len(event.User) == 0 {
		return
	}

	lines := make([]string, 0, 3)
	for _, key := range []string{"username", "email", "id"} {
		if value := mapString(event.User, key); value != "" {
			lines = append(lines, fmt.Sprintf("- **%s:** %s", key, value))
		}
	}
	if len(lines) == 0 {
		return
	}

	b.WriteString("## User\n\n")
	b.WriteString(strings.Join(lines, "\n"))
	b.WriteString("\n\n")
}

func writeTags(b *strings.Builder, issue map[string]any, event *IssueEventDetail) {
	tags := issueTags(issue)
	if event != nil {
		tags = mergeTags(tags, event.Tags)
	}
	if len(tags) == 0 {
		return
	}

	b.WriteString("## Tags\n\n")
	for _, tag := range tags {
		if tag.Key == "" || tag.Value == "" {
			continue
		}
		fmt.Fprintf(b, "- **%s:** %s\n", tag.Key, tag.Value)
	}
	b.WriteString("\n")
}

func writeContexts(b *strings.Builder, event *IssueEventDetail) {
	if event == nil || len(event.Contexts) == 0 {
		return
	}

	keys := make([]string, 0, len(event.Contexts))
	for key := range event.Contexts {
		if key == "replay" || key == "trace" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return
	}

	b.WriteString("## Contexts\n\n")
	for _, key := range keys {
		formatted := formatContextValue(event.Contexts[key])
		if formatted == "" {
			continue
		}
		fmt.Fprintf(b, "- **%s:** %s\n", key, formatted)
	}
	b.WriteString("\n")
}

func writeBreadcrumbs(b *strings.Builder, event *IssueEventDetail) {
	values := entryList(event, "breadcrumbs", "values")
	if len(values) == 0 {
		return
	}
	if len(values) > maxBreadcrumbs {
		values = values[len(values)-maxBreadcrumbs:]
	}

	b.WriteString("## Breadcrumbs\n\n")
	for _, raw := range values {
		crumb, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		level := firstNonEmpty(mapString(crumb, "level"), mapString(crumb, "type"))
		category := mapString(crumb, "category")
		message := firstNonEmpty(mapString(crumb, "message"), formatAny(crumb["data"]))
		parts := []string{}
		if level != "" {
			parts = append(parts, level)
		}
		if category != "" {
			parts = append(parts, category)
		}
		line := strings.Join(parts, " · ")
		if message != "" {
			if line != "" {
				line += ": "
			}
			line += message
		}
		if line == "" {
			continue
		}
		fmt.Fprintf(b, "- %s\n", line)
	}
	b.WriteString("\n")
}

func writeAdditionalData(b *strings.Builder, event *IssueEventDetail) {
	if event == nil {
		return
	}
	extra := event.Extra
	if len(extra) == 0 {
		extra = event.Context
	}
	if len(extra) == 0 {
		return
	}

	formatted := formatAny(extra)
	if formatted == "" {
		return
	}

	b.WriteString("## Additional Data\n\n")
	fmt.Fprintf(b, "```\n%s\n```\n\n", formatted)
}

func writeSDK(b *strings.Builder, event *IssueEventDetail) {
	if event == nil || len(event.SDK) == 0 {
		return
	}

	name := firstNonEmpty(mapString(event.SDK, "name"), mapString(event.SDK, "Name"))
	version := firstNonEmpty(mapString(event.SDK, "version"), mapString(event.SDK, "Version"))
	label := strings.TrimSpace(strings.Join([]string{name, version}, " "))
	if label == "" {
		return
	}

	b.WriteString("## SDK\n\n")
	fmt.Fprintf(b, "%s\n\n", label)
}

func writeReplayAndTrace(b *strings.Builder, event *IssueEventDetail, issue map[string]any) {
	replayID := contextField(event, "replay", "replay_id")
	if replayID == "" {
		replayID = tagValue(issue, event, "replayId")
	}
	traceID := contextField(event, "trace", "trace_id")

	if replayID == "" && traceID == "" {
		return
	}

	if replayID != "" {
		fmt.Fprintf(b, "**Replay ID:** %s\n", replayID)
	}
	if traceID != "" {
		fmt.Fprintf(b, "**Trace ID:** %s\n", traceID)
	}
	b.WriteString("\n")
}

type stackBlock struct {
	heading string
	frames  []string
}

func exceptionBlocks(event *IssueEventDetail) []stackBlock {
	frames := stackFrames(event)
	if len(frames) == 0 {
		return nil
	}

	var blocks []stackBlock
	for _, group := range frameGroups(event) {
		if len(group.frames) == 0 {
			continue
		}
		formatted := formatFrames(group.frames)
		if len(formatted) == 0 {
			continue
		}
		blocks = append(blocks, stackBlock{heading: group.heading, frames: formatted})
	}
	if len(blocks) > 0 {
		return blocks
	}

	formatted := formatFrames(frames)
	if len(formatted) == 0 {
		return nil
	}
	return []stackBlock{{frames: formatted}}
}

type frameGroup struct {
	heading string
	frames  []map[string]any
}

func frameGroups(event *IssueEventDetail) []frameGroup {
	var groups []frameGroup
	for _, entry := range eventEntries(event) {
		switch entry.Type {
		case "exception":
			for _, raw := range asList(entry.Data["values"]) {
				value, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				heading := strings.TrimSpace(strings.Join([]string{
					mapString(value, "type"),
					mapString(value, "value"),
				}, ": "))
				groups = append(groups, frameGroup{
					heading: heading,
					frames:  framesFrom(value["stacktrace"]),
				})
			}
		case "stacktrace":
			groups = append(groups, frameGroup{frames: framesFrom(entry.Data)})
		}
	}
	return groups
}

func stackFrames(event *IssueEventDetail) []map[string]any {
	var frames []map[string]any
	for _, group := range frameGroups(event) {
		frames = append(frames, group.frames...)
	}
	return frames
}

func framesFrom(stacktrace any) []map[string]any {
	trace, ok := stacktrace.(map[string]any)
	if !ok {
		return nil
	}
	var frames []map[string]any
	for _, raw := range asList(trace["frames"]) {
		frame, ok := raw.(map[string]any)
		if ok {
			frames = append(frames, frame)
		}
	}
	return frames
}

func formatFrames(frames []map[string]any) []string {
	if len(frames) > maxStackFrames {
		frames = frames[len(frames)-maxStackFrames:]
	}

	lines := make([]string, 0, len(frames))
	for i := len(frames) - 1; i >= 0; i-- {
		frame := frames[i]
		function := firstNonEmpty(mapString(frame, "function"), mapString(frame, "rawFunction"), "unknown")
		file := firstNonEmpty(
			mapString(frame, "filename"),
			mapString(frame, "absPath"),
			mapString(frame, "abs_path"),
			mapString(frame, "module"),
		)
		line := firstNonEmpty(formatAny(frame["lineNo"]), formatAny(frame["lineno"]), formatAny(frame["line_no"]))
		location := file
		if line != "" {
			if location != "" {
				location += ":"
			}
			location += line
		}
		if inApp(frame) {
			lines = append(lines, fmt.Sprintf("%s (%s) [in app]", function, location))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", function, location))
	}
	return lines
}

func inApp(frame map[string]any) bool {
	switch value := frame["inApp"].(type) {
	case bool:
		return value
	}
	switch value := frame["in_app"].(type) {
	case bool:
		return value
	}
	return false
}

func eventMessage(event *IssueEventDetail) string {
	if event == nil {
		return ""
	}
	if message := strings.TrimSpace(event.Message); message != "" {
		return message
	}
	data := entryData(event, "message")
	return firstNonEmpty(mapString(data, "formatted"), mapString(data, "message"))
}

func releaseLabel(issue map[string]any, event *IssueEventDetail) string {
	return firstNonEmpty(
		eventString(event, func(e *IssueEventDetail) string { return e.Release }),
		tagValue(issue, event, "release"),
	)
}

func nonzeroScalar(value any) string {
	text := strings.TrimSpace(formatScalar(value))
	if text == "" || text == "0" {
		return ""
	}
	return text
}

type requestHeader struct {
	name  string
	value string
}

var sensitiveHeaderParts = []string{
	"authorization",
	"cookie",
	"token",
	"secret",
	"password",
	"api-key",
	"apikey",
}

func isSensitiveHeader(name string) bool {
	lower := strings.ToLower(name)
	for _, part := range sensitiveHeaderParts {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}

func safeRequestHeaders(raw any) []requestHeader {
	headers := make([]requestHeader, 0)
	for _, header := range requestHeaders(raw) {
		if header.name == "" || header.value == "" || isSensitiveHeader(header.name) {
			continue
		}
		headers = append(headers, header)
	}
	return headers
}

func requestHeaders(raw any) []requestHeader {
	switch typed := raw.(type) {
	case []any:
		headers := make([]requestHeader, 0, len(typed))
		for _, item := range typed {
			if header, ok := headerFromValue(item); ok {
				headers = append(headers, header)
			}
		}
		return headers
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		headers := make([]requestHeader, 0, len(keys))
		for _, key := range keys {
			value := strings.TrimSpace(formatScalar(typed[key]))
			if key == "" || value == "" {
				continue
			}
			headers = append(headers, requestHeader{name: key, value: value})
		}
		return headers
	default:
		return nil
	}
}

func headerFromValue(value any) (requestHeader, bool) {
	switch typed := value.(type) {
	case []any:
		if len(typed) < 2 {
			return requestHeader{}, false
		}
		name := strings.TrimSpace(formatScalar(typed[0]))
		headerValue := strings.TrimSpace(formatScalar(typed[1]))
		if name == "" || headerValue == "" {
			return requestHeader{}, false
		}
		return requestHeader{name: name, value: headerValue}, true
	case map[string]any:
		name := firstNonEmpty(mapString(typed, "name"), mapString(typed, "key"))
		headerValue := mapString(typed, "value")
		if name == "" || headerValue == "" {
			return requestHeader{}, false
		}
		return requestHeader{name: name, value: headerValue}, true
	default:
		return requestHeader{}, false
	}
}

func eventEntries(event *IssueEventDetail) []IssueEventEntry {
	if event == nil {
		return nil
	}
	return event.Entries
}

func entryData(event *IssueEventDetail, entryType string) map[string]any {
	for _, entry := range eventEntries(event) {
		if entry.Type == entryType {
			return entry.Data
		}
	}
	return nil
}

func entryList(event *IssueEventDetail, entryType, key string) []any {
	data := entryData(event, entryType)
	if len(data) == 0 {
		return nil
	}
	return asList(data[key])
}

func asList(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func asStringMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case *Issue:
		if typed == nil {
			return nil
		}
		return structMap(*typed)
	case Issue:
		return structMap(typed)
	default:
		return nil
	}
}

func structMap(value any) map[string]any {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}
	return out
}

func mapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return strings.TrimSpace(formatScalar(values[key]))
}

func eventString(event *IssueEventDetail, read func(*IssueEventDetail) string) string {
	if event == nil {
		return ""
	}
	return strings.TrimSpace(read(event))
}

func projectLabel(issue map[string]any) string {
	project, ok := issue["project"].(map[string]any)
	if !ok {
		return ""
	}
	return firstNonEmpty(mapString(project, "name"), mapString(project, "slug"))
}

func assignedLabel(issue map[string]any) string {
	assigned, ok := issue["assignedTo"].(map[string]any)
	if !ok {
		return ""
	}
	return firstNonEmpty(mapString(assigned, "name"), mapString(assigned, "email"))
}

func issueTags(issue map[string]any) []IssueTag {
	if issue == nil {
		return nil
	}
	return coerceTags(issue["tags"])
}

func coerceTags(raw any) []IssueTag {
	switch typed := raw.(type) {
	case []IssueTag:
		return typed
	case []any:
		tags := make([]IssueTag, 0, len(typed))
		for _, item := range typed {
			tagMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := mapString(tagMap, "key")
			if key == "" {
				continue
			}
			tags = append(tags, IssueTag{Key: key, Value: mapString(tagMap, "value")})
		}
		return tags
	default:
		return nil
	}
}

func mergeTags(base []IssueTag, extra []IssueTag) []IssueTag {
	seen := map[string]int{}
	merged := append([]IssueTag{}, base...)
	for i, tag := range merged {
		seen[tag.Key] = i
	}
	for _, tag := range extra {
		if tag.Key == "" {
			continue
		}
		if i, ok := seen[tag.Key]; ok {
			if tag.Value != "" {
				merged[i].Value = tag.Value
			}
			continue
		}
		seen[tag.Key] = len(merged)
		merged = append(merged, tag)
	}
	return merged
}

func tagValue(issue map[string]any, event *IssueEventDetail, key string) string {
	for _, tag := range mergeTags(issueTags(issue), eventTags(event)) {
		if tag.Key == key {
			return tag.Value
		}
	}
	return ""
}

func eventTags(event *IssueEventDetail) []IssueTag {
	if event == nil {
		return nil
	}
	return event.Tags
}

func contextField(event *IssueEventDetail, contextName, field string) string {
	if event == nil {
		return ""
	}
	raw, ok := event.Contexts[contextName].(map[string]any)
	if !ok {
		return ""
	}
	return mapString(raw, field)
}

func formatContextValue(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		parts := make([]string, 0, 4)
		for _, key := range []string{"name", "version", "runtime"} {
			if item := mapString(typed, key); item != "" && !containsString(parts, item) {
				parts = append(parts, item)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, " ")
		}
		return formatAny(typed)
	default:
		return formatScalar(value)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func formatScalar(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case json.Number:
		return typed.String()
	case int:
		return fmt.Sprintf("%d", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return fmt.Sprintf("%g", typed)
	case bool:
		return fmt.Sprintf("%t", typed)
	default:
		return ""
	}
}

func formatAny(value any) string {
	if value == nil {
		return ""
	}
	if scalar := formatScalar(value); scalar != "" {
		return scalar
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	text := strings.TrimSpace(string(encoded))
	if text == "null" || text == "{}" || text == "[]" {
		return ""
	}
	if len(text) > 2000 {
		return text[:2000] + "…"
	}
	return text
}

func issueID(issue any) string {
	issueMap := asStringMap(issue)
	if issueMap == nil {
		return ""
	}
	return mapString(issueMap, "id")
}

type warningLogger interface {
	Warnf(format string, args ...any)
}

// FetchedIssueDescription loads the issue and preferred event, then renders
// IssueDescription. API failures keep the webhook issue and omit the event.
func FetchedIssueDescription(client *Client, issue any, logger warningLogger) string {
	if client == nil {
		return IssueDescription(issue, nil)
	}

	id := issueID(issue)
	if id == "" {
		return IssueDescription(issue, nil)
	}

	described := issue
	fetched, err := client.GetIssue(id)
	if err != nil {
		warnf(logger, "failed to retrieve sentry issue %s: %v", id, err)
	} else if fetched != nil {
		described = fetched
	}

	event, err := client.GetPreferredIssueEvent(id)
	if err != nil {
		warnf(logger, "failed to retrieve sentry issue event %s: %v", id, err)
	}

	return IssueDescription(described, event)
}

func warnf(logger warningLogger, format string, args ...any) {
	if logger == nil {
		return
	}
	logger.Warnf(format, args...)
}
