package executors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__SpecBuilder_Build(t *testing.T) {

	t.Run("semaphore spec", func(t *testing.T) {
		builder := SpecBuilder{}
		spec := models.ExecutorSpec{
			Type: models.ExecutorSpecTypeSemaphore,
			Semaphore: &models.SemaphoreExecutorSpec{
				OrganizationURL: "http://localhost:8000",
				APIToken:        "${{ secrets.TOKEN }}",
				ProjectID:       "demo-project",
				TaskID:          "demo-task",
				Branch:          "main",
				PipelineFile:    ".semaphore/run.yml",
				Parameters: map[string]string{
					"PARAM_1": "${{ inputs.VAR_1 }}",
				},
			},
		}

		v, err := builder.Build(spec, map[string]any{"VAR_1": "hello"}, map[string]string{"TOKEN": "token"})
		require.NoError(t, err)
		assert.Equal(t, v.Type, models.ExecutorSpecTypeSemaphore)
		assert.Equal(t, v.Semaphore.OrganizationURL, "http://localhost:8000")
		assert.Equal(t, v.Semaphore.APIToken, "token")
		assert.Equal(t, v.Semaphore.ProjectID, "demo-project")
		assert.Equal(t, v.Semaphore.TaskID, "demo-task")
		assert.Equal(t, v.Semaphore.Branch, "main")
		assert.Equal(t, v.Semaphore.PipelineFile, ".semaphore/run.yml")
		assert.Equal(t, map[string]string{"PARAM_1": "hello"}, v.Semaphore.Parameters)
	})

	t.Run("http spec", func(t *testing.T) {
		builder := SpecBuilder{}
		spec := models.ExecutorSpec{
			Type: models.ExecutorSpecTypeHTTP,
			HTTP: &models.HTTPExecutorSpec{
				URL: "http://localhost:8000",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"X-Param-A":     "${{ inputs.VAR_A }}",
					"Authorization": "Bearer ${{ secrets.TOKEN }}",
				},
				Payload: map[string]string{
					"b": "${{ inputs.VAR_B }}",
					"c": "static",
				},
				ResponsePolicy: &models.HTTPResponsePolicy{
					StatusCodes: []uint32{200, 201},
				},
			},
		}

		v, err := builder.Build(spec, map[string]any{"VAR_A": "hello", "VAR_B": "hi"}, map[string]string{"TOKEN": "mytoken"})
		require.NoError(t, err)
		assert.Equal(t, v.Type, models.ExecutorSpecTypeHTTP)
		assert.Equal(t, v.HTTP.URL, "http://localhost:8000")
		assert.Equal(t, v.HTTP.Headers, map[string]string{
			"Content-Type":  "application/json",
			"X-Param-A":     "hello",
			"Authorization": "Bearer mytoken",
		})
		assert.Equal(t, v.HTTP.Payload, map[string]string{
			"b": "hi",
			"c": "static",
		})
		assert.Equal(t, v.HTTP.ResponsePolicy.StatusCodes, []uint32{200, 201})
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
