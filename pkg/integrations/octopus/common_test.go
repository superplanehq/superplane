package octopus

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Octopus__verifyWebhookHeader(t *testing.T) {
	t.Run("valid secret -> no error", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(webhookHeaderKey, "my-secret")

		err := verifyWebhookHeader(core.WebhookRequestContext{
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "my-secret"},
		})

		require.NoError(t, err)
	})

	t.Run("missing webhook context -> error", func(t *testing.T) {
		err := verifyWebhookHeader(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.ErrorContains(t, err, "missing webhook context")
	})

	t.Run("missing header -> error", func(t *testing.T) {
		err := verifyWebhookHeader(core.WebhookRequestContext{
			Headers: http.Header{},
			Webhook: &contexts.WebhookContext{Secret: "my-secret"},
		})

		assert.ErrorContains(t, err, "missing X-SuperPlane-Webhook-Secret header")
	})

	t.Run("wrong secret -> error", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(webhookHeaderKey, "wrong-secret")

		err := verifyWebhookHeader(core.WebhookRequestContext{
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "my-secret"},
		})

		assert.ErrorContains(t, err, "invalid webhook secret")
	})

	t.Run("empty webhook secret -> error", func(t *testing.T) {
		headers := http.Header{}
		headers.Set(webhookHeaderKey, "some-value")

		err := verifyWebhookHeader(core.WebhookRequestContext{
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: ""},
		})

		assert.ErrorContains(t, err, "missing webhook secret")
	})
}

func Test__Octopus__normalizeEventCategories(t *testing.T) {
	t.Run("removes duplicates and sorts", func(t *testing.T) {
		result := normalizeEventCategories([]string{
			"DeploymentFailed",
			"DeploymentSucceeded",
			"DeploymentFailed",
		})

		assert.Equal(t, []string{"DeploymentFailed", "DeploymentSucceeded"}, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := normalizeEventCategories([]string{" DeploymentQueued ", "  DeploymentStarted"})
		assert.Equal(t, []string{"DeploymentQueued", "DeploymentStarted"}, result)
	})

	t.Run("removes empty strings", func(t *testing.T) {
		result := normalizeEventCategories([]string{"DeploymentQueued", "", "  ", "DeploymentFailed"})
		assert.Equal(t, []string{"DeploymentFailed", "DeploymentQueued"}, result)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		result := normalizeEventCategories([]string{})
		assert.Empty(t, result)
	})
}

func Test__Octopus__filterAllowedEventCategories(t *testing.T) {
	allowed := []string{"DeploymentSucceeded", "DeploymentFailed"}

	t.Run("filters to allowed only", func(t *testing.T) {
		result := filterAllowedEventCategories(
			[]string{"DeploymentSucceeded", "SomethingElse", "DeploymentFailed"},
			allowed,
		)

		assert.Equal(t, []string{"DeploymentSucceeded", "DeploymentFailed"}, result)
	})

	t.Run("removes duplicates", func(t *testing.T) {
		result := filterAllowedEventCategories(
			[]string{"DeploymentSucceeded", "DeploymentSucceeded"},
			allowed,
		)

		assert.Equal(t, []string{"DeploymentSucceeded"}, result)
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		result := filterAllowedEventCategories(
			[]string{"SomethingUnknown"},
			allowed,
		)

		assert.Empty(t, result)
	})
}

func Test__Octopus__readString(t *testing.T) {
	assert.Equal(t, "hello", readString("hello"))
	assert.Equal(t, "", readString(nil))
	assert.Equal(t, "", readString(123))
	assert.Equal(t, "", readString(map[string]any{}))
}

func Test__Octopus__readMap(t *testing.T) {
	m := map[string]any{"key": "value"}
	assert.Equal(t, m, readMap(m))
	assert.Equal(t, map[string]any{}, readMap(nil))
	assert.Equal(t, map[string]any{}, readMap("not a map"))
}

func Test__Octopus__isTaskCompleted(t *testing.T) {
	assert.True(t, isTaskCompleted(TaskStateSuccess))
	assert.True(t, isTaskCompleted(TaskStateFailed))
	assert.True(t, isTaskCompleted(TaskStateCanceled))
	assert.True(t, isTaskCompleted(TaskStateTimedOut))
	assert.False(t, isTaskCompleted(TaskStateQueued))
	assert.False(t, isTaskCompleted(TaskStateExecuting))
	assert.False(t, isTaskCompleted(TaskStateCancelling))
}

func Test__Octopus__isTaskSuccessful(t *testing.T) {
	assert.True(t, isTaskSuccessful(TaskStateSuccess))
	assert.False(t, isTaskSuccessful(TaskStateFailed))
	assert.False(t, isTaskSuccessful(TaskStateCanceled))
	assert.False(t, isTaskSuccessful(TaskStateTimedOut))
}

func Test__Octopus__isResolvedValue(t *testing.T) {
	assert.True(t, isResolvedValue("Projects-1"))
	assert.True(t, isResolvedValue("Environments-2"))
	assert.False(t, isResolvedValue(""))
	assert.False(t, isResolvedValue("{{ .project }}"))
	assert.False(t, isResolvedValue("prefix-{{expr}}"))
}

func Test__Octopus__payloadType(t *testing.T) {
	assert.Equal(t, "octopus.deployment.queued", payloadType(EventCategoryDeploymentQueued))
	assert.Equal(t, "octopus.deployment.started", payloadType(EventCategoryDeploymentStarted))
	assert.Equal(t, "octopus.deployment.succeeded", payloadType(EventCategoryDeploymentSucceeded))
	assert.Equal(t, "octopus.deployment.failed", payloadType(EventCategoryDeploymentFailed))
	assert.Equal(t, "octopus.deployment.unknown", payloadType("Unknown"))
}

func Test__Octopus__webhookRequestIsJSON(t *testing.T) {
	t.Run("application/json -> true", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		assert.True(t, webhookRequestIsJSON(core.WebhookRequestContext{Headers: headers}))
	})

	t.Run("application/json; charset=utf-8 -> true", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "application/json; charset=utf-8")
		assert.True(t, webhookRequestIsJSON(core.WebhookRequestContext{Headers: headers}))
	})

	t.Run("text/plain -> false", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Content-Type", "text/plain")
		assert.False(t, webhookRequestIsJSON(core.WebhookRequestContext{Headers: headers}))
	})

	t.Run("empty content type -> false", func(t *testing.T) {
		assert.False(t, webhookRequestIsJSON(core.WebhookRequestContext{Headers: http.Header{}}))
	})
}

func Test__Octopus__readRelatedDocumentIDs(t *testing.T) {
	t.Run("parses related document IDs", func(t *testing.T) {
		event := map[string]any{
			"RelatedDocumentIds": []any{
				"Projects-1",
				"Environments-2",
				"Deployments-3",
				"Releases-4",
			},
		}

		result := readRelatedDocumentIDs(event)
		assert.Equal(t, []string{"Projects-1"}, result["Projects"])
		assert.Equal(t, []string{"Environments-2"}, result["Environments"])
		assert.Equal(t, []string{"Deployments-3"}, result["Deployments"])
		assert.Equal(t, []string{"Releases-4"}, result["Releases"])
	})

	t.Run("missing RelatedDocumentIds -> empty map", func(t *testing.T) {
		result := readRelatedDocumentIDs(map[string]any{})
		assert.Empty(t, result)
	})

	t.Run("non-array RelatedDocumentIds -> empty map", func(t *testing.T) {
		result := readRelatedDocumentIDs(map[string]any{
			"RelatedDocumentIds": "not-an-array",
		})
		assert.Empty(t, result)
	})

	t.Run("skips non-string entries", func(t *testing.T) {
		result := readRelatedDocumentIDs(map[string]any{
			"RelatedDocumentIds": []any{
				"Projects-1",
				123,
				"Environments-2",
			},
		})
		assert.Equal(t, []string{"Projects-1"}, result["Projects"])
		assert.Equal(t, []string{"Environments-2"}, result["Environments"])
	})

	t.Run("skips entries without dash separator", func(t *testing.T) {
		result := readRelatedDocumentIDs(map[string]any{
			"RelatedDocumentIds": []any{
				"Projects-1",
				"InvalidFormat",
			},
		})
		assert.Len(t, result, 1)
		assert.Equal(t, []string{"Projects-1"}, result["Projects"])
	})
}

func Test__Octopus__containsRelatedDocument(t *testing.T) {
	docs := map[string][]string{
		"Projects":     {"Projects-1", "Projects-2"},
		"Environments": {"Environments-5"},
	}

	assert.True(t, containsRelatedDocument(docs, "Projects", "Projects-1"))
	assert.True(t, containsRelatedDocument(docs, "Projects", "Projects-2"))
	assert.False(t, containsRelatedDocument(docs, "Projects", "Projects-99"))
	assert.False(t, containsRelatedDocument(docs, "Deployments", "Deployments-1"))
}

func Test__Octopus__mergeStringSlices(t *testing.T) {
	t.Run("merges and deduplicates", func(t *testing.T) {
		result := mergeStringSlices(
			[]string{"a", "b"},
			[]string{"b", "c"},
		)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("preserves order", func(t *testing.T) {
		result := mergeStringSlices(
			[]string{"c", "a"},
			[]string{"b"},
		)
		assert.Equal(t, []string{"c", "a", "b"}, result)
	})

	t.Run("empty slices", func(t *testing.T) {
		result := mergeStringSlices([]string{}, []string{})
		assert.Empty(t, result)
	})
}
