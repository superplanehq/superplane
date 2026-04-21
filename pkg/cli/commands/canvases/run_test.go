package canvases

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const runTestCanvasID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func runDescribeResponse(canvasID string, templates []map[string]any) string {
	canvas := map[string]any{
		"canvas": map[string]any{
			"metadata": map[string]any{
				"id":   canvasID,
				"name": "run-test",
			},
			"spec": map[string]any{
				"nodes": []map[string]any{
					{
						"id":   "start-node",
						"name": "Manual Run",
						"type": "TYPE_TRIGGER",
						"trigger": map[string]any{
							"name": "start",
						},
						"configuration": map[string]any{
							"templates": templates,
						},
					},
					{
						"id":   "component-node",
						"name": "Some Component",
						"type": "TYPE_COMPONENT",
					},
				},
				"edges": []map[string]any{},
			},
		},
	}

	raw, err := json.Marshal(canvas)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func TestRunCommandRequiresCanvasAndFlags(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")

	cmd := &runCommand{node: strPtr(""), template: strPtr("Hello"), payloadJSON: strPtr("")}
	err := cmd.Execute(ctx)
	require.ErrorContains(t, err, "canvas")

	ctx.Args = []string{"my-canvas"}

	cmd = &runCommand{node: strPtr(""), template: strPtr("Hello"), payloadJSON: strPtr("")}
	err = cmd.Execute(ctx)
	require.ErrorContains(t, err, "--node is required")

	cmd = &runCommand{node: strPtr("start-node"), template: strPtr(""), payloadJSON: strPtr("")}
	err = cmd.Execute(ctx)
	require.ErrorContains(t, err, "--template is required")
}

func TestRunCommandEmitsEventWithSavedTemplatePayload(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello, World!"}},
	}

	var emittedBody map[string]any

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + runTestCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(runDescribeResponse(runTestCanvasID, templates)))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + runTestCanvasID + "/nodes/start-node/events",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				require.NoError(t, json.Unmarshal(raw, &emittedBody))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"eventId":"evt-1"}`))
			},
		},
	)

	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{runTestCanvasID}

	cmd := &runCommand{
		node:        strPtr("start-node"),
		template:    strPtr("Hello World"),
		payloadJSON: strPtr(""),
	}
	err := cmd.Execute(ctx)
	require.NoError(t, err)

	require.Equal(t, "Hello World", emittedBody["channel"])
	data, ok := emittedBody["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello, World!", data["message"])
	require.Contains(t, stdout.String(), "evt-1")

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + runTestCanvasID,
		http.MethodPost + " /api/v1/canvases/" + runTestCanvasID + "/nodes/start-node/events",
	})
}

func TestRunCommandUsesPayloadOverride(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello, World!"}},
	}

	var emittedBody map[string]any

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + runTestCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(runDescribeResponse(runTestCanvasID, templates)))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + runTestCanvasID + "/nodes/start-node/events",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				require.NoError(t, json.Unmarshal(raw, &emittedBody))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"eventId":"evt-2"}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{runTestCanvasID}

	override := `{"message":"Override"}`
	cmd := &runCommand{
		node:        strPtr("start-node"),
		template:    strPtr("Hello World"),
		payloadJSON: &override,
	}
	require.NoError(t, cmd.Execute(ctx))

	data, ok := emittedBody["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Override", data["message"])
}

func TestRunCommandRejectsUnknownTemplate(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello, World!"}},
	}

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + runTestCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(runDescribeResponse(runTestCanvasID, templates)))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{runTestCanvasID}

	cmd := &runCommand{
		node:        strPtr("start-node"),
		template:    strPtr("Does Not Exist"),
		payloadJSON: strPtr(""),
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Does Not Exist")
	require.Contains(t, err.Error(), "Hello World")
}

func TestRunCommandRejectsNonTriggerNode(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello"}},
	}

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + runTestCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(runDescribeResponse(runTestCanvasID, templates)))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{runTestCanvasID}

	cmd := &runCommand{
		node:        strPtr("component-node"),
		template:    strPtr("Hello World"),
		payloadJSON: strPtr(""),
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not a trigger")
}

func TestRunCommandRejectsMissingNode(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello"}},
	}

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases/" + runTestCanvasID,
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(runDescribeResponse(runTestCanvasID, templates)))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{runTestCanvasID}

	cmd := &runCommand{
		node:        strPtr("nonexistent"),
		template:    strPtr("Hello World"),
		payloadJSON: strPtr(""),
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func strPtr(s string) *string {
	return &s
}
