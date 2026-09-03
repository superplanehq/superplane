package merge_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/components/merge"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	_ "github.com/superplanehq/superplane/pkg/registryimports"
)

// Components that accumulate several queue items into one execution, the way
// merge does, must be known to models.IsAggregatingComponent: replaying one
// without an input for every live incoming source produces an execution that
// waits forever for a source that never arrives. Nothing in the type system
// says which components do that, so every registered component is classified
// here and a new one fails until it is.
var nonAggregatingComponents = []string{
	"addMemory",
	"addPullRequest",
	"addPullRequestActivity",
	"addRunError",
	"addWorkOrderArtifact",
	"addWorkOrderComment",
	"approval",
	"broadcastMessage",
	"createWorkOrder",
	"deleteMemory",
	"display",
	"filter",
	"findPullRequest",
	"findWorkOrder",
	"forEach",
	"graphql",
	"http",
	"if",
	"loop",
	"noop",
	"readMemory",
	"reportWorkOrderCheck",
	"runApp",
	"runner",
	"runnerBash",
	"runnerClaudeCode",
	"runnerCodex",
	"runnerJS",
	"runnerOpenRouter",
	"runnerPython",
	"setWorkOrderStatusNote",
	"ssh",
	"timeGate",
	"updateMemory",
	"updatePullRequest",
	"updatePullRequestActivity",
	"updateWorkOrderArtifact",
	"updateWorkOrderStatus",
	"upsertMemory",
	"wait",
}

func Test_EveryComponentIsClassifiedForReplay(t *testing.T) {
	r, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	require.True(t, models.IsAggregatingComponent((&merge.Merge{}).Name()), "merge waits for every distinct incoming source")

	unclassified := make([]string, 0)
	for _, action := range r.ListActions() {
		name := action.Name()
		if models.IsAggregatingComponent(name) || slices.Contains(nonAggregatingComponents, name) {
			continue
		}

		unclassified = append(unclassified, name)
	}

	require.Empty(t, unclassified,
		"these components are not classified for replay: add each to models.aggregatingComponents if it accumulates several inputs into one execution, otherwise list it here",
	)
}
