package installation

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubResponse struct {
	status int
	body   string
}

// stubHTTP overrides the package fetcher so installation tests stay
// deterministic instead of depending on mutable upstream repositories. URLs
// absent from the map respond with 404.
func stubHTTP(t *testing.T, responses map[string]stubResponse) {
	t.Helper()
	original := httpGet
	httpGet = func(rawURL string) (*http.Response, error) {
		resp, ok := responses[rawURL]
		if !ok {
			resp = stubResponse{status: http.StatusNotFound}
		}
		return &http.Response{
			StatusCode: resp.status,
			Body:       io.NopCloser(strings.NewReader(resp.body)),
		}, nil
	}
	t.Cleanup(func() { httpGet = original })
}

func TestParseCanvasYAMLWithResourceHeaders(t *testing.T) {
	raw := []byte(`apiVersion: v1
kind: Canvas
metadata:
  name: Preview Environments
  description: StoreJS preview
spec:
  nodes: []
  edges: []
`)

	canvas, err := parseCanvasYAML(raw)
	require.NoError(t, err)
	assert.Equal(t, "Preview Environments", canvas.GetMetadata().GetName())
	assert.Equal(t, "StoreJS preview", canvas.GetMetadata().GetDescription())
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

func TestFetchCanvasResolvesRefAndParses(t *testing.T) {
	repo := &Repository{Owner: "acme", Name: "demo"}
	stubHTTP(t, map[string]stubResponse{
		rawFileURL(repo, "main", canvasFileName): {
			status: http.StatusOK,
			body: `apiVersion: v1
kind: Canvas
metadata:
  name: Preview Environments
spec:
  nodes:
    - name: start
  edges: []
`,
		},
	})

	canvas, ref, err := FetchCanvas(repo)
	require.NoError(t, err)
	assert.Equal(t, "main", ref)
	assert.Equal(t, "Preview Environments", canvas.GetMetadata().GetName())
	assert.NotEmpty(t, canvas.GetSpec().GetNodes())
}

func TestFetchConsoleRequiresRef(t *testing.T) {
	_, err := FetchConsole(&Repository{Owner: "acme", Name: "demo"}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolved ref")
}

func TestFetchConsoleReturnsNilWhenMissing(t *testing.T) {
	// A missing console.yaml (404) is opt-in: FetchConsole must return
	// (nil, nil) so apps without a bundled console still install cleanly.
	stubHTTP(t, map[string]stubResponse{})

	repo := &Repository{Owner: "acme", Name: "demo"}

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
