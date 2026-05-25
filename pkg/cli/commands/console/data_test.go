package console

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDataReturnsMemoryRowsFromAPI(t *testing.T) {
	server := newRouteAPITestServer(t, map[string]string{
		"/api/v1/canvases/canvas-123/dashboard": `{
			"dashboard": {
				"panels": [{
					"id": "p1",
					"type": "table",
					"content": {
						"dataSource": {"kind": "memory", "namespace": "users"},
						"render": {"kind": "table", "columns": [{"field": "id"}]}
					}
				}]
			}
		}`,
		"/api/v1/canvases/canvas-123/memory": `{
			"items": [
				{"namespace": "users", "values": {"id": "u1", "name": "Alice"}},
				{"namespace": "other", "values": {"id": "x"}}
			]
		}`,
	})

	ctx, stdout := newCommandContext(t, server, "json")
	ctx.Args = []string{"p1"}
	cmd := &dataCommand{canvasID: stringPtr("canvas-123"), limit: int64Ptr(0)}

	require.NoError(t, cmd.Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, `"source": "memory"`)
	require.Contains(t, out, `"id": "u1"`)
	require.Contains(t, out, `"name": "Alice"`)
	require.NotContains(t, out, `"id": "x"`)
}

func TestDataReturnsRunsRowsFromAPI(t *testing.T) {
	server := newRouteAPITestServer(t, map[string]string{
		"/api/v1/canvases/canvas-123/dashboard": `{
			"dashboard": {
				"panels": [{
					"id": "p1",
					"type": "table",
					"content": {
						"dataSource": {"kind": "runs", "limit": 5},
						"render": {"kind": "table", "columns": [{"field": "id"}]}
					}
				}]
			}
		}`,
		"/api/v1/canvases/canvas-123/runs": `{
			"runs": [
				{"id": "r1", "state": "STATE_FINISHED"},
				{"id": "r2", "state": "STATE_STARTED"}
			],
			"totalCount": 2,
			"hasNextPage": false
		}`,
	})

	ctx, stdout := newCommandContext(t, server, "json")
	ctx.Args = []string{"p1"}
	cmd := &dataCommand{canvasID: stringPtr("canvas-123"), limit: int64Ptr(0)}

	require.NoError(t, cmd.Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, `"source": "runs"`)
	require.Contains(t, out, `"id": "r1"`)
	require.Contains(t, out, `"id": "r2"`)
	require.Contains(t, out, `"totalCount": 2`)
}

func TestDataReportsMissingDataSourceForMarkdownPanel(t *testing.T) {
	server := newRouteAPITestServer(t, map[string]string{
		"/api/v1/canvases/canvas-123/dashboard": `{
			"dashboard": {
				"panels": [{"id": "md", "type": "markdown", "content": {"title": "Notes"}}]
			}
		}`,
	})

	ctx, _ := newCommandContext(t, server, "text")
	ctx.Args = []string{"md"}
	cmd := &dataCommand{canvasID: stringPtr("canvas-123"), limit: int64Ptr(0)}

	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no data source")
}

func TestDataFlattensMemoryFieldPath(t *testing.T) {
	server := newRouteAPITestServer(t, map[string]string{
		"/api/v1/canvases/canvas-123/dashboard": `{
			"dashboard": {
				"panels": [{
					"id": "p1",
					"type": "table",
					"content": {
						"dataSource": {"kind": "memory", "namespace": "events", "fieldPath": "items"},
						"render": {"kind": "table", "columns": [{"field": "id"}]}
					}
				}]
			}
		}`,
		"/api/v1/canvases/canvas-123/memory": `{
			"items": [
				{"namespace": "events", "values": {"items": [{"id": "1"}, {"id": "2"}]}}
			]
		}`,
	})

	ctx, stdout := newCommandContext(t, server, "json")
	ctx.Args = []string{"p1"}
	cmd := &dataCommand{canvasID: stringPtr("canvas-123"), limit: int64Ptr(0)}

	require.NoError(t, cmd.Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, `"id": "1"`)
	require.Contains(t, out, `"id": "2"`)
}
