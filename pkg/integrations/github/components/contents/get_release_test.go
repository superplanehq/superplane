package contents

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetRelease__Execute__Validation(t *testing.T) {
	component := GetRelease{}
	helloRepo := common.Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	t.Run("returns error when releaseId is nil for byId strategy", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				InstallationID: "12345",
				Owner:          "testhq",
				Repositories:   []common.Repository{helloRepo},
				GitHubApp:      common.GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: common.NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":      "hello",
				"releaseStrategy": "byId",
				"releaseId":       nil,
			},
		})

		require.ErrorContains(t, err, "release ID is required when using byId strategy")
	})

	t.Run("returns error when tagName is empty for specific strategy", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: common.Metadata{
				InstallationID: "12345",
				Owner:          "testhq",
				Repositories:   []common.Repository{helloRepo},
				GitHubApp:      common.GitHubAppMetadata{ID: 12345},
			},
			Configuration: map[string]any{
				"privateKey": "test-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: common.NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"repository":      "hello",
				"releaseStrategy": "specific",
				"tagName":         "",
			},
		})

		require.ErrorContains(t, err, "tag name is required when using specific tag strategy")
	})
}
