package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateRelease__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreateRelease{}

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

func Test__CreateRelease__Configuration(t *testing.T) {
	component := CreateRelease{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		fieldNames := make(map[string]bool)
		for _, field := range config {
			fieldNames[field.Name] = field.Required
		}

		assert.True(t, fieldNames["repository"], "repository should be required")
		assert.True(t, fieldNames["versionStrategy"], "versionStrategy should be required")
		assert.False(t, fieldNames["tagName"], "tagName should be optional (conditionally required)")
		assert.False(t, fieldNames["name"], "name should be optional")
		assert.False(t, fieldNames["draft"], "draft should be optional")
		assert.False(t, fieldNames["prerelease"], "prerelease should be optional")
		assert.False(t, fieldNames["generateReleaseNotes"], "generateReleaseNotes should be optional")
		assert.False(t, fieldNames["body"], "body should be optional")
	})

	t.Run("versionStrategy has correct options", func(t *testing.T) {
		var foundField bool

		for _, field := range config {
			if field.Name == "versionStrategy" {
				foundField = true
				assert.Equal(t, "select", field.Type)
				assert.NotNil(t, field.TypeOptions)
				assert.NotNil(t, field.TypeOptions.Select)

				options := field.TypeOptions.Select.Options
				assert.Len(t, options, 4, "should have 4 version strategy options")

				values := make([]string, len(options))
				for i, opt := range options {
					values[i] = opt.Value
				}

				assert.Contains(t, values, "manual")
				assert.Contains(t, values, "patch")
				assert.Contains(t, values, "minor")
				assert.Contains(t, values, "major")
				break
			}
		}

		assert.True(t, foundField, "versionStrategy field should exist in configuration")
	})

	t.Run("tagName has visibility conditions", func(t *testing.T) {
		var foundField bool

		for _, field := range config {
			if field.Name == "tagName" {
				foundField = true
				assert.Len(t, field.VisibilityConditions, 1, "should have 1 visibility condition")
				assert.Equal(t, "versionStrategy", field.VisibilityConditions[0].Field)
				assert.Equal(t, []string{"manual"}, field.VisibilityConditions[0].Values)

				assert.Len(t, field.RequiredConditions, 1, "should have 1 required condition")
				assert.Equal(t, "versionStrategy", field.RequiredConditions[0].Field)
				assert.Equal(t, []string{"manual"}, field.RequiredConditions[0].Values)
				break
			}
		}

		assert.True(t, foundField, "tagName field should exist in configuration")
	})
}

func Test__CreateRelease__IncrementVersion(t *testing.T) {
	component := CreateRelease{}

	t.Run("patch increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "1.2.4"},
			{"with v prefix", "v1.2.3", "v1.2.4"},
			{"with version prefix", "version-1.2.3", "version-1.2.4"},
			{"patch rollover", "1.2.9", "1.2.10"},
			{"double digit patch", "1.2.99", "1.2.100"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "patch")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("minor increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "1.3.0"},
			{"with v prefix", "v1.2.3", "v1.3.0"},
			{"with version prefix", "version-1.2.3", "version-1.3.0"},
			{"minor rollover", "1.9.3", "1.10.0"},
			{"resets patch", "1.2.99", "1.3.0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "minor")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("major increment works correctly", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    string
			expected string
		}{
			{"simple version", "1.2.3", "2.0.0"},
			{"with v prefix", "v1.2.3", "v2.0.0"},
			{"with version prefix", "version-1.2.3", "version-2.0.0"},
			{"major rollover", "9.2.3", "10.0.0"},
			{"resets minor and patch", "1.99.99", "2.0.0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, err := component.incrementVersion(tc.input, "major")
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("invalid version format returns error", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"no dots", "123"},
			{"only major.minor", "1.2"},
			{"non-numeric", "v1.2.x"},
			{"empty string", ""},
			{"only prefix", "v"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := component.incrementVersion(tc.version, "patch")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid version format")
			})
		}
	})

	t.Run("version with extra parts extracts first three", func(t *testing.T) {
		// The regex extracts the first major.minor.patch it finds
		result, err := component.incrementVersion("1.2.3.4", "patch")
		require.NoError(t, err)
		assert.Equal(t, "1.2.4", result)
	})

	t.Run("invalid strategy returns error", func(t *testing.T) {
		_, err := component.incrementVersion("v1.2.3", "invalid")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid version strategy")
	})
}

func Test__CreateRelease__SemverRegex(t *testing.T) {
	t.Run("valid semantic versions match", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"simple version", "1.2.3"},
			{"with v prefix", "v1.2.3"},
			{"with version prefix", "version-1.2.3"},
			{"double digits", "10.20.30"},
			{"triple digits", "100.200.300"},
			{"with release prefix", "release-1.2.3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.True(t, semverRegex.MatchString(tc.version))
			})
		}
	})

	t.Run("invalid formats do not match", func(t *testing.T) {
		testCases := []struct {
			name    string
			version string
		}{
			{"no dots", "123"},
			{"only major.minor", "1.2"},
			{"non-numeric major", "vx.2.3"},
			{"non-numeric minor", "v1.x.3"},
			{"non-numeric patch", "v1.2.x"},
			{"empty string", ""},
			{"only prefix", "v"},
			{"spaces", "1. 2.3"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				assert.False(t, semverRegex.MatchString(tc.version))
			})
		}
	})

	t.Run("versions with extra parts match first three", func(t *testing.T) {
		// The regex extracts the first major.minor.patch, ignoring extra parts
		assert.True(t, semverRegex.MatchString("1.2.3.4"))
		assert.True(t, semverRegex.MatchString("v10.20.30.40"))
	})
}

func Test__CreateRelease__Component_Interface(t *testing.T) {
	component := CreateRelease{}

	t.Run("Name returns correct value", func(t *testing.T) {
		assert.Equal(t, "github.createRelease", component.Name())
	})

	t.Run("Label returns correct value", func(t *testing.T) {
		assert.Equal(t, "Create Release", component.Label())
	})

	t.Run("Description returns correct value", func(t *testing.T) {
		assert.Equal(t, "Create a new release in a GitHub repository", component.Description())
	})

	t.Run("Icon returns correct value", func(t *testing.T) {
		assert.Equal(t, "github", component.Icon())
	})

	t.Run("Color returns correct value", func(t *testing.T) {
		assert.Equal(t, "gray", component.Color())
	})

	t.Run("OutputChannels returns default channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		assert.Len(t, channels, 1)
		assert.Equal(t, core.DefaultOutputChannel, channels[0])
	})

	t.Run("Actions returns empty slice", func(t *testing.T) {
		actions := component.Actions()
		assert.Empty(t, actions)
	})
}
