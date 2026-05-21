package models

import (
	"encoding/json"
	"testing"

	"github.com/ghodss/yaml"
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

func TestParseDashboardValidatesResourceIdentity(t *testing.T) {
	testCases := []struct {
		name          string
		raw           string
		errorContains string
	}{
		{
			name: "missing apiVersion",
			raw: `
kind: Dashboard
spec:
  panels: []
  layout: []
`,
			errorContains: "dashboard apiVersion is required",
		},
		{
			name: "unsupported apiVersion",
			raw: `
apiVersion: v2
kind: Dashboard
spec:
  panels: []
  layout: []
`,
			errorContains: `unsupported dashboard apiVersion "v2"`,
		},
		{
			name: "unsupported kind",
			raw: `
apiVersion: v1
kind: Canvas
spec:
  panels: []
  layout: []
`,
			errorContains: `unsupported resource kind "Canvas"`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := ParseDashboard([]byte(testCase.raw))
			require.Error(t, err)
			require.Contains(t, err.Error(), testCase.errorContains)
		})
	}
}

func TestParseDashboardDefaultsEmptyCollections(t *testing.T) {
	raw := []byte(`
apiVersion: v1
kind: Dashboard
spec: {}
`)

	resource, err := ParseDashboard(raw)
	require.NoError(t, err)
	require.NotNil(t, resource.Spec.Panels)
	require.Empty(t, resource.Spec.Panels)
	require.NotNil(t, resource.Spec.Layout)
	require.Empty(t, resource.Spec.Layout)
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

func TestDashboardResourceFromDashboardDefaultsEmptyCollections(t *testing.T) {
	dashboard := openapi_client.CanvasesCanvasDashboard{}
	dashboard.SetCanvasId("canvas-123")

	resource := DashboardResourceFromDashboard(dashboard, "My Canvas")

	jsonPayload, err := json.Marshal(resource)
	require.NoError(t, err)
	require.Contains(t, string(jsonPayload), `"panels":[]`)
	require.Contains(t, string(jsonPayload), `"layout":[]`)
	require.NotContains(t, string(jsonPayload), `"panels":null`)
	require.NotContains(t, string(jsonPayload), `"layout":null`)

	yamlPayload, err := yaml.Marshal(resource)
	require.NoError(t, err)
	require.Contains(t, string(yamlPayload), "panels: []")
	require.Contains(t, string(yamlPayload), "layout: []")
	require.NotContains(t, string(yamlPayload), "panels: null")
	require.NotContains(t, string(yamlPayload), "layout: null")
}
