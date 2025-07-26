package executors

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__SpecBuilder_Build(t *testing.T) {
	builder := SpecBuilder{}

	t.Run("fields in spec are resolved", func(t *testing.T) {
		specData, err := json.Marshal(map[string]any{
			"branch":       "main",
			"pipelineFile": ".semaphore/run.yml",
			"parameters": map[string]any{
				"PARAM_1": "${{ inputs.VAR_1 }}",
				"PARAM_2": "${{ secrets.TOKEN }}",
			},
		})

		require.NoError(t, err)

		d, err := builder.Build(specData, map[string]any{"VAR_1": "hello"}, map[string]string{"TOKEN": "token"})
		require.NoError(t, err)

		var resolvedSpec map[string]any
		err = json.Unmarshal(d, &resolvedSpec)
		require.NoError(t, err)
		assert.Equal(t, resolvedSpec["branch"], "main")
		assert.Equal(t, resolvedSpec["pipelineFile"], ".semaphore/run.yml")
		assert.Equal(t, map[string]any{"PARAM_1": "hello", "PARAM_2": "token"}, resolvedSpec["parameters"])
	})
}

func Test__SpecBuilder_ResolveExpression(t *testing.T) {
	t.Run("no expression", func(t *testing.T) {
		builder := SpecBuilder{}
		v, err := builder.ResolveExpression("hello", map[string]any{}, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "hello", v.(string))
	})

	t.Run("expression with input that exists", func(t *testing.T) {
		builder := SpecBuilder{}
		v, err := builder.ResolveExpression("${{ inputs.VAR_1 }}", map[string]any{"VAR_1": "hello"}, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "hello", v.(string))
	})

	t.Run("expression with input that does not exist", func(t *testing.T) {
		builder := SpecBuilder{}
		_, err := builder.ResolveExpression("${{ inputs.VAR_2 }}", map[string]any{"VAR_1": "hello"}, map[string]string{})
		require.ErrorContains(t, err, "input VAR_2 not found")
	})

	t.Run("expression with secret", func(t *testing.T) {
		builder := SpecBuilder{}
		v, err := builder.ResolveExpression("${{ secrets.SECRET_1 }}", map[string]any{}, map[string]string{"SECRET_1": "sensitive-value"})
		require.NoError(t, err)
		assert.Equal(t, "sensitive-value", v.(string))
	})

	t.Run("expression with raw value and bracket syntax", func(t *testing.T) {
		builder := SpecBuilder{}
		v, err := builder.ResolveExpression("Hello, ${{ inputs.NAME }}", map[string]any{"NAME": "joe"}, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "Hello, joe", v.(string))
	})

	t.Run("expression with raw value and bracket syntax with input that does not exist", func(t *testing.T) {
		builder := SpecBuilder{}
		_, err := builder.ResolveExpression("Hello, ${{ inputs.NAMEE }}", map[string]any{}, map[string]string{})
		require.ErrorContains(t, err, "input NAMEE not found")
	})

	t.Run("expression with raw value and double bracket syntax", func(t *testing.T) {
		builder := SpecBuilder{}
		v, err := builder.ResolveExpression(
			"Hello, ${{ inputs.NAME }} ${{ inputs.SURNAME }}",
			map[string]any{"NAME": "joe", "SURNAME": "doe"},
			map[string]string{},
		)

		require.NoError(t, err)
		assert.Equal(t, "Hello, joe doe", v.(string))
	})

	t.Run("expression with secret that does not exist", func(t *testing.T) {
		builder := SpecBuilder{}
		_, err := builder.ResolveExpression("${{ secrets.SECRET_2 }}", map[string]any{}, map[string]string{})
		require.ErrorContains(t, err, "secret SECRET_2 not found")
	})
}
