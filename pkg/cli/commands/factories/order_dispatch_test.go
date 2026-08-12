package factories

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	cli "github.com/superplanehq/superplane/test/support/cli"
)

const testOrderDispatchFactoryID = "11111111-1111-1111-1111-111111111111"
const testOrderDispatchOrderID = "order-1"

func newOrderDispatchServer(t *testing.T, capture *map[string]any, statusCode int, respBody string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v1/factories/"+testOrderDispatchFactoryID+"/orders/"+testOrderDispatchOrderID+"/dispatch" {
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if capture != nil {
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			*capture = body
		}
		w.Header().Set("Content-Type", "application/json")
		if statusCode != 0 {
			w.WriteHeader(statusCode)
		}
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestOrderDispatchCommand_OrderRequired(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	line := "build"
	err := (&orderDispatchCommand{line: &line}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--order is required")
}

func TestOrderDispatchCommand_LineRequired(t *testing.T) {
	ctx, _ := cli.NewCommandContext(t, nil, "text")
	orderID := testOrderDispatchOrderID
	err := (&orderDispatchCommand{orderID: &orderID}).Execute(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--line is required")
}

func TestOrderDispatchCommand_Success(t *testing.T) {
	var body map[string]any
	server := newOrderDispatchServer(t, &body, 0, `{"order":{"id":"order-1","title":"Ship it","state":"STATE_OPEN"}}`)
	ctx, stdout := cli.NewCommandContext(t, server, "text")

	factory := testOrderDispatchFactoryID
	orderID := testOrderDispatchOrderID
	line := "build"
	err := (&orderDispatchCommand{factory: &factory, orderID: &orderID, line: &line}).Execute(ctx)
	require.NoError(t, err)

	require.NotNil(t, body)
	assert.Equal(t, "build", body["lineName"])

	out := stdout.String()
	assert.Contains(t, out, "Work order dispatched: order-1")
	assert.Contains(t, out, `"build"`)
	assert.Contains(t, out, "Open")
}

func TestOrderDispatchCommand_JSONOutput(t *testing.T) {
	server := newOrderDispatchServer(t, nil, 0, `{"order":{"id":"order-1","title":"Ship it","state":"STATE_OPEN"}}`)
	ctx, stdout := cli.NewCommandContext(t, server, "json")

	factory := testOrderDispatchFactoryID
	orderID := testOrderDispatchOrderID
	line := "build"
	err := (&orderDispatchCommand{factory: &factory, orderID: &orderID, line: &line}).Execute(ctx)
	require.NoError(t, err)

	assert.Contains(t, stdout.String(), `"id": "order-1"`)
}

func TestOrderDispatchCommand_ErrorPropagation(t *testing.T) {
	server := newOrderDispatchServer(t, nil, http.StatusConflict, `{"code":9,"message":"work order already has an active execution"}`)
	ctx, _ := cli.NewCommandContext(t, server, "text")

	factory := testOrderDispatchFactoryID
	orderID := testOrderDispatchOrderID
	line := "build"
	err := (&orderDispatchCommand{factory: &factory, orderID: &orderID, line: &line}).Execute(ctx)
	require.Error(t, err)
	// FormatCommandError is what "superplane factory orders dispatch" actually
	// runs errors through (via core.Bind); apply it here too so the assertion
	// matches the message a user would see, not the raw HTTP client error.
	formatted := core.FormatCommandError(err)
	assert.Contains(t, formatted.Error(), "already has an active execution")
}
