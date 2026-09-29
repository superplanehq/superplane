package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__intakeFilterExpressionFor_Linear(t *testing.T) {
	expression := intakeLinearFilterExpression(intakeSettings{
		LinearProjectIDs: []string{"project-1", "project-2"},
		LinearLabels:     []string{"bug"},
	})

	assert.Contains(t, expression, `root().data.data.projectId`)
	assert.Contains(t, expression, `"project-1"`)
	assert.Contains(t, expression, `"project-2"`)
	assert.Contains(t, expression, `any(root().data.data.labels, .name in ["bug"])`)
}

func Test__intakeFilterExpressionFor_Linear__Empty(t *testing.T) {
	assert.Equal(t, "true", intakeLinearFilterExpression(intakeSettings{}))
}
