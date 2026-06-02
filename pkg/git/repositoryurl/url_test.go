package repositoryurl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test__SuperplaneCloneURL(t *testing.T) {
	t.Setenv("BASE_URL", "https://app.superplane.com")

	canvasID := "550e8400-e29b-41d4-a716-446655440000"
	assert.Equal(t, "https://app.superplane.com/git/"+canvasID+".git", SuperplaneCloneURL(canvasID))
	assert.Empty(t, SuperplaneCloneURL("not-a-uuid"))
	assert.Empty(t, SuperplaneCloneURL("  "))
}

func Test__SuperplaneCloneURL__withoutBaseURL(t *testing.T) {
	t.Setenv("BASE_URL", "")
	t.Setenv("PORT", "8000")

	canvasID := "550e8400-e29b-41d4-a716-446655440000"
	assert.Equal(t, "http://localhost:8000/git/"+canvasID+".git", SuperplaneCloneURL(canvasID))
}
