package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__AddIncidentActivity__Configuration__incidentIsIntegrationResource(t *testing.T) {
	fields := (&AddIncidentActivity{}).Configuration()

	require.NotEmpty(t, fields)
	require.Equal(t, "incident", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncident, fields[0].TypeOptions.Resource.Type)
}
