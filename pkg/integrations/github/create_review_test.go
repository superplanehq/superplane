package github

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func TestCreateReview_Name(t *testing.T) {
	c := &CreateReview{}
	if got := c.Name(); got != "github.createReview" {
		t.Errorf("Name() = %v, want %v", got, "github.createReview")
	}
}

func TestCreateReview_Label(t *testing.T) {
	c := &CreateReview{}
	if got := c.Label(); got != "Create Review" {
		t.Errorf("Label() = %v, want %v", got, "Create Review")
	}
}

func TestCreateReview_Description(t *testing.T) {
	c := &CreateReview{}
	if got := c.Description(); got != "Create a review on a GitHub pull request" {
		t.Errorf("Description() = %v, want %v", got, "Create a review on a GitHub pull request")
	}
}

func TestCreateReview_Icon(t *testing.T) {
	c := &CreateReview{}
	if got := c.Icon(); got != "github" {
		t.Errorf("Icon() = %v, want %v", got, "github")
	}
}

func TestCreateReview_Color(t *testing.T) {
	c := &CreateReview{}
	if got := c.Color(); got != "gray" {
		t.Errorf("Color() = %v, want %v", got, "gray")
	}
}

func TestCreateReview_Configuration(t *testing.T) {
	c := &CreateReview{}
	config := c.Configuration()

	if len(config) != 6 {
		t.Errorf("Configuration() returned %d fields, want 6", len(config))
	}

	fieldNames := make(map[string]bool)
	for _, field := range config {
		fieldNames[field.Name] = true
	}

	requiredFields := []string{"repository", "pullRequestNumber", "event", "body", "commitId", "comments"}
	for _, fieldName := range requiredFields {
		if !fieldNames[fieldName] {
			t.Errorf("Configuration() missing '%s' field", fieldName)
		}
	}
}

func TestCreateReview_Configuration_EventOptions(t *testing.T) {
	c := &CreateReview{}
	config := c.Configuration()

	var eventField *struct {
		Name        string
		TypeOptions interface{}
	}

	for _, field := range config {
		if field.Name == "event" {
			if field.TypeOptions == nil || field.TypeOptions.Select == nil {
				t.Error("event field should have Select type options")
				return
			}

			options := field.TypeOptions.Select.Options
			expectedOptions := map[string]string{
				"APPROVE":         "Approve",
				"REQUEST_CHANGES": "Request Changes",
				"COMMENT":         "Comment",
				"PENDING":         "Pending",
			}

			if len(options) != len(expectedOptions) {
				t.Errorf("event field has %d options, want %d", len(options), len(expectedOptions))
			}

			for _, opt := range options {
				if expectedLabel, ok := expectedOptions[opt.Value]; ok {
					if opt.Label != expectedLabel {
						t.Errorf("event option %s has label %s, want %s", opt.Value, opt.Label, expectedLabel)
					}
				} else {
					t.Errorf("unexpected event option value: %s", opt.Value)
				}
			}
			return
		}
	}

	if eventField == nil {
		t.Error("event field not found in configuration")
	}
}

func TestCreateReview_Configuration_CommentsSchema(t *testing.T) {
	c := &CreateReview{}
	config := c.Configuration()

	for _, field := range config {
		if field.Name == "comments" {
			if field.TypeOptions == nil || field.TypeOptions.List == nil {
				t.Error("comments field should have List type options")
				return
			}

			listOptions := field.TypeOptions.List
			if listOptions.ItemDefinition == nil {
				t.Error("comments field should have ItemDefinition")
				return
			}

			schema := listOptions.ItemDefinition.Schema
			expectedFields := []string{"path", "body", "line", "side", "startLine", "startSide"}

			if len(schema) != len(expectedFields) {
				t.Errorf("comments schema has %d fields, want %d", len(schema), len(expectedFields))
			}

			schemaFieldNames := make(map[string]bool)
			for _, f := range schema {
				schemaFieldNames[f.Name] = true
			}

			for _, fieldName := range expectedFields {
				if !schemaFieldNames[fieldName] {
					t.Errorf("comments schema missing '%s' field", fieldName)
				}
			}
			return
		}
	}

	t.Error("comments field not found in configuration")
}

func TestCreateReview_Documentation(t *testing.T) {
	c := &CreateReview{}
	doc := c.Documentation()

	if doc == "" {
		t.Error("Documentation() should not be empty")
	}

	keywords := []string{"APPROVE", "REQUEST_CHANGES", "COMMENT", "pull request", "review"}
	for _, keyword := range keywords {
		found := false
		if len(doc) > 0 {
			for i := 0; i <= len(doc)-len(keyword); i++ {
				if doc[i:i+len(keyword)] == keyword {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Documentation() should mention '%s'", keyword)
		}
	}
}

func Test__CreateReview__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	component := CreateReview{}

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
