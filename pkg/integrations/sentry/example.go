package sentry

func (t *OnIssue) ExampleData() map[string]any {
	return map[string]any{
		"resource": "issue",
		"action":   "created",
		"installation": map[string]any{
			"uuid": "7a485448-a9e2-4c85-8a3c-4f44175783c9",
		},
		"actor": map[string]any{
			"type": "user",
			"id":   "789",
			"name": "Person",
		},
		"data": map[string]any{
			"issue": sentryIssueExample(),
		},
		"timestamp": "2022-04-04T18:17:18.320000Z",
	}
}

func sentryIssueExample() map[string]any {
	return map[string]any{
		"id":        "123",
		"shortId":   "IPE-1",
		"title":     "Error #1: This is a test error!",
		"culprit":   "SentryCustomError(frontend/src/util)",
		"level":     "error",
		"status":    "unresolved",
		"firstSeen": "2022-04-04T18:17:18.320000Z",
		"lastSeen":  "2022-04-04T18:17:18.320000Z",
		"project": map[string]any{
			"id":   "456",
			"name": "ipe",
			"slug": "ipe",
		},
		"assignedTo": map[string]any{
			"type": "user",
			"id":   "789",
			"name": "Person",
		},
	}
}
