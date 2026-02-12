package linear

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIssue__Configuration(t *testing.T) {
	component := &CreateIssue{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "linear.createIssue", component.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "Create Issue", component.Label())
	})

	t.Run("returns configuration fields", func(t *testing.T) {
		fields := component.Configuration()
		require.NotEmpty(t, fields)

		fieldNames := make(map[string]bool)
		for _, f := range fields {
			fieldNames[f.Name] = true
		}

		assert.True(t, fieldNames["teamId"])
		assert.True(t, fieldNames["title"])
		assert.True(t, fieldNames["description"])
	})
}

func Test__CreateIssue__Execute__Validation(t *testing.T) {
	component := &CreateIssue{}

	t.Run("returns error when teamId is missing", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Teams: []Team{{ID: "team1", Name: "Engineering", Key: "ENG"}},
			},
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"title": "Test Issue",
			},
		}

		_, err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "teamId is required")
	})

	t.Run("returns error when title is missing", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Teams: []Team{{ID: "team1", Name: "Engineering", Key: "ENG"}},
			},
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		ctx := core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"teamId": "team1",
			},
		}

		_, err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
}

func Test__OnIssueCreated__Configuration(t *testing.T) {
	trigger := &OnIssueCreated{}

	t.Run("has correct name", func(t *testing.T) {
		assert.Equal(t, "linear.onIssueCreated", trigger.Name())
	})

	t.Run("has correct label", func(t *testing.T) {
		assert.Equal(t, "On Issue Created", trigger.Label())
	})
}

func Test__OnIssueCreated__Match(t *testing.T) {
	trigger := &OnIssueCreated{}

	t.Run("matches create action with Issue type", func(t *testing.T) {
		ctx := core.TriggerContext{
			Metadata: map[string]interface{}{
				"webhook": map[string]interface{}{
					"action": "create",
					"type":   "Issue",
					"data": map[string]interface{}{
						"id":         "issue1",
						"identifier": "ENG-123",
						"title":      "Test Issue",
						"teamId":     "team1",
					},
				},
			},
			Configuration: map[string]interface{}{},
		}

		matched, output, err := trigger.Match(ctx)
		require.NoError(t, err)
		assert.True(t, matched)
		assert.Equal(t, "issue1", output["id"])
		assert.Equal(t, "ENG-123", output["identifier"])
	})

	t.Run("does not match update action", func(t *testing.T) {
		ctx := core.TriggerContext{
			Metadata: map[string]interface{}{
				"webhook": map[string]interface{}{
					"action": "update",
					"type":   "Issue",
					"data": map[string]interface{}{
						"id": "issue1",
					},
				},
			},
			Configuration: map[string]interface{}{},
		}

		matched, _, err := trigger.Match(ctx)
		require.NoError(t, err)
		assert.False(t, matched)
	})

	t.Run("filters by team when specified", func(t *testing.T) {
		ctx := core.TriggerContext{
			Metadata: map[string]interface{}{
				"webhook": map[string]interface{}{
					"action": "create",
					"type":   "Issue",
					"data": map[string]interface{}{
						"id":     "issue1",
						"teamId": "team2",
					},
				},
			},
			Configuration: map[string]interface{}{
				"teamId": "team1",
			},
		}

		matched, _, err := trigger.Match(ctx)
		require.NoError(t, err)
		assert.False(t, matched)
	})
}
