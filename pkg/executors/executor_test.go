package executors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ResolveExpression(t *testing.T) {
	t.Run("no expression", func(t *testing.T) {
		v, err := resolveExpression("hello", map[string]any{}, map[string]string{})
		require.NoError(t, err)
		assert.Equal(t, "hello", v.(string))
	})

	t.Run("expression with input that exists", func(t *testing.T) {
		v, err := resolveExpression("${{ inputs.VAR_1 }}", map[string]any{
			"VAR_1": "hello",
		}, map[string]string{})

		require.NoError(t, err)
		assert.Equal(t, "hello", v.(string))
	})

	t.Run("expression with input that does not exist", func(t *testing.T) {
		_, err := resolveExpression("${{ inputs.VAR_2 }}", map[string]any{
			"VAR_1": "hello",
		}, map[string]string{})

		require.ErrorContains(t, err, "input VAR_2 not found")
	})

	t.Run("expression with secret", func(t *testing.T) {
		v, err := resolveExpression("${{ secrets.SECRET_1 }}", map[string]any{}, map[string]string{
			"SECRET_1": "sensitive-value",
		})

		require.NoError(t, err)
		assert.Equal(t, "sensitive-value", v.(string))
	})

	t.Run("expression with raw value and bracket syntax", func(t *testing.T) {
		v, err := resolveExpression("Hello, ${{ inputs.NAME }}", map[string]any{
			"NAME": "joe",
		}, map[string]string{})

		require.NoError(t, err)
		assert.Equal(t, "Hello, joe", v.(string))
	})

	t.Run("expression with raw value and bracket syntax with input that does not exist", func(t *testing.T) {
		_, err := resolveExpression("Hello, ${{ inputs.NAMEE }}", map[string]any{
			"NAME": "joe",
		}, map[string]string{})

		require.ErrorContains(t, err, "input NAMEE not found")
	})

	t.Run("expression with raw value and double bracket syntax", func(t *testing.T) {
		v, err := resolveExpression("Hello, ${{ inputs.NAME }} ${{ inputs.SURNAME }}", map[string]any{
			"NAME":    "joe",
			"SURNAME": "doe",
		}, map[string]string{})

		require.NoError(t, err)
		assert.Equal(t, "Hello, joe doe", v.(string))
	})

	t.Run("expression with secret that does not exist", func(t *testing.T) {
		_, err := resolveExpression("${{ secrets.SECRET_2 }}", map[string]any{}, map[string]string{
			"SECRET_1": "sensitive-value",
		})

		require.ErrorContains(t, err, "secret SECRET_2 not found")
	})
}
