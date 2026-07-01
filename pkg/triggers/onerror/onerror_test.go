package onerror

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnError_Metadata(t *testing.T) {
	tr := &OnError{}

	assert.Equal(t, "onError", tr.Name())
	assert.NotEmpty(t, tr.Label())
	assert.NotEmpty(t, tr.Description())
	assert.NotEmpty(t, tr.Icon())
}

func TestOnError_HasNoHooksOrConfiguration(t *testing.T) {
	tr := &OnError{}

	assert.Empty(t, tr.Hooks(), "On Error must not expose user-invokable hooks")
	assert.Empty(t, tr.Configuration(), "On Error must not require configuration")
}

func TestOnError_ExampleDataMatchesPayloadShape(t *testing.T) {
	tr := &OnError{}

	example := tr.ExampleData()
	require.NotNil(t, example)
	assert.Equal(t, PayloadType, example["type"])

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, data, "node")
	assert.Contains(t, data, "error")
	assert.Contains(t, data, "run")
	assert.Contains(t, data, "root")
	assert.Contains(t, data, "payloads")
}
