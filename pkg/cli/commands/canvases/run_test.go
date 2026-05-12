package canvases

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunCommandRequiresCanvasArg(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{}

	trigger := "n1"
	template := "T1"
	cmd := &runCommand{trigger: &trigger, template: &template}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "required")
}

func TestRunCommandRequiresTrigger(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"some-canvas"}
	template := "T1"
	empty := ""
	cmd := &runCommand{trigger: &empty, template: &template}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--trigger")
}

func TestRunCommandRejectsTemplateWithReplay(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"4e9ae08d-0363-40d2-ba2c-5f6389a418d8"}
	trigger := "n1"
	template := "T1"
	replay := "evt-1"
	cmd := &runCommand{trigger: &trigger, template: &template, replay: &replay}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot use --template together with --replay")
}

func TestRunCommandRequiresTemplateOrReplay(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"4e9ae08d-0363-40d2-ba2c-5f6389a418d8"}
	trigger := "n1"
	empty := ""
	cmd := &runCommand{trigger: &trigger, template: &empty, replay: &empty}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "either --template or --replay")
}

func TestRunCommandRejectsPayloadFileWithReplay(t *testing.T) {
	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{"4e9ae08d-0363-40d2-ba2c-5f6389a418d8"}
	trigger := "n1"
	replay := "evt-1"
	payloadFile := "/tmp/x.json"
	empty := ""
	cmd := &runCommand{trigger: &trigger, template: &empty, replay: &replay, payloadFile: &payloadFile}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot use --payload-file with --replay")
}

func TestRunCommandInvokeManualHook(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	nodeID := "start-node"

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var payload map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &payload))
				params, ok := payload["parameters"].(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "Hello World", params["template"])

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":{"template":"Hello World"}}`))
			},
		},
	)

	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{canvasID}
	trigger := nodeID
	template := "Hello World"
	empty := ""
	cmd := &runCommand{trigger: &trigger, template: &template, replay: &empty, payloadFile: &empty}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "Hello World")
	server.AssertCalls(t, []string{
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
	})
}

func TestRunCommandInvokeManualHookWithPayloadFile(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	nodeID := "start-node"

	dir := t.TempDir()
	payloadPath := filepath.Join(dir, "payload.json")
	require.NoError(t, os.WriteFile(payloadPath, []byte(`{"message":"from-file"}`), 0o644))

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var envelope map[string]interface{}
				require.NoError(t, json.Unmarshal(body, &envelope))
				params := envelope["parameters"].(map[string]interface{})
				require.Equal(t, "Hello World", params["template"])
				override := params["payload"].(map[string]interface{})
				require.Equal(t, "from-file", override["message"])

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"result":{"template":"Hello World"}}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{canvasID}
	trigger := nodeID
	template := "Hello World"
	empty := ""
	cmd := &runCommand{trigger: &trigger, template: &template, replay: &empty, payloadFile: &payloadPath}

	require.NoError(t, cmd.Execute(ctx))
	server.AssertCalls(t, []string{
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
	})
}

func TestRunCommandReemitTriggerEvent(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	nodeID := "start-node"
	eventID := "f1e2d3c4-b2a1-0000-0000-000000000001"

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/events/" + eventID + "/reemit",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				require.Len(t, body, 0)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"eventId":"new-event-id"}`))
			},
		},
	)

	ctx, stdout := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{canvasID}
	trigger := nodeID
	empty := ""
	replay := eventID
	cmd := &runCommand{trigger: &trigger, template: &empty, replay: &replay}

	require.NoError(t, cmd.Execute(ctx))
	require.Contains(t, stdout.String(), "new-event-id")
	server.AssertCalls(t, []string{
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/events/" + eventID + "/reemit",
	})
}

func TestRunCommandResolveCanvasNameThenInvoke(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	nodeID := "start-node"

	server := newAPITestServer(
		t,
		requestExpectation{
			method: http.MethodGet,
			path:   "/api/v1/canvases",
			handle: func(t *testing.T, w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"canvases":[{"metadata":{"id":"` + canvasID + `","name":"my-canvas"}}]}`))
			},
		},
		requestExpectation{
			method: http.MethodPost,
			path:   "/api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
			handle: func(t *testing.T, w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
			},
		},
	)

	ctx, _ := newCreateCommandContextForTest(t, server.server, "text")
	ctx.Args = []string{"my-canvas"}
	trigger := nodeID
	template := "Hello World"
	empty := ""
	cmd := &runCommand{trigger: &trigger, template: &template, replay: &empty, payloadFile: &empty}

	require.NoError(t, cmd.Execute(ctx))
	server.AssertCalls(t, []string{
		http.MethodGet + " /api/v1/canvases",
		http.MethodPost + " /api/v1/canvases/" + canvasID + "/triggers/" + nodeID + "/hooks/run",
	})
}

func TestRunCommandPayloadFileMustBeObject(t *testing.T) {
	canvasID := "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	dir := t.TempDir()
	payloadPath := filepath.Join(dir, "payload.json")
	require.NoError(t, os.WriteFile(payloadPath, []byte(`["not","object"]`), 0o644))

	ctx, _ := newCreateCommandContextForTest(t, nil, "text")
	ctx.Args = []string{canvasID}
	trigger := "n1"
	template := "T"
	emptyReplay := ""
	cmd := &runCommand{trigger: &trigger, template: &template, payloadFile: &payloadPath, replay: &emptyReplay}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "JSON object"), err.Error())
}
