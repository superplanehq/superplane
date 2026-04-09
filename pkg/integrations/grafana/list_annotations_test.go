package grafana

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__validateListAnnotationTimeRangeMS__OK(t *testing.T) {
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 200))
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 100))
	require.NoError(t, validateListAnnotationTimeRangeMS(0, 200))
	require.NoError(t, validateListAnnotationTimeRangeMS(100, 0))
}

func Test__validateListAnnotationTimeRangeMS__RejectsInvertedRange(t *testing.T) {
	err := validateListAnnotationTimeRangeMS(200, 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "to must be at or after from")
}

func Test__ListAnnotations__Configuration__timeFieldsAreStrings(t *testing.T) {
	component := &ListAnnotations{}
	fields := component.Configuration()

	fieldTypes := map[string]string{}
	fieldDefaults := map[string]any{}
	fieldPlaceholders := map[string]string{}
	for _, field := range fields {
		fieldTypes[field.Name] = field.Type
		fieldDefaults[field.Name] = field.Default
		fieldPlaceholders[field.Name] = field.Placeholder
	}

	require.Equal(t, "string", fieldTypes["from"])
	require.Equal(t, "string", fieldTypes["to"])
	require.Equal(t, `{{ now() - duration("1h") }}`, fieldDefaults["from"])
	require.Equal(t, `{{ now() - duration("1h") }}`, fieldPlaceholders["from"])
	require.Equal(t, `{{ now() }}`, fieldDefaults["to"])
	require.Equal(t, `{{ now() }}`, fieldPlaceholders["to"])
}

func Test__ListAnnotations__Setup__StoresDashboardTitleMetadata(t *testing.T) {
	component := &ListAnnotations{}
	metadata := &contexts.MetadataContext{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"dashboard": {"title": "Operations", "uid": "dash-1"}
				}`)),
			},
		},
	}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"dashboardUID": "dash-1",
		},
		Metadata: metadata,
		HTTP:     httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, AnnotationNodeMetadata{DashboardTitle: "Operations"}, metadata.Metadata)
}

func Test__ListAnnotations__Setup__FallsBackToSearchWhenDashboardResponseIsTooLarge(t *testing.T) {
	component := &ListAnnotations{}
	metadata := &contexts.MetadataContext{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("a"), maxResponseSize+1))),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"uid":"dash-1","title":"Operations","type":"dash-db"}
				]`)),
			},
		},
	}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"dashboardUID": "dash-1",
		},
		Metadata: metadata,
		HTTP:     httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, AnnotationNodeMetadata{DashboardTitle: "Operations"}, metadata.Metadata)
	require.Len(t, httpContext.Requests, 2)
	require.Contains(t, httpContext.Requests[0].URL.Path, "/api/dashboards/uid/dash-1")
	require.Contains(t, httpContext.Requests[1].URL.Path, "/api/search")
}

func Test__filterAnnotations(t *testing.T) {
	panelID := int64(7)

	annotations := []Annotation{
		{ID: 1, PanelID: 7, Text: "Deploy completed"},
		{ID: 2, PanelID: 9, Text: "Deploy queued"},
		{ID: 3, PanelID: 7, Text: "Incident opened"},
	}

	filtered := filterAnnotations(annotations, &panelID, "deploy")
	require.Len(t, filtered, 1)
	require.Equal(t, int64(1), filtered[0].ID)

	filtered = filterAnnotations(annotations, nil, "INCIDENT")
	require.Len(t, filtered, 1)
	require.Equal(t, int64(3), filtered[0].ID)

	filtered = filterAnnotations(annotations, &panelID, "")
	require.Len(t, filtered, 2)
	require.Equal(t, int64(1), filtered[0].ID)
	require.Equal(t, int64(3), filtered[1].ID)
}

func Test__ListAnnotations__Execute__emitsResolvedTimeRange(t *testing.T) {
	component := &ListAnnotations{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":1,"text":"Deploy completed","time":0}
				]`)),
			},
		},
	}
	executionState := &contexts.ExecutionStateContext{}
	base := time.Now().UTC().Truncate(time.Second)
	from := base.Add(-time.Hour).Format(time.RFC3339)
	to := base.Format(time.RFC3339)

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"from": from,
			"to":   to,
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	require.Len(t, executionState.Payloads, 1)
	output := executionState.Payloads[0].(map[string]any)["data"].(ListAnnotationsOutput)
	require.Equal(t, base.Add(-time.Hour).UTC().Format(time.RFC3339Nano), output.From)
	require.Equal(t, base.UTC().Format(time.RFC3339Nano), output.To)
}
