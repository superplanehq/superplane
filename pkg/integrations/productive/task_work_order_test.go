package productive

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskURL(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "https://app.productive.io/12345/tasks/91", TaskURL("12345", "91"))
	assert.Equal(t, "", TaskURL("", "91"))
	assert.Equal(t, "", TaskURL("12345", ""))
}

func TestTaskURLFragment(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/tasks/91", TaskURLFragment("91"))
	assert.Equal(t, "", TaskURLFragment("  "))
}

func TestTaskIDFromURL(t *testing.T) {
	t.Parallel()

	taskID, ok := TaskIDFromURL("https://app.productive.io/12345/tasks/91")
	require.True(t, ok)
	assert.Equal(t, "91", taskID)

	taskID, ok = TaskIDFromURL("https://app.productive.io/12345-name/tasks/91/")
	require.True(t, ok)
	assert.Equal(t, "91", taskID)

	_, ok = TaskIDFromURL("https://app.productive.io/12345/projects/91")
	assert.False(t, ok)

	_, ok = TaskIDFromURL("https://example.com/12345/tasks/91")
	assert.False(t, ok)

	nine, ok := TaskIDFromURL("https://app.productive.io/12345/tasks/9")
	require.True(t, ok)
	ninetyOne, ok := TaskIDFromURL("https://app.productive.io/12345/tasks/91")
	require.True(t, ok)
	assert.NotEqual(t, nine, ninetyOne)
}

func TestTaskEnvelope_StampsTaskURL(t *testing.T) {
	t.Parallel()

	envelope := TaskEnvelope(TaskCreatedEvent, map[string]any{"id": "91", "type": "tasks"}, "12345")
	assert.Equal(t, "https://app.productive.io/12345/tasks/91", envelope["url"])
	assert.Equal(t, map[string]any{"event": TaskCreatedEvent}, envelope["meta"])

	withoutOrg := TaskEnvelope(TaskCreatedEvent, map[string]any{"id": "91"}, "")
	_, hasURL := withoutOrg["url"]
	assert.False(t, hasURL)
}
