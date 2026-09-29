package datadog

import "strings"

// ErrorTrackingIssuePayload shapes one Error Tracking issue like the alert
// the intake trigger emits, so a seeded issue and a received alert take the
// same path through the graph.
func ErrorTrackingIssuePayload(issue ErrorTrackingIssue) ErrorTrackingAlertPayload {
	service := strings.TrimSpace(issue.Service)
	env := ""
	if issue.Sample != nil {
		env = strings.ToLower(strings.TrimSpace(issue.Sample.Env))
	}

	link := strings.TrimSpace(issue.URL)
	text := DescribeErrorTrackingIssue(issue, AlertDetails{Link: link})
	tags := errorTrackingIssueTags(service, env)

	return ErrorTrackingAlertPayload{
		ID:              strings.TrimSpace(issue.ID),
		EventType:       ErrorTrackingAlertEventType,
		Title:           limitRunes(issue.IssueTitle(), maxAlertTitleRunes),
		Body:            text,
		AlertTransition: AlertTransitionTriggered,
		Tags:            tags,
		AlertQuery:      errorTrackingServiceQuery(service),
		AlertScope:      tags,
		Link:            link,
		Description:     text,
		Environment:     env,
	}
}

func errorTrackingServiceQuery(service string) string {
	if service == "" {
		return ""
	}
	return "service:" + service
}

func errorTrackingIssueTags(service, env string) string {
	tags := make([]string, 0, 2)
	if service != "" {
		tags = append(tags, "service:"+service)
	}
	if env != "" {
		tags = append(tags, "env:"+env)
	}
	return strings.Join(tags, ",")
}
