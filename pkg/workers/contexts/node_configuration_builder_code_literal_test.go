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

// Regression test for https://github.com/superplanehq/superplane/issues/5615.
func Test_NodeConfigurationBuilder_CodeField_JavaScriptBareStringGetsQuoted(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"timestamp": "2026-07-02T08:53:06.174546542Z",
		}).
		WithConfigurationFields([]configuration.Field{codeTextField("script", "javascript")})

	result, err := builder.Build(map[string]any{
		"script": "console.log({{ root().timestamp }});",
	})

	require.NoError(t, err)
	assert.Equal(t, `console.log("2026-07-02T08:53:06.174546542Z");`, result["script"])
}

func Test_NodeConfigurationBuilder_CodeField_JavaScriptScalarTypes(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"count":   42,
			"active":  true,
			"missing": nil,
			"nested":  map[string]any{"a": 1, "b": "x"},
		})

	fields := []configuration.Field{codeTextField("script", "javascript")}

	result, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"script": "const n = {{ root().count }}; const ok = {{ root().active }}; " +
			"const m = {{ root().missing }}; const obj = {{ root().nested }};",
	})

	require.NoError(t, err)
	assert.Equal(t,
		`const n = 42; const ok = true; const m = null; const obj = {"a":1,"b":"x"};`,
		result["script"],
	)
}

func Test_NodeConfigurationBuilder_CodeField_PythonLiterals(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"active":  true,
			"missing": nil,
			"nested":  map[string]any{"a": nil, "b": true},
		})

	fields := []configuration.Field{codeTextField("script", "python")}

	result, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"script": "ok = {{ root().active }}\nm = {{ root().missing }}\nobj = {{ root().nested }}",
	})

	require.NoError(t, err)
	assert.Equal(t, "ok = True\nm = None\nobj = {\"a\": None, \"b\": True}", result["script"])
}

func Test_NodeConfigurationBuilder_CodeField_ShellQuoting(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"message": "hello world",
		})

	fields := []configuration.Field{codeTextField("commands", "shell")}

	result, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"commands": "echo {{ root().message }}",
	})

	require.NoError(t, err)
	assert.Equal(t, `echo 'hello world'`, result["commands"])
}

func Test_NodeConfigurationBuilder_CodeField_ShellArithmeticExpansionStaysUnquoted(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{"count": 5}).
		WithConfigurationFields([]configuration.Field{codeTextField("commands", "shell")})

	result, err := builder.Build(map[string]any{
		"commands": "echo $(( {{ root().count }} + 1 ))",
	})

	require.NoError(t, err)
	assert.Equal(t, "echo $(( 5 + 1 ))", result["commands"])
}

func Test_NodeConfigurationBuilder_CodeField_PythonTripleQuotedPlaceholderIsNotReQuoted(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{"name": "john", "n": 5}).
		WithConfigurationFields([]configuration.Field{codeTextField("script", "python")})

	result, err := builder.Build(map[string]any{
		"script": "msg = \"\"\"He said \"hi\" to {{ root().name }}\"\"\"\nn = {{ root().n }}",
	})

	require.NoError(t, err)
	assert.Equal(t, "msg = \"\"\"He said \"hi\" to john\"\"\"\nn = 5", result["script"])
}

func Test_NodeConfigurationBuilder_CodeField_AlreadyQuotedPlaceholderIsNotDoubleQuoted(t *testing.T) {
	builder := NewNodeConfigurationBuilder(nil, uuid.New()).
		WithRootPayload(map[string]any{
			"timestamp": "2026-07-02T08:53:06Z",
		})

	fields := []configuration.Field{codeTextField("script", "javascript")}

	backtickResult, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"script": "console.log(`Time: {{ root().timestamp }}`);",
	})
	require.NoError(t, err)
	assert.Equal(t, "console.log(`Time: 2026-07-02T08:53:06Z`);", backtickResult["script"])

	doubleQuoteResult, err := builder.WithConfigurationFields(fields).Build(map[string]any{
		"script": "console.log(\"Time: {{ root().timestamp }}\");",
	})
	require.NoError(t, err)
	assert.Equal(t, "console.log(\"Time: 2026-07-02T08:53:06Z\");", doubleQuoteResult["script"])
}

func Test_NodeConfigurationBuilder_CodeField_NonCodeFieldsKeepRawSubstitution(t *testing.T) {
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
