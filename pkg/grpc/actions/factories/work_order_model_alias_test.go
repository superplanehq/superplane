package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func Test__ResolveDispatchModelAliases__KeepsAStableVersionedId(t *testing.T) {
	t.Parallel()

	dispatches := resolveDispatchModelAliases([]models.FactoryWorkOrderLineDispatchRecord{{
		FactoryWorkOrderLineDispatch: models.FactoryWorkOrderLineDispatch{Model: "sonnet"},
	}})
	assert.Equal(t, "claude-sonnet-4-6", dispatches[0].Model)

	dispatches = resolveDispatchModelAliases([]models.FactoryWorkOrderLineDispatchRecord{{
		FactoryWorkOrderLineDispatch: models.FactoryWorkOrderLineDispatch{Model: "claude-opus-4-6"},
	}})
	assert.Equal(t, "claude-opus-4-6", dispatches[0].Model)

	names := resolveModelNames([]string{"opus", "claude-sonnet-4-6"})
	assert.Equal(t, []string{"claude-opus-5-5", "claude-sonnet-4-6"}, names)
}
