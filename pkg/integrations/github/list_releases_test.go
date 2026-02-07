package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListReleases__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := ListReleases{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}

func Test__ListReleases__Execute__Validation(t *testing.T) {
	component := ListReleases{}
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}

	integrationCtx := &contexts.IntegrationContext{
		Metadata: Metadata{
			InstallationID: "12345",
			Owner:          "testhq",
			Repositories:   []Repository{helloRepo},
			GitHubApp:      GitHubAppMetadata{ID: 12345},
		},
		Configuration: map[string]any{"privateKey": "test-key"},
	}

	executeWithConfig := func(t *testing.T, config map[string]any) error {
		t.Helper()

		return component.Execute(core.ExecutionContext{
			Integration:    integrationCtx,
			NodeMetadata:   &contexts.MetadataContext{Metadata: NodeMetadata{Repository: &helloRepo}},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  config,
		})
	}

	t.Run("perPage validation", func(t *testing.T) {
		testCases := []struct {
			name          string
			perPageValue  string
			expectedError string
		}{
			{
				name:          "non-numeric",
				perPageValue:  "abc",
				expectedError: "invalid perPage value 'abc': must be a valid number",
			},
			{
				name:          "zero",
				perPageValue:  "0",
				expectedError: "perPage must be greater than 0",
			},
			{
				name:          "negative",
				perPageValue:  "-5",
				expectedError: "perPage must be greater than 0",
			},
			{
				name:          "too large",
				perPageValue:  "500",
				expectedError: "perPage must be <= 100",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				err := executeWithConfig(t, map[string]any{
					"repository": "hello",
					"perPage":    testCase.perPageValue,
				})

				require.ErrorContains(t, err, testCase.expectedError)
			})
		}
	})

	t.Run("page validation", func(t *testing.T) {
		testCases := []struct {
			name          string
			pageValue     string
			expectedError string
		}{
			{
				name:          "non-numeric",
				pageValue:     "abc",
				expectedError: "invalid page value 'abc': must be a valid number",
			},
			{
				name:          "zero",
				pageValue:     "0",
				expectedError: "page must be greater than 0",
			},
			{
				name:          "negative",
				pageValue:     "-2",
				expectedError: "page must be greater than 0",
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				err := executeWithConfig(t, map[string]any{
					"repository": "hello",
					"page":       testCase.pageValue,
				})

				require.ErrorContains(t, err, testCase.expectedError)
			})
		}
	})
}
