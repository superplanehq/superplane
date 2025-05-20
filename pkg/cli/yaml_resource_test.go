package cli

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func Test__ParseYamlResourceHeaders__InvalidValidResource(t *testing.T) {
	resource := []byte(`
	        kind: Projec			t
apiVersio              n: v1alpha`)

	_, _, err := ParseYamlResourceHeaders(resource)

	assert.Equal(t, err.Error(), "failed to parse resource; error converting YAML to JSON: yaml: line 2: found character that cannot start any token")
}

func Test__ParseYamlResourceHeaders__ValidResource(t *testing.T) {
	resource := []byte(`
kind: Project
apiVersion: v1alpha`)

	apiVersion, kind, err := ParseYamlResourceHeaders(resource)

	assert.Nil(t, err)
	assert.Equal(t, kind, "Project")
	assert.Equal(t, apiVersion, "v1alpha")
}

func Test__ParseYamlResourceHeaders__KindMissing(t *testing.T) {
	resource := []byte(`apiVersion: v1alpha`)

	_, _, err := ParseYamlResourceHeaders(resource)

	assert.Equal(t, err.Error(), "failed to parse resource's kind")
}

func Test__ParseYamlResourceHeaders__ApiVersionMissing(t *testing.T) {
	resource := []byte(`kind: Project`)

	_, _, err := ParseYamlResourceHeaders(resource)

	assert.Equal(t, err.Error(), "failed to parse resource's api version")
}

func Test__ParseYamlResourceHeaders__KindIsWrongType(t *testing.T) {
	resource := []byte(`
kind:
  test: Project
apiVersion: v1alpha`)

	_, _, err := ParseYamlResourceHeaders(resource)

	assert.Equal(t, err.Error(), "failed to parse resource's kind")
}

func Test__ParseYamlResourceHeaders__ApiVersionWrongType(t *testing.T) {
	resource := []byte(`
kind: Project
apiVersion:
  test: v1alpha`)

	_, _, err := ParseYamlResourceHeaders(resource)

	assert.Equal(t, err.Error(), "failed to parse resource's api version")
}
