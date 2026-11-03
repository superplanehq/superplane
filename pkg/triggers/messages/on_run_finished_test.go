package messages

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func TestOnRunFinishedRequiresSourceEntryNode(t *testing.T) {
	trigger := &OnRunFinished{}
	for _, source := range []string{"", "   "} {
		require.Error(t, trigger.Setup(core.TriggerContext{Configuration: map[string]any{"sourceEntryNodeId": source}}))
	}
	require.NoError(t, trigger.Setup(core.TriggerContext{Configuration: map[string]any{"sourceEntryNodeId": "assessment"}}))
}
