package runner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestFormatFleetOptionLabel(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "aws-amd64 (t3.micro, amd64)", FormatFleetOptionLabel(BrokerFleet{
		ID: "aws-amd64", Size: "t3.micro", Arch: "amd64",
	}))
	assert.Equal(t, "pool-a (t4g.small)", FormatFleetOptionLabel(BrokerFleet{ID: "pool-a", Size: "t4g.small"}))
	assert.Equal(t, "pool-b", FormatFleetOptionLabel(BrokerFleet{ID: "pool-b"}))
}

func TestRequireMachineType(t *testing.T) {
	t.Parallel()
	got, err := requireMachineType("node-fleet")
	require.NoError(t, err)
	assert.Equal(t, "node-fleet", got)

	_, err = requireMachineType("")
	require.Error(t, err)
}

func TestListFleets(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"aws-standard-amd64","provisioner":"aws","arch":"amd64","size":"t3.micro"},
					{"id":"aws-standard-arm64","arch":"arm64","size":"t4g.micro"}
				]`)),
			},
		},
	}

	broker, err := NewBrokerClient(httpContext)
	require.NoError(t, err)

	fleets, err := broker.ListFleets()
	require.NoError(t, err)
	require.Len(t, fleets, 2)
	assert.Equal(t, "aws-standard-amd64", fleets[0].ID)
	assert.Equal(t, "t3.micro", fleets[0].Size)
}

func TestEnrichRunnerConfigurationFields(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"aws-standard-amd64","arch":"amd64","size":"t3.micro"},
					{"id":"aws-standard-arm64","arch":"arm64","size":"t4g.micro"}
				]`)),
			},
		},
	}

	fields := (&Runner{}).Configuration()
	enriched := EnrichRunnerConfigurationFields(httpContext, fields)

	var fleetField *configuration.Field
	for i := range enriched {
		if enriched[i].Name == configurationFieldMachineType {
			fleetField = &enriched[i]
			break
		}
	}
	require.NotNil(t, fleetField)
	require.NotNil(t, fleetField.TypeOptions)
	require.NotNil(t, fleetField.TypeOptions.Select)
	require.Len(t, fleetField.TypeOptions.Select.Options, 2)
	assert.Nil(t, fleetField.Default)
	assert.Equal(t, "aws-standard-amd64 (t3.micro, amd64)", fleetField.TypeOptions.Select.Options[0].Label)
}
