package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__InstanceComponents__ConfigurationUsesInstanceResource(t *testing.T) {
	components := []struct {
		name   string
		fields []configuration.Field
	}{
		{name: "get", fields: (&GetInstance{}).Configuration()},
		{name: "update", fields: (&UpdateInstance{}).Configuration()},
		{name: "power", fields: (&ManageInstancePower{}).Configuration()},
		{name: "delete", fields: (&DeleteInstance{}).Configuration()},
	}

	for _, component := range components {
		t.Run(component.name, func(t *testing.T) {
			require.NotEmpty(t, component.fields)
			field := component.fields[0]

			assert.Equal(t, "instanceId", field.Name)
			assert.Equal(t, "Instance", field.Label)
			assert.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
			require.NotNil(t, field.TypeOptions)
			require.NotNil(t, field.TypeOptions.Resource)
			assert.Equal(t, ResourceTypeInstance, field.TypeOptions.Resource.Type)
		})
	}
}
