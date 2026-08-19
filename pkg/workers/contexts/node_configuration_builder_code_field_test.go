package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func boolPtr(v bool) *bool {
	return &v
}

func textField(name, language string, allowExpressions *bool) configuration.Field {
	return configuration.Field{
		Name: name,
		Type: configuration.FieldTypeText,
		TypeOptions: &configuration.TypeOptions{
			Text: &configuration.TextTypeOptions{
				Language:         language,
				AllowExpressions: allowExpressions,
			},
		},
	}
}

// Regression test: text fields with AllowExpressions=false leave {{ ... }}
// placeholders as literal text instead of resolving them.
func Test_NodeConfigurationBuilder_TextField_ExpressionsNotResolvedWhenDisabled(t *testing.T) {
	for _, language := range []string{"javascript", "python", "shell"} {
		t.Run(language, func(t *testing.T) {
			builder := NewNodeConfigurationBuilder(nil, uuid.New()).
				WithRootPayload(map[string]any{
					"timestamp": "2026-07-02T08:53:06.174546542Z",
				}).
				WithConfigurationFields([]configuration.Field{
					textField("script", language, boolPtr(false)),
				})

			result, err := builder.Build(map[string]any{
				"script": "console.log({{ root().timestamp }});",
			})

			require.NoError(t, err)
			assert.Equal(t, "console.log({{ root().timestamp }});", result["script"])
		})
	}
}

func Test_NodeConfigurationBuilder_TextField_ExpressionsResolvedByDefault(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"name": "john",
		})

	fields := []configuration.Field{
		{Name: "message", Type: configuration.FieldTypeText},
		textField("body", "json", nil),
		textField("script", "javascript", nil),
	}

	result, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"message": "Hello {{ root().name }}",
		"body":    `{"name": "{{ root().name }}"}`,
		"script":  "const name = {{ root().name }};",
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello john", result["message"])
	assert.Equal(t, `{"name": "john"}`, result["body"])
	assert.Equal(t, "const name = john;", result["script"])
}
