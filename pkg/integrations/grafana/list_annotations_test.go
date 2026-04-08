package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

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

func Test__ListAnnotations__Setup__StoresDashboardTitleMetadata(t *testing.T) {
	component := &ListAnnotations{}
	metadata := &contexts.MetadataContext{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
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
}
