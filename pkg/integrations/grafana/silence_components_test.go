package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
)

func Test__GetSilence__Configuration__silenceIsIntegrationResource(t *testing.T) {
	component := GetSilence{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	require.Equal(t, "silence", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeSilence, fields[0].TypeOptions.Resource.Type)
}

func Test__DeleteSilence__Configuration__silenceIsIntegrationResource(t *testing.T) {
	component := DeleteSilence{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	require.Equal(t, "silence", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeSilence, fields[0].TypeOptions.Resource.Type)
}

func Test__GetSilence__decodeGetSilenceSpec__acceptsLegacySilenceId(t *testing.T) {
	spec, err := decodeGetSilenceSpec(map[string]any{
		"silenceId": "silence-123",
	})

	require.NoError(t, err)
	require.Equal(t, "silence-123", spec.Silence)
}

func Test__DeleteSilence__decodeDeleteSilenceSpec__acceptsLegacySilenceId(t *testing.T) {
	spec, err := decodeDeleteSilenceSpec(map[string]any{
		"silenceId": "silence-123",
	})

	require.NoError(t, err)
	require.Equal(t, "silence-123", spec.Silence)
}
