package contexts

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func codeTextField(name, language string) configuration.Field {
	return configuration.Field{
		Name: name,
		Type: configuration.FieldTypeText,
		TypeOptions: &configuration.TypeOptions{
			Text: &configuration.TextTypeOptions{
				Language: language,
			},
		},
	}
}

// Regression test: {{ ... }} expressions in code fields (runner scripts) are
// left untouched as literal text instead of being resolved, since naive
// substitution can produce syntactically invalid or unsafe code.
func Test_NodeConfigurationBuilder_CodeField_ExpressionsAreNotResolved(t *testing.T) {
	for _, language := range []string{"javascript", "python", "shell"} {
		t.Run(language, func(t *testing.T) {
			builder := NewNodeConfigurationBuilder(nil, uuid.New()).
				WithRootPayload(map[string]any{
					"timestamp": "2026-07-02T08:53:06.174546542Z",
				}).
				WithConfigurationFields([]configuration.Field{codeTextField("script", language)})

			result, err := builder.Build(map[string]any{
				"script": "console.log({{ root().timestamp }});",
			})

			require.NoError(t, err)
			assert.Equal(t, "console.log({{ root().timestamp }});", result["script"])
		})
	}
}

func Test_NodeConfigurationBuilder_NonCodeFieldsKeepResolvingExpressions(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"name": "john",
		})

	fields := []configuration.Field{
		{Name: "message", Type: configuration.FieldTypeText},
		codeTextField("body", "json"),
	}

	result, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"message": "Hello {{ root().name }}",
		"body":    `{"name": "{{ root().name }}"}`,
	})

	require.NoError(t, err)
	assert.Equal(t, "Hello john", result["message"])
	assert.Equal(t, `{"name": "john"}`, result["body"])
}
