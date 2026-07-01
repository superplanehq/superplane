package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__CreateComputeInstance__ConfigurationUsesIntegrationResources(t *testing.T) {
	fields := (&CreateComputeInstance{}).Configuration()

	assertResourceField(t, fields, "imageOs", ResourceTypeImageOS)
	assertResourceField(t, fields, "bootVolumeVpusPerGB", ResourceTypeBootVolumeVPU)
}

func assertResourceField(t *testing.T, fields []configuration.Field, name string, resourceType string) {
	t.Helper()

	for _, field := range fields {
		if field.Name != name {
			continue
		}

		require.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, resourceType, field.TypeOptions.Resource.Type)
		return
	}

	t.Fatalf("field %q not found", name)
}
