package fleets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstRunManagerYAML(t *testing.T) {
	t.Setenv("APP_ENV", "")
	yaml := FirstRunManagerYAML("https://superplane.example/", "token-value")
	assert.Contains(t, yaml, "superplaneUrl: https://superplane.example")
	assert.Contains(t, yaml, "installationAdminToken: token-value")
	assert.Contains(t, yaml, "id: e1-large-amd64")
	assert.Contains(t, yaml, "<PACKER_IMAGE_ID>")
	assert.Contains(t, yaml, "<RUNNER_RELEASE_BASE_URL>")
	assert.Equal(t, "e1-large-amd64", FirstRunFleetID())
}

func TestFirstRunManagerYAMLOmitsSelfHostPlaceholdersInDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	yaml := FirstRunManagerYAML("http://app:8000", "token-value")
	assert.Contains(t, yaml, "id: local")
	assert.Contains(t, yaml, "provider: docker")
	assert.Contains(t, yaml, "id: e1-large-amd64")
	assert.NotContains(t, yaml, "installationAdminToken")
	assert.NotContains(t, yaml, "<PACKER_IMAGE_ID>")
	assert.NotContains(t, yaml, "token-value")
}
