package canvas

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/layout"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"github.com/superplanehq/superplane/test/support/cli"
)

type requestExpectation struct {
	method string
	path   string
	handle func(t *testing.T, w http.ResponseWriter, r *http.Request)
}

type apiTestServer struct {
	t            *testing.T
	expectations []requestExpectation
	calls        []string
	server       *httptest.Server
}

func newAPITestServer(t *testing.T, expectations ...requestExpectation) *apiTestServer {
	t.Helper()

	s := &apiTestServer{t: t, expectations: expectations}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls = append(s.calls, r.Method+" "+r.URL.Path)

		if len(s.expectations) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		next := s.expectations[0]
		require.Equal(t, next.method, r.Method)
		require.Equal(t, next.path, r.URL.Path)

		s.expectations = s.expectations[1:]
		if next.handle != nil {
			next.handle(t, w, r)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *apiTestServer) AssertCalls(t *testing.T, calls []string) {
	t.Helper()
	require.Equal(t, calls, s.calls)
	require.Len(t, s.expectations, 0, "unused request expectations")
}

const cliTestUserID = "user-1"

func draftVersionsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions"
}

func expectMe() requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   "/api/v1/me",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"` + cliTestUserID + `"}}`))
		},
	}
}

func expectListDraftBranchesEmpty(canvasID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[]}`))
		},
	}
}

func expectCreateDraftBranch(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodPost,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + versionID + `"}}}`))
		},
	}
}

func repositoryCanvasFilePath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/repository/file"
}

func repositoryCommitsPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/repository/commits"
}

func describeVersionPath(canvasID, versionID string) string {
	return "/api/v1/canvases/" + canvasID + "/versions/" + versionID
}

func stagingPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/staging"
}

func expectListLiveVersion(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   draftVersionsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"` + versionID + `"}}]}`))
		},
	}
}

func expectStageCanvasYAML(canvasID string) requestExpectation {
	return requestExpectation{
		method: http.MethodPut,
		path:   stagingPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml"]}}`))
		},
	}
}

func expectCommitStaging(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodPost,
		path:   stagingPath(canvasID) + "/commit",
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + versionID + `","canvasId":"` + canvasID + `"}}}`))
		},
	}
}

func expectCommitCanvasYAML(canvasID, versionID string, assertYAML func(t *testing.T, yaml string)) requestExpectation {
	return requestExpectation{
		method: http.MethodPost,
		path:   repositoryCommitsPath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			rawBody, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload map[string]any
			require.NoError(t, json.Unmarshal(rawBody, &payload))
			require.Equal(t, versionID, payload["versionId"])
			if assertYAML != nil {
				operations, ok := payload["operations"].([]any)
				require.True(t, ok)
				require.NotEmpty(t, operations)
				first, ok := operations[0].(map[string]any)
				require.True(t, ok)
				encoded, ok := first["content"].(string)
				require.True(t, ok)
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				require.NoError(t, err)
				assertYAML(t, string(decoded))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		},
	}
}

func expectDescribeCanvasVersion(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   describeVersionPath(canvasID, versionID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + versionID + `","canvasId":"` + canvasID + `"}}}`))
		},
	}
}

// expectValidateDraftVersion mocks the describe-version call used by
// common.ResolveDraftVersionID to validate an explicit --draft-id. It
// returns a draft version owned by the test user.
func expectValidateDraftVersion(canvasID, versionID string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   describeVersionPath(canvasID, versionID),
		handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"` + versionID + `","canvasId":"` + canvasID + `"}}}`))
		},
	}
}

func expectFetchCanvasYAML(canvasID, versionID, yamlBody string) requestExpectation {
	return requestExpectation{
		method: http.MethodGet,
		path:   repositoryCanvasFilePath(canvasID),
		handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "canvas.yaml", r.URL.Query().Get("path"))
			require.Equal(t, versionID, r.URL.Query().Get("version_id"))
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(yamlBody))
		},
	}
}

func TestUpdateWithoutFileReturnsError(t *testing.T) {
	server := newAPITestServer(t)
	ctx, _ := cli.NewCommandContext(t, server.server, "text")
	err := (&updateCommand{}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "file")
}

func writeTestCanvasFileWithoutMetadataID(t *testing.T, name string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}

func writeTestCanvasFileWithMetadataID(t *testing.T, name, canvasID string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: " + canvasID + "\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))
	return filePath
}

func TestResolveCanvasForFileUpdateWithoutIDReturnsError(t *testing.T) {
	filePath := writeTestCanvasFileWithoutMetadataID(t, "parse-check")
	_, _, err := resolveCanvasForFileUpdate(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.id is required")
}

func TestResolveCanvasForFileUpdateWithIDSucceeds(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	filePath := writeTestCanvasFileWithMetadataID(t, "my-canvas", canvasID)
	gotID, canvas, err := resolveCanvasForFileUpdate(filePath)
	require.NoError(t, err)
	require.Equal(t, canvasID, gotID)
	md := canvas.GetMetadata()
	require.Equal(t, canvasID, (&md).GetId())
}

func TestResolveCanvasForFileUpdateWhitespaceIDReturnsError(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	content := []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: \"   \"\n  name: x\nspec:\n  nodes: []\n  edges: []\n",
	)
	require.NoError(t, os.WriteFile(filePath, content, 0o644))

	_, _, err := resolveCanvasForFileUpdate(filePath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.id is required")
}

func TestUpdateFromFileJSONOutputWhenDraft(t *testing.T) {
	t.Helper()
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		expectValidateDraftVersion(canvasID, "version-1"),
		expectStageCanvasYAML(canvasID),
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "json-out", canvasID)
	file := filePath
	draftID := "version-1"
	ctx, stdout := cli.NewCommandContext(t, server.server, "json")

	err := (&updateCommand{file: &file, draftID: &draftID}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), `"staged": "true"`)

	server.AssertCalls(t, []string{
		http.MethodGet + " " + describeVersionPath(canvasID, "version-1"),
		http.MethodPut + " " + stagingPath(canvasID),
	})
}

func TestUpdateFromFileWhenCommitFailsReturnsWrappedError(t *testing.T) {
	t.Helper()
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		expectListLiveVersion(canvasID, "ver-1"),
		expectStageCanvasYAML(canvasID),
		requestExpectation{
			method: http.MethodPost,
			path:   stagingPath(canvasID) + "/commit",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"commit failed"}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "pub-fail", canvasID)
	file := filePath
	ctx, _ := cli.NewCommandContext(t, server.server, "text")

	err := (&updateCommand{file: &file}).Execute(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "canvas was staged but commit failed"), err.Error())

	server.AssertCalls(t, []string{
		http.MethodGet + " " + draftVersionsPath(canvasID),
		http.MethodPut + " " + stagingPath(canvasID),
		http.MethodPost + " " + stagingPath(canvasID) + "/commit",
	})
}

func TestUpdateFromFileWhenStageFailsReturnsError(t *testing.T) {
	t.Helper()
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		expectValidateDraftVersion(canvasID, "version-1"),
		requestExpectation{
			method: http.MethodPut,
			path:   stagingPath(canvasID),
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"message":"version conflict"}`))
			},
		},
	)

	filePath := writeTestCanvasFileWithMetadataID(t, "put-fail", canvasID)
	file := filePath
	draftID := "version-1"
	ctx, _ := cli.NewCommandContext(t, server.server, "text")

	err := (&updateCommand{file: &file, draftID: &draftID}).Execute(ctx)
	require.Error(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " " + describeVersionPath(canvasID, "version-1"),
		http.MethodPut + " " + stagingPath(canvasID),
	})
}

func TestUpdateFromFileTextOutputCountsIntegrations(t *testing.T) {
	t.Helper()
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

	server := newAPITestServer(
		t,
		expectListLiveVersion(canvasID, "version-1"),
		expectStageCanvasYAML(canvasID),
		expectCommitStaging(canvasID, "version-1"),
		expectFetchCanvasYAML(
			canvasID,
			"version-1",
			"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: "+canvasID+"\n  name: integ\nspec:\n  nodes:\n    - id: n1\n      component: noop\n      integration:\n        id: int-1\n  edges: []\n",
		),
	)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "canvas.yaml")
	yaml := "" +
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: " + canvasID + "\n  name: integ\nspec:\n  nodes: []\n  edges: []\n"
	require.NoError(t, os.WriteFile(filePath, []byte(yaml), 0o644))

	file := filePath
	ctx, stdout := cli.NewCommandContext(t, server.server, "text")

	err := (&updateCommand{file: &file}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Integrations: 1")

	server.AssertCalls(t, []string{
		http.MethodGet + " " + draftVersionsPath(canvasID),
		http.MethodPut + " " + stagingPath(canvasID),
		http.MethodPost + " " + stagingPath(canvasID) + "/commit",
		http.MethodGet + " " + repositoryCanvasFilePath(canvasID),
	})
}

func TestBuildDefaultAutoLayoutUsesFullCanvas(t *testing.T) {
	autoLayout := layout.DefaultAutoLayout()

	if autoLayout.GetAlgorithm() != openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL {
		t.Fatalf("expected horizontal auto-layout, got %s", autoLayout.GetAlgorithm())
	}
	if autoLayout.GetScope() != openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS {
		t.Fatalf("expected full-canvas scope, got %s", autoLayout.GetScope())
	}
	if autoLayout.HasNodeIds() {
		t.Fatalf("expected no node ids for default full-canvas strategy, got %v", autoLayout.GetNodeIds())
	}
}
