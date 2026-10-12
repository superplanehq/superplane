package linear

// IssueEventPayload builds the Linear webhook body for one issue, so an
// imported issue uses the same shape as a live linear.issue event.
func IssueEventPayload(issue Issue) map[string]any {
	labels := make([]any, 0, len(issue.Labels))
	for _, label := range issue.Labels {
		labels = append(labels, map[string]any{
			"id":   label.ID,
			"name": label.Name,
		})
	}

	projectID := ""
	if issue.Project != nil {
		projectID = issue.Project.ID
	}

	state := map[string]any{}
	if issue.State != nil {
		state = map[string]any{
			"id":   issue.State.ID,
			"name": issue.State.Name,
			"type": issue.State.Type,
		}
	}

	team := map[string]any{}
	teamID := ""
	if issue.Team != nil {
		teamID = issue.Team.ID
		team = map[string]any{
			"id":   issue.Team.ID,
			"key":  issue.Team.Key,
			"name": issue.Team.Name,
		}
	}

	return map[string]any{
		"action": "create",
		"type":   IssueResourceType,
		"url":    issue.URL,
		"data": map[string]any{
			"id":          issue.ID,
			"identifier":  issue.Identifier,
			"title":       issue.Title,
			"description": issue.Description,
			"projectId":   projectID,
			"teamId":      teamID,
			"state":       state,
			"team":        team,
			"labels":      labels,
		},
	}
}
