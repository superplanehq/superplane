package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedJSONDecodesValueOnce(t *testing.T) {
	example := NewEmbeddedJSON([]byte(`{"status":"ready"}`))

	first := example.Value()
	first["status"] = "changed"

	require.Equal(t, map[string]any{"status": "changed"}, example.Value())
}

func TestEmbeddedJSONReturnsEmptyMapForInvalidJSON(t *testing.T) {
	example := NewEmbeddedJSON([]byte(`invalid`))

	require.Empty(t, example.Value())
}
