package queue

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/cli"
)

const (
	testCanvasID  = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"
	testNodeID    = "node-1"
	testItemID    = "item-1"
	testQueuePath = "/api/v1/canvases/" + testCanvasID + "/nodes/" + testNodeID + "/queue"
)

func writeQueueItemsResponse(w http.ResponseWriter, payload string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(payload))
}

func newQueueServer(t *testing.T, listPayload string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testQueuePath:
			writeQueueItemsResponse(w, listPayload)
		case r.Method == http.MethodDelete && r.URL.Path == testQueuePath+"/"+testItemID:
			writeQueueItemsResponse(w, `{}`)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestListRendersQueueItems(t *testing.T) {
	server := newQueueServer(t,
		`{"items":[{"id":"`+testItemID+`","createdAt":"2024-01-01T10:00:00Z","rootEvent":{"id":"evt-1","nodeId":"`+testNodeID+`"}}]}`)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	canvasID := testCanvasID
	nodeID := testNodeID
	require.NoError(t, (&ListQueueItemsCommand{CanvasID: &canvasID, NodeID: &nodeID}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "CREATED_AT")
	require.Contains(t, out, "ROOT_EVENT_ID")
	require.Contains(t, out, "SOURCE")
	require.Contains(t, out, testItemID)
	require.Contains(t, out, "2024-01-01T10:00:00Z")
	require.Contains(t, out, "evt-1")
}

func TestListRendersEmptyQueue(t *testing.T) {
	server := newQueueServer(t, `{"items":[]}`)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	canvasID := testCanvasID
	nodeID := testNodeID
	require.NoError(t, (&ListQueueItemsCommand{CanvasID: &canvasID, NodeID: &nodeID}).Execute(ctx))
	require.Contains(t, stdout.String(), "No queue items found.")
}

func TestListUsesActiveAppWhenNoAppFlag(t *testing.T) {
	server := newQueueServer(t, `{"items":[]}`)

	ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{
		ActiveApp: testCanvasID,
	})
	nodeID := testNodeID
	canvasID := ""
	require.NoError(t, (&ListQueueItemsCommand{CanvasID: &canvasID, NodeID: &nodeID}).Execute(ctx))
}

func TestDeleteQueueItem(t *testing.T) {
	server := newQueueServer(t, `{}`)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	canvasID := testCanvasID
	nodeID := testNodeID
	itemID := testItemID
	require.NoError(t, (&DeleteQueueItemCommand{CanvasID: &canvasID, NodeID: &nodeID, ItemID: &itemID}).Execute(ctx))
	require.Contains(t, stdout.String(), "Queue item deleted: "+testItemID)
}

func TestListAPIErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == testQueuePath:
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := cli.NewCommandContext(t, server, "text")
	canvasID := testCanvasID
	nodeID := testNodeID
	err := (&ListQueueItemsCommand{CanvasID: &canvasID, NodeID: &nodeID}).Execute(ctx)
	require.Error(t, err)
}
