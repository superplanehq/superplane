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

func TestRunCommandInvokesHookWithTemplate(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello, World!"}},
	}

	var sentBody map[string]any

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
			path:   "/api/v1/canvases/" + runTestCanvasID + "/triggers/start-node/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				require.NoError(t, json.Unmarshal(raw, &sentBody))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":{"template":"Hello World"}}`))
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

	parameters, ok := sentBody["parameters"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Hello World", parameters["template"])
	require.NotContains(t, parameters, "payload", "payload override should not be sent when not provided")
	require.Contains(t, stdout.String(), "Run started")

	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases/" + runTestCanvasID,
		http.MethodPost + " /api/v1/canvases/" + runTestCanvasID + "/triggers/start-node/hooks/run",
	})
}

func TestRunCommandSendsPayloadOverride(t *testing.T) {
	templates := []map[string]any{
		{"name": "Hello World", "payload": map[string]any{"message": "Hello, World!"}},
	}

	var sentBody map[string]any

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
			path:   "/api/v1/canvases/" + runTestCanvasID + "/triggers/start-node/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				require.NoError(t, json.Unmarshal(raw, &sentBody))
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":{"template":"Hello World"}}`))
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

	parameters, ok := sentBody["parameters"].(map[string]any)
	require.True(t, ok)
	payload, ok := parameters["payload"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "Override", payload["message"])
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
