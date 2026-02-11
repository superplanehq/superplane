package circleci

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__RunPipeline__buildParameters(t *testing.T) {
	t.Run("builds parameters map", func(t *testing.T) {
		tp := &RunPipeline{}
		params := []Parameter{
			{Name: "env", Value: "production"},
			{Name: "version", Value: "1.0.0"},
		}

		result := tp.buildParameters(params)

		assert.Equal(t, "production", result["env"])
		assert.Equal(t, "1.0.0", result["version"])
		assert.Len(t, result, 2)
	})
}
