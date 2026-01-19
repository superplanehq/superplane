package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__PublishCommitStatus__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := PublishCommitStatus{}

	t.Run("repository is required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &contexts.MetadataContext{},
			Configuration:   map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set successfully", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, component.Setup(core.SetupContext{
			AppInstallation: appCtx,
			Metadata:        &nodeMetadataCtx,
			Configuration:   map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
	})
}

func Test__PublishCommitStatus__ValidateSHA(t *testing.T) {
	t.Run("valid 40-char hex SHA passes", func(t *testing.T) {
		validSHA := "abc123def456789012345678901234567890abcd"
		assert.True(t, shaRegex.MatchString(validSHA))
	})

	t.Run("invalid SHA formats fail", func(t *testing.T) {
		testCases := []struct {
			name string
			sha  string
		}{
			{"too short", "abc123"},
			{"too long", "abc123def456789012345678901234567890abcdef"},
			{"uppercase letters", "ABC123DEF456789012345678901234567890ABCD"},
			{"invalid characters", "xyz123def456789012345678901234567890abcd"},
			{"with spaces", "abc123 ef456789012345678901234567890abcd"},
			{"empty string", ""},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.False(t, shaRegex.MatchString(tc.sha))
			})
		}
	})
}
