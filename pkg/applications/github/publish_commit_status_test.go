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

func Test__PublishCommitStatus__Configuration(t *testing.T) {
	component := PublishCommitStatus{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		fieldNames := make(map[string]bool)
		for _, field := range config {
			fieldNames[field.Name] = field.Required
		}

		assert.True(t, fieldNames["repository"], "repository should be required")
		assert.True(t, fieldNames["sha"], "sha should be required")
		assert.True(t, fieldNames["state"], "state should be required")
		assert.True(t, fieldNames["context"], "context should be required")
		assert.False(t, fieldNames["description"], "description should be optional")
		assert.False(t, fieldNames["targetUrl"], "targetUrl should be optional")
	})

	t.Run("state has correct options", func(t *testing.T) {
		var foundStateField bool

		for _, field := range config {
			if field.Name == "state" {
				foundStateField = true
				assert.Equal(t, "select", field.Type)
				assert.NotNil(t, field.TypeOptions)
				assert.NotNil(t, field.TypeOptions.Select)

				options := field.TypeOptions.Select.Options
				assert.Len(t, options, 4, "should have 4 state options")

				values := make([]string, len(options))
				for i, opt := range options {
					values[i] = opt.Value
				}

				assert.Contains(t, values, "pending")
				assert.Contains(t, values, "success")
				assert.Contains(t, values, "failure")
				assert.Contains(t, values, "error")
				break
			}
		}

		assert.True(t, foundStateField, "state field should exist in configuration")
	})
}
