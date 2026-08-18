package files

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "keeps relative path", path: "workflows/release.yaml", expected: "workflows/release.yaml"},
		{name: "trims surrounding whitespace", path: "  workflows/release.yaml  ", expected: "workflows/release.yaml"},
		{name: "removes leading separators", path: "///workflows/release.yaml", expected: "workflows/release.yaml"},
		{name: "normalizes windows separators", path: `\workflows\release.yaml`, expected: "workflows/release.yaml"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, NormalizePath(testCase.path))
		})
	}
}

func TestIsSpecFilePath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "canvas spec", path: CanvasYAMLPath, expected: true},
		{name: "console spec", path: ConsoleYAMLPath, expected: true},
		{name: "nested canvas filename", path: "examples/canvas.yaml", expected: false},
		{name: "other yaml file", path: "workflow.yaml", expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsSpecFilePath(testCase.path))
		})
	}
}
