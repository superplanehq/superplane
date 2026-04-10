package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetDataSource__Setup(t *testing.T) {
	component := GetDataSource{}

	t.Run("data source uid is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSourceUid": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "dataSourceUid is required")
	})

	t.Run("stores data source metadata", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": 7,
						"uid": "loki-main",
						"name": "Main Loki",
						"type": "loki",
						"url": "https://grafana.example.com/loki"
					}`)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSourceUid": "loki-main",
			},
			HTTP:     httpContext,
			Metadata: metadata,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, GetDataSourceNodeMetadata{
			DataSourceUID:  "loki-main",
			DataSourceName: "Main Loki",
			DataSourceType: "loki",
		}, metadata.Get())
	})
}
