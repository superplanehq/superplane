package canvases

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const testReadmeCanvasID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func TestReadmeGetFetchesLiveContent(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "", r.URL.Query().Get("versionId"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"v1","versionState":"published","content":"# hello"}`))
			},
		},
	)

	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	target := testReadmeCanvasID
	ctx.Args = []string{target}

	draft := false
	version := ""
	err := (&readmeGetCommand{draft: &draft, version: &version}).Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, "# hello", stdout.String())
	server.AssertCalls(t, []string{http.MethodGet + " /api/v1/canvases/" + testReadmeCanvasID + "/readme"})
}

func TestReadmeGetDraftPassesVersionFlag(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "draft", r.URL.Query().Get("versionId"))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"v-draft","versionState":"draft","content":"draft"}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{testReadmeCanvasID}

	draft := true
	version := ""
	err := (&readmeGetCommand{draft: &draft, version: &version}).Execute(ctx)
	require.NoError(t, err)
}

func TestReadmeUpdatePublishesWhenChangeManagementDisabled(t *testing.T) {
	readmeContent := "# update"

	server := newAPITestServer(
		t,
		// 1. Ensure a draft exists: list versions (none)
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[]}`))
			},
		},
		// 2. Create draft
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","canvasId":"` + testReadmeCanvasID + `"}}}`))
			},
		},
		// 3. Update readme
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				var body map[string]any
				require.NoError(t, json.Unmarshal(raw, &body))
				require.Equal(t, "draft-1", body["versionId"])
				require.Equal(t, readmeContent, body["content"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"draft-1","content":"` + readmeContent + `"}`))
			},
		},
		// 4. Describe canvas (for change management check)
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testReadmeCanvasID + `"},"spec":{"changeManagement":{"enabled":false}}}}`))
			},
		},
		// 5. Publish draft
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions/draft-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1","canvasId":"` + testReadmeCanvasID + `"}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{testReadmeCanvasID}

	file := ""
	content := readmeContent
	message := ""
	draftOnly := false
	err := (&readmeUpdateCommand{
		file:      &file,
		content:   &content,
		message:   &message,
		draftOnly: &draftOnly,
	}).Execute(ctx)
	require.NoError(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + testReadmeCanvasID + "/versions",
		http.MethodPost + " /api/v1/canvases/" + testReadmeCanvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + testReadmeCanvasID + "/readme",
		http.MethodGet + " /api/v1/canvases/" + testReadmeCanvasID,
		http.MethodPatch + " /api/v1/canvases/" + testReadmeCanvasID + "/versions/draft-1/publish",
	})
}

func TestReadmeUpdateCreatesChangeRequestWhenChangeManagementEnabled(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"draft-1","content":"hi"}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testReadmeCanvasID + `"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/change-requests",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				var body map[string]any
				require.NoError(t, json.Unmarshal(raw, &body))
				require.Equal(t, "draft-1", body["versionId"])
				require.Equal(t, "Docs only", body["title"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"changeRequest":{"metadata":{"id":"cr-1","canvasId":"` + testReadmeCanvasID + `","versionId":"draft-1","status":"STATUS_OPEN"}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{testReadmeCanvasID}

	file := ""
	content := "hi"
	message := "Docs only"
	draftOnly := false
	err := (&readmeUpdateCommand{
		file:      &file,
		content:   &content,
		message:   &message,
		draftOnly: &draftOnly,
	}).Execute(ctx)
	require.NoError(t, err)
}

func TestReadmeUpdateDraftOnlyStopsAfterUpdate(t *testing.T) {
	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"draft-1","content":"hi"}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testReadmeCanvasID + `"},"spec":{"changeManagement":{"enabled":true}}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{testReadmeCanvasID}

	file := ""
	content := "hi"
	message := ""
	draftOnly := true
	err := (&readmeUpdateCommand{
		file:      &file,
		content:   &content,
		message:   &message,
		draftOnly: &draftOnly,
	}).Execute(ctx)
	require.NoError(t, err)

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + testReadmeCanvasID + "/versions",
		http.MethodPut + " /api/v1/canvases/" + testReadmeCanvasID + "/readme",
		http.MethodGet + " /api/v1/canvases/" + testReadmeCanvasID,
	})
}

func TestReadmeUpdateReadsFromFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(filePath, []byte("from file"), 0o600))

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"versions":[{"metadata":{"id":"draft-1","state":"STATE_DRAFT"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPut,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/readme",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				var body map[string]any
				require.NoError(t, json.Unmarshal(raw, &body))
				require.Equal(t, "from file", body["content"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvasId":"` + testReadmeCanvasID + `","versionId":"draft-1","content":"from file"}`))
			},
		},
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + testReadmeCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvas":{"metadata":{"id":"` + testReadmeCanvasID + `"},"spec":{"changeManagement":{"enabled":false}}}}`))
			},
		},
		requestExpectation{
			method: http.MethodPatch,
			path:   "/api/v1/canvases/" + testReadmeCanvasID + "/versions/draft-1/publish",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"draft-1"}}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{testReadmeCanvasID}

	file := filePath
	content := ""
	message := ""
	draftOnly := false
	err := (&readmeUpdateCommand{
		file:      &file,
		content:   &content,
		message:   &message,
		draftOnly: &draftOnly,
	}).Execute(ctx)
	require.NoError(t, err)
}
