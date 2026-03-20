package runner

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

func Test__RegistrationURL(t *testing.T) {
	t.Parallel()

	runner, err := New(ClientConfig{
		HubURL:            "http://example.com",
		RegistrationToken: "test-token",
	}, func(context.Context, protocol.JobAssignMessage) (json.RawMessage, error) {
		return nil, nil
	})

	require.NoError(t, err)

	url := runner.registrationURL()
	require.Equal(t, "ws://example.com/api/v1/register?token=test-token", url)
}
