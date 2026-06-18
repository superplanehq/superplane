package console

import (
	"strings"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/require"
)

func TestParseConsoleYAMLRejectsWrongHeaders(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		message string
	}{
		{"empty", "", "empty"},
		{"missing apiVersion", "kind: Console\nmetadata: {}\nspec: {panels: [], layout: []}\n", "apiVersion is required"},
		{"wrong apiVersion", "apiVersion: v2\nkind: Console\nmetadata: {}\nspec: {panels: [], layout: []}\n", "unsupported apiVersion"},
		{"missing kind", "apiVersion: v1\nmetadata: {}\nspec: {panels: [], layout: []}\n", "kind is required"},
		{"legacy dashboard kind", "apiVersion: v1\nkind: Dashboard\nmetadata: {}\nspec: {panels: [], layout: []}\n", "unsupported kind"},
		{"unknown top-level field", "apiVersion: v1\nkind: Console\nmetadata: {}\nspec: {panels: [], layout: []}\nextra: 1\n", "unknown field"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseConsoleYAML([]byte(tc.input))
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.message)
		})
	}
}

func TestParseConsoleYAMLRoundTrip(t *testing.T) {
	parsed, err := ParseConsoleYAML([]byte(`apiVersion: v1
kind: Console
metadata:
  canvasId: c-1
  name: my
spec:
  panels:
    - id: a
      type: markdown
      content:
        body: hi
  layout:
    - i: a
      x: 0
      y: 0
      w: 4
      h: 2
      minW: 2
`))
	require.NoError(t, err)
	require.Equal(t, "v1", parsed.APIVersion)
	require.Equal(t, "Console", parsed.Kind)
	require.Equal(t, "c-1", parsed.Metadata.CanvasID)
	require.Equal(t, "my", parsed.Metadata.Name)
	require.Len(t, parsed.Spec.Panels, 1)
	require.Equal(t, "a", parsed.Spec.Panels[0].ID)
	require.Equal(t, "hi", parsed.Spec.Panels[0].Content["body"])
	require.NotNil(t, parsed.Spec.Layout[0].MinW)
	require.Equal(t, 2, *parsed.Spec.Layout[0].MinW)

	// Production YAML output uses core.Renderer (ghodss/yaml), not yaml.v3.
	encoded, err := yaml.Marshal(parsed)
	require.NoError(t, err)
	output := string(encoded)
	require.True(t, strings.HasPrefix(output, "apiVersion: v1"))
	require.Contains(t, output, "kind: Console")
	require.Contains(t, output, "minW: 2")
}
