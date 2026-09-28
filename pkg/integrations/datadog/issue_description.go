package datadog

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxIssueDescriptionRunes = 19000
	maxAlertMessageRunes     = 1600
)

// AlertDetails is the monitor text that arrived with an Error Tracking alert.
type AlertDetails struct {
	Title        string
	Body         string
	EventMessage string
	Link         string
	Tags         string
	AlertScope   string
	AlertQuery   string
	AggregKey    string
}

// DescribeErrorTrackingIssue renders the issue and the alert as task markdown.
// Empty sections are omitted. The result stays within the work order limit.
func DescribeErrorTrackingIssue(issue ErrorTrackingIssue, alert AlertDetails) string {
	var b strings.Builder
	writeIssueLink(&b, issue)
	writeErrorSection(&b, issue)
	writeLocationSection(&b, issue)
	writeErrorSampleSection(&b, issue)
	writeStackSection(&b, issue.Stack)
	writeRelatedLogsSection(&b, issue)
	writeImpactSection(&b, issue)
	writeOwnershipSection(&b, issue)
	writeAlertSection(&b, issue, alert)

	text := strings.TrimSpace(b.String())
	if text == "" {
		text = strings.TrimSpace(alert.Body)
	}
	return limitRunes(text, maxIssueDescriptionRunes)
}

func writeIssueLink(b *strings.Builder, issue ErrorTrackingIssue) {
	if strings.TrimSpace(issue.URL) == "" {
		return
	}
	b.WriteString("[View in Datadog](")
	b.WriteString(strings.TrimSpace(issue.URL))
	b.WriteString(")")
}

func writeErrorSection(b *strings.Builder, issue ErrorTrackingIssue) {
	lines := []string{}
	if errorType := strings.TrimSpace(issue.ErrorType); errorType != "" {
		lines = append(lines, "- **Type:** "+markdownInline(errorType))
	}
	if issue.IsCrash {
		lines = append(lines, "- **Crash:** yes")
	}
	message := strings.TrimSpace(issue.ErrorMessage)
	if message != "" {
		if strings.Contains(message, "\n") {
			lines = append(lines, "", fencedBlock(message))
		} else {
			lines = append(lines, "- **Message:** "+markdownInline(message))
		}
	}
	writeSection(b, "Error", strings.Join(lines, "\n"))
}

func writeLocationSection(b *strings.Builder, issue ErrorTrackingIssue) {
	lines := []string{}
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, "- **"+label+":** "+markdownInline(value))
	}
	if path := strings.TrimSpace(issue.FilePath); path != "" {
		lines = append(lines, "- **File:** `"+strings.ReplaceAll(path, "`", "'")+"`")
	}
	if name := strings.TrimSpace(issue.FunctionName); name != "" {
		lines = append(lines, "- **Function:** `"+strings.ReplaceAll(name, "`", "'")+"`")
	}
	add("Service", issue.Service)
	if !strings.EqualFold(strings.TrimSpace(issue.Platform), "UNKNOWN") {
		add("Platform", displayToken(issue.Platform))
	}
	if len(issue.Languages) > 0 {
		names := make([]string, 0, len(issue.Languages))
		for _, language := range issue.Languages {
			names = append(names, displayToken(language))
		}
		add("Languages", strings.Join(names, ", "))
	}
	writeSection(b, "Location", strings.Join(lines, "\n"))
}

func writeErrorSampleSection(b *strings.Builder, issue ErrorTrackingIssue) {
	sample := issue.Sample
	if sample == nil {
		return
	}

	lines := []string{}
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, "- **"+label+":** "+markdownInline(value))
	}
	add("Time", formatTime(sample.Timestamp))
	add("Origin", sample.Origin)
	add("Env", sample.Env)
	add("Version", sample.Version)
	add("Host", sample.Host)
	add("Resource", sample.Resource)
	add("Route", sample.Route)
	add("Action", sample.Action)
	add("Request", sampleRequest(*sample))
	add("User", sample.UserID)
	add("Request ID", sample.RequestID)
	add("Trace ID", sample.TraceID)
	add("Fingerprint", sample.Fingerprint)
	if len(sample.Breadcrumbs) > 0 {
		lines = append(lines, "- **Breadcrumbs:**")
		for i, step := range sample.Breadcrumbs {
			lines = append(lines, "  "+strconv.Itoa(i+1)+". "+markdownInline(step))
		}
	}
	if link := strings.TrimSpace(sample.TraceURL); link != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "[View the trace]("+link+")")
	}
	writeSection(b, "Error sample", strings.Join(lines, "\n"))
}

func sampleRequest(sample ErrorSample) string {
	method := strings.TrimSpace(sample.HTTPMethod)
	path := strings.TrimSpace(sample.HTTPPath)
	status := strings.TrimSpace(sample.HTTPStatus)
	request := strings.TrimSpace(method + " " + path)
	if request != "" && status != "" {
		return request + " -> " + status
	}
	return firstNonEmpty(request, status)
}

func writeRelatedLogsSection(b *strings.Builder, issue ErrorTrackingIssue) {
	if len(issue.RelatedLogs) == 0 {
		return
	}

	lines := make([]string, 0, len(issue.RelatedLogs))
	for _, line := range issue.RelatedLogs {
		formatted := formatLogLine(line)
		if formatted == "" {
			continue
		}
		lines = append(lines, formatted)
	}
	if len(lines) == 0 {
		return
	}

	body := fencedBlock(strings.Join(lines, "\n"))
	if link := relatedLogsLink(issue); link != "" {
		body += "\n\n[View logs in Datadog](" + link + ")"
	}
	writeSection(b, "Related logs", body)
}

func relatedLogsLink(issue ErrorTrackingIssue) string {
	if issue.Sample == nil {
		return ""
	}
	return strings.TrimSpace(issue.Sample.LogsURL)
}

func formatLogLine(line LogLine) string {
	parts := make([]string, 0, 4)
	if !line.Timestamp.IsZero() {
		parts = append(parts, line.Timestamp.UTC().Format("15:04:05.000"))
	}
	if status := strings.ToUpper(strings.TrimSpace(line.Status)); status != "" {
		parts = append(parts, status)
	}
	if service := strings.TrimSpace(line.Service); service != "" {
		parts = append(parts, service)
	}
	if message := strings.TrimSpace(line.Message); message != "" {
		parts = append(parts, message)
	}
	return strings.Join(parts, " ")
}

func writeStackSection(b *strings.Builder, stack string) {
	stack = strings.TrimSpace(stack)
	if stack == "" {
		return
	}
	writeSection(b, "Stack trace", fencedBlock(stack))
}

func writeImpactSection(b *strings.Builder, issue ErrorTrackingIssue) {
	lines := []string{}
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, "- **"+label+":** "+markdownInline(value))
	}
	add("State", displayToken(issue.State))
	if issue.HasActivity {
		add("Occurrences", formatCount(issue.TotalCount))
		add("Impacted users", formatCount(issue.ImpactedUsers))
		add("Impacted sessions", formatCount(issue.ImpactedSessions))
	}
	add("First seen", formatTime(issue.FirstSeen))
	add("Last seen", formatTime(issue.LastSeen))
	add("First version", issue.FirstSeenVersion)
	add("Last version", issue.LastSeenVersion)
	if issue.Regression != nil {
		add("Regressed at", formatTime(issue.Regression.RegressedAt))
		add("Regressed version", issue.Regression.RegressedAtVersion)
		add("Previously resolved", formatTime(issue.Regression.ResolvedAt))
	}
	if id := strings.TrimSpace(issue.ID); id != "" {
		add("Issue ID", id)
	}
	writeSection(b, "Impact", strings.Join(lines, "\n"))
}

func writeOwnershipSection(b *strings.Builder, issue ErrorTrackingIssue) {
	lines := []string{}
	if assignee := strings.TrimSpace(issue.Assignee); assignee != "" {
		lines = append(lines, "- **Assignee:** "+markdownInline(assignee))
	}
	if len(issue.Teams) > 0 {
		lines = append(lines, "- **Teams:** "+markdownInline(strings.Join(issue.Teams, ", ")))
	}
	if label := caseLabel(issue); label != "" {
		lines = append(lines, "- **Case:** "+label)
	}
	writeSection(b, "Ownership", strings.Join(lines, "\n"))
}

func caseLabel(issue ErrorTrackingIssue) string {
	label := strings.TrimSpace(issue.CaseKey)
	title := strings.TrimSpace(issue.CaseTitle)
	switch {
	case label != "" && title != "":
		label = label + " " + title
	case title != "":
		label = title
	}
	if label == "" {
		return ""
	}
	if url := strings.TrimSpace(issue.CaseURL); url != "" {
		return "[" + markdownInline(label) + "](" + url + ")"
	}
	return markdownInline(label)
}

func writeAlertSection(b *strings.Builder, issue ErrorTrackingIssue, alert AlertDetails) {
	lines := []string{}
	add := func(label, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		lines = append(lines, "- **"+label+":** "+markdownInline(value))
	}

	monitorTitle := strings.TrimSpace(alert.Title)
	if monitorTitle != "" && !strings.EqualFold(monitorTitle, issue.IssueTitle()) {
		add("Monitor", monitorTitle)
	}
	add("Tags", alert.Tags)
	add("Scope", alert.AlertScope)
	add("Query", alert.AlertQuery)
	if message := alertMessage(alert, issue.ErrorMessage); message != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, message)
	}
	if link := strings.TrimSpace(alert.Link); link != "" && link != strings.TrimSpace(issue.URL) {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "[Open the monitor]("+link+")")
	}
	writeSection(b, "Alert", strings.Join(lines, "\n"))
}

func alertMessage(alert AlertDetails, errorMessage string) string {
	message := strings.TrimSpace(alert.EventMessage)
	body := strings.TrimSpace(alert.Body)
	if len(message) < len(body) {
		message = body
	}
	if message == "" || strings.EqualFold(message, strings.TrimSpace(errorMessage)) {
		return ""
	}
	return limitRunes(message, maxAlertMessageRunes)
}

func fencedBlock(text string) string {
	text = strings.TrimRight(text, "\n")
	fence := "```"
	for strings.Contains(text, fence) {
		fence += "`"
	}
	return fence + "\n" + text + "\n" + fence
}

func writeSection(b *strings.Builder, title, body string) {
	body = strings.TrimSpace(body)
	if body == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("## ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(body)
}

func displayToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Fields(strings.ReplaceAll(strings.ToLower(value), "_", " "))
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func formatCount(value int64) string {
	return strconv.FormatInt(value, 10)
}

func markdownInline(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	return value
}

func limitRunes(text string, max int) string {
	text = strings.TrimSpace(text)
	if max <= 0 || utf8.RuneCountInString(text) <= max {
		return text
	}
	if max <= 3 {
		return "..."
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:max-3])) + "..."
}
