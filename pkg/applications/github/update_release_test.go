package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateRelease__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := UpdateRelease{}

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

func Test__UpdateRelease__Configuration(t *testing.T) {
	component := UpdateRelease{}
	config := component.Configuration()

	t.Run("has required fields", func(t *testing.T) {
		fieldNames := make(map[string]bool)
		for _, field := range config {
			fieldNames[field.Name] = field.Required
		}

		assert.True(t, fieldNames["repository"], "repository should be required")
		assert.True(t, fieldNames["releaseStrategy"], "releaseStrategy should be required")
		assert.False(t, fieldNames["tagName"], "tagName should be optional (conditionally required)")
		assert.False(t, fieldNames["name"], "name should be optional")
		assert.False(t, fieldNames["body"], "body should be optional")
		assert.False(t, fieldNames["draft"], "draft should be optional")
		assert.False(t, fieldNames["prerelease"], "prerelease should be optional")
		assert.False(t, fieldNames["generateReleaseNotes"], "generateReleaseNotes should be optional")
	})

	t.Run("has correct field types", func(t *testing.T) {
		fieldTypes := make(map[string]string)
		for _, field := range config {
			fieldTypes[field.Name] = field.Type
		}

		assert.Equal(t, "string", fieldTypes["repository"])
		assert.Equal(t, "select", fieldTypes["releaseStrategy"])
		assert.Equal(t, "string", fieldTypes["tagName"])
		assert.Equal(t, "string", fieldTypes["name"])
		assert.Equal(t, "text", fieldTypes["body"])
		assert.Equal(t, "boolean", fieldTypes["draft"])
		assert.Equal(t, "boolean", fieldTypes["prerelease"])
		assert.Equal(t, "boolean", fieldTypes["generateReleaseNotes"])
	})

	t.Run("releaseStrategy has correct options", func(t *testing.T) {
		var foundField bool

		for _, field := range config {
			if field.Name == "releaseStrategy" {
				foundField = true
				assert.Equal(t, "select", field.Type)
				assert.NotNil(t, field.TypeOptions)
				assert.NotNil(t, field.TypeOptions.Select)

				options := field.TypeOptions.Select.Options
				assert.Len(t, options, 4, "should have 4 release strategy options")

				values := make([]string, len(options))
				for i, opt := range options {
					values[i] = opt.Value
				}

				assert.Contains(t, values, "specific")
				assert.Contains(t, values, "latest")
				assert.Contains(t, values, "latestDraft")
				assert.Contains(t, values, "latestPrerelease")
				break
			}
		}

		assert.True(t, foundField, "releaseStrategy field should exist in configuration")
	})

	t.Run("tagName has visibility conditions", func(t *testing.T) {
		var foundField bool

		for _, field := range config {
			if field.Name == "tagName" {
				foundField = true
				assert.Len(t, field.VisibilityConditions, 1, "should have 1 visibility condition")
				assert.Equal(t, "releaseStrategy", field.VisibilityConditions[0].Field)
				assert.Equal(t, []string{"specific"}, field.VisibilityConditions[0].Values)

				assert.Len(t, field.RequiredConditions, 1, "should have 1 required condition")
				assert.Equal(t, "releaseStrategy", field.RequiredConditions[0].Field)
				assert.Equal(t, []string{"specific"}, field.RequiredConditions[0].Values)
				break
			}
		}

		assert.True(t, foundField, "tagName field should exist in configuration")
	})

	t.Run("has correct placeholders and descriptions", func(t *testing.T) {
		for _, field := range config {
			switch field.Name {
			case "tagName":
				assert.Contains(t, field.Placeholder, "v1.0.0")
				assert.Contains(t, field.Placeholder, "event.data.release.tag_name")
				assert.Contains(t, field.Description, "template variables")
			case "name":
				assert.Contains(t, field.Description, "leave empty to keep current")
			case "body":
				assert.Contains(t, field.Description, "leave empty to keep current")
			case "generateReleaseNotes":
				assert.Contains(t, field.Description, "generate")
			}
		}
	})
}

func Test__UpdateRelease__Component_Interface(t *testing.T) {
	component := UpdateRelease{}

	t.Run("Name returns correct value", func(t *testing.T) {
		assert.Equal(t, "github.updateRelease", component.Name())
	})

	t.Run("Label returns correct value", func(t *testing.T) {
		assert.Equal(t, "Update Release", component.Label())
	})

	t.Run("Description returns correct value", func(t *testing.T) {
		assert.Equal(t, "Update an existing release in a GitHub repository", component.Description())
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
