package manualrun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinPath(t *testing.T) {
	assert.Equal(t, "body", joinPath("", "body"))
	assert.Equal(t, "body.name", joinPath("body", "name"))
	assert.Equal(t, "items[0]", joinPath("items", "[0]"))
	assert.Equal(t, "items[0].id", joinPath("items[0]", "id"))
}

func TestWalkPayload_nestedMap(t *testing.T) {
	payload := map[string]any{
		"body": map[string]any{
			"name": "Alice",
			"size": "large",
		},
	}
	var paths []string
	WalkPayload(payload, "", func(path string, value any) WalkControl {
		paths = append(paths, path)
		return WalkContinue
	})
	assert.ElementsMatch(t, []string{"body.name", "body.size"}, paths)
}

func TestWalkPayload_mapString(t *testing.T) {
	payload := map[string]string{
		"message": "hello",
	}
	var paths []string
	WalkPayload(payload, "", func(path string, value any) WalkControl {
		paths = append(paths, path)
		return WalkContinue
	})
	assert.Equal(t, []string{"message"}, paths)
}

func TestWalkPayload_arrayPaths(t *testing.T) {
	payload := map[string]any{
		"items": []any{
			map[string]any{
				"id": "a",
			},
			map[string]any{
				"id": "b",
			},
		},
	}
	var paths []string
	WalkPayload(payload, "", func(path string, value any) WalkControl {
		paths = append(paths, path)
		return WalkContinue
	})
	assert.ElementsMatch(t, []string{"items[0].id", "items[1].id"}, paths)
}

func TestWalkPayload_stopsEarly(t *testing.T) {
	payload := []any{"first", "second", "third"}
	var visited []string
	result := WalkPayload(payload, "", func(path string, value any) WalkControl {
		visited = append(visited, path)
		if path == "[0]" {
			return WalkStop
		}
		return WalkContinue
	})
	assert.Equal(t, WalkStop, result)
	assert.Equal(t, []string{"[0]"}, visited)
}

func TestWalkPayload_leafAtRoot(t *testing.T) {
	var path string
	WalkPayload("hello", "", func(p string, value any) WalkControl {
		path = p
		assert.Equal(t, "hello", value)
		return WalkContinue
	})
	assert.Equal(t, "", path)
}
