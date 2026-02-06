package circleci

func (c *TriggerPipeline) ExampleOutput() map[string]any {
	return map[string]any{
		"pipeline": map[string]any{
			"id":         "1285fe1d-d3a6-44fc-8886-8979558254c4",
			"number":     130,
			"created_at": "2021-09-01T22:49:03.544Z",
		},
		"workflows": []map[string]any{
			{
				"id":     "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
				"name":   "build-test-deploy",
				"status": "success",
			},
		},
	}
}

func (t *OnPipelineCompleted) ExampleData() map[string]any {
	return map[string]any{
		"type":        "workflow-completed",
		"id":          "3888f21b-eaa7-38e3-8f3d-75a63bba8895",
		"happened_at": "2021-09-01T22:49:34.317Z",
		"workflow": map[string]any{
			"id":         "fda08377-fe7e-46b1-8992-3a7aaecac9c3",
			"name":       "build-test-deploy",
			"status":     "success",
			"created_at": "2021-09-01T22:49:03.616Z",
			"stopped_at": "2021-09-01T22:49:34.170Z",
			"url":        "https://app.circleci.com/pipelines/github/username/repo/130/workflows/fda08377-fe7e-46b1-8992-3a7aaecac9c3",
		},
		"pipeline": map[string]any{
			"id":         "1285fe1d-d3a6-44fc-8886-8979558254c4",
			"number":     130,
			"created_at": "2021-09-01T22:49:03.544Z",
		},
		"project": map[string]any{
			"id":   "84996744-a854-4f5e-aea3-04e2851dc1d2",
			"name": "repo",
			"slug": "github/username/repo",
		},
	}
}
