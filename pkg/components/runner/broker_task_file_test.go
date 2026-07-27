package runner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrokerCreateTaskRequestIncludesFiles(t *testing.T) {
	t.Parallel()

	req := brokerCreateTaskRequest{
		FleetID:    "e1-large-amd64",
		Commands:   []BrokerCommand{{Command: `cat "$SUPERPLANE_TASK_DIR/hi.txt"`}},
		Files:      []BrokerTaskFile{{Path: "hi.txt", Content: "hello", Mode: "0644"}},
		WebhookURL: "https://example/hook",
	}
	b, err := json.Marshal(req)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	files, ok := got["files"].([]any)
	require.True(t, ok)
	require.Len(t, files, 1)
	file := files[0].(map[string]any)
	assert.Equal(t, "hi.txt", file["path"])
	assert.Equal(t, "hello", file["content"])
	assert.Equal(t, "0644", file["mode"])
}
