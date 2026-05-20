package models

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestParseDashboard(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Dashboard
metadata:
  name: Deploy health
spec:
  panels:
    - id: status
      type: markdown
      content:
        title: Status
        body: Ready
  layout:
    - i: status
      x: 0
      y: 0
      w: 4
      h: 3
`)

	resource, err := ParseDashboard(raw)
	require.NoError(t, err)
	require.Equal(t, DashboardKind, resource.Kind)
	require.Equal(t, "Deploy health", resource.Metadata.Name)
	require.Len(t, resource.Spec.Panels, 1)
	require.Equal(t, "status", resource.Spec.Panels[0].GetId())
	require.Equal(t, "Status", resource.Spec.Panels[0].GetContent()["title"])
	require.Len(t, resource.Spec.Layout, 1)
	require.Equal(t, int32(4), resource.Spec.Layout[0].GetW())
}

func TestParseDashboardRejectsUnknownFields(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Dashboard
spec:
  panels: []
  layout: []
unexpected: true
`)

	_, err := ParseDashboard(raw)
	require.Error(t, err)
	require.Contains(t, err.Error(), `unknown field "unexpected"`)
}

func TestDashboardConversions(t *testing.T) {
	dashboard := openapi_client.CanvasesCanvasDashboard{}
	dashboard.SetCanvasId("canvas-123")
	dashboard.SetPanels([]openapi_client.CanvasesDashboardPanel{
		{Id: openapi_client.PtrString("p1"), Type: openapi_client.PtrString("markdown")},
	})
	dashboard.SetLayout([]openapi_client.CanvasesDashboardLayoutItem{
		{I: openapi_client.PtrString("p1"), W: openapi_client.PtrInt32(4), H: openapi_client.PtrInt32(2)},
	})

	resource := DashboardResourceFromDashboard(dashboard, "My Canvas")
	require.Equal(t, DashboardAPIVersion, resource.APIVersion)
	require.Equal(t, DashboardKind, resource.Kind)
	require.Equal(t, "canvas-123", resource.Metadata.CanvasID)
	require.Equal(t, "My Canvas", resource.Metadata.Name)
	require.Len(t, resource.Spec.Panels, 1)

	body := UpdateDashboardRequestFromDashboard(resource)
	require.Len(t, body.GetPanels(), 1)
	require.Len(t, body.GetLayout(), 1)
}
