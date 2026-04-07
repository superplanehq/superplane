package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__GetSilence__Configuration__silenceIdIsIntegrationResource(t *testing.T) {
	component := GetSilence{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	require.Equal(t, "silenceId", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeSilence, fields[0].TypeOptions.Resource.Type)
}

func Test__DeleteSilence__Configuration__silenceIdIsIntegrationResource(t *testing.T) {
	component := DeleteSilence{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	require.Equal(t, "silenceId", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeSilence, fields[0].TypeOptions.Resource.Type)
}
