package installation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCanvasYAMLWithResourceHeaders(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Preview Environments
  description: StoreJS preview
  isTemplate: false
spec:
  nodes: []
  edges: []
`)

	canvas, err := parseCanvasYAML(raw)
	require.NoError(t, err)
	assert.Equal(t, "Preview Environments", canvas.GetMetadata().GetName())
	assert.Equal(t, "StoreJS preview", canvas.GetMetadata().GetDescription())
	assert.False(t, canvas.GetMetadata().GetIsTemplate())
	assert.Empty(t, canvas.GetMetadata().GetId())
}

func TestParseCanvasYAMLRejectsUnsupportedKind(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Dashboard
metadata:
  name: Not a canvas
spec:
  panels: []
`)

	_, err := parseCanvasYAML(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported resource kind "Dashboard"`)
}

func TestFetchCanvasFromPublicRepo(t *testing.T) {
	repo, err := ParseRepository("github.com/superplanehq/preview-env-github-digitalocean")
	require.NoError(t, err)

	canvas, ref, err := FetchCanvas(repo)
	require.NoError(t, err)
	assert.Equal(t, "main", ref)
	assert.NotEmpty(t, canvas.GetMetadata().GetName())
	assert.NotEmpty(t, canvas.GetSpec().GetNodes())
}

func TestFetchConsoleRequiresRef(t *testing.T) {
	_, err := FetchConsole(&Repository{Owner: "acme", Name: "demo"}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolved ref")
}

func TestFetchConsoleReturnsNilWhenMissing(t *testing.T) {
	// The reference app repo ships a canvas.yaml but no console.yaml.
	// FetchConsole must treat the missing file as opt-in (nil, nil) so that
	// apps without a bundled console still install cleanly.
	repo, err := ParseRepository("github.com/superplanehq/preview-env-github-digitalocean")
	require.NoError(t, err)

	console, err := FetchConsole(repo, "main")
	require.NoError(t, err)
	assert.Nil(t, console)
}

func TestRawFileURLBuildsExpectedPath(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	assert.Equal(t,
		"https://raw.githubusercontent.com/acme/demo/main/console.yaml",
		rawFileURL(repo, "main", consoleFileName),
	)
}
