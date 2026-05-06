package contexts

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/impl"
	"gorm.io/datatypes"
)

func TestEventContextDefaultRunTitle(t *testing.T) {
	triggerName := "testDefaultRunTitle"
	registry.RegisterTrigger(triggerName, impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name:            triggerName,
		DefaultRunTitle: "Run: {{ root().data.message }}",
	}))

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	node := &models.CanvasNode{
		Type: models.NodeTypeTrigger,
		Ref:  datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
	}

	ctx := NewEventContext(nil, node, nil, reg)
	runTitle, err := ctx.resolveRunTitle(
		map[string]any{"data": map[string]any{"message": "hello"}},
		map[string]any{"type": "test", "data": map[string]any{"message": "hello"}},
	)
	require.NoError(t, err)
	require.NotNil(t, runTitle)
	require.Equal(t, "Run: hello", *runTitle)
}

func TestEventContextEmptyRunTitleTemplateDisablesDefault(t *testing.T) {
	triggerName := "testEmptyRunTitleTemplate"
	registry.RegisterTrigger(triggerName, impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name:            triggerName,
		DefaultRunTitle: "Run: {{ root().data.message }}",
	}))

	reg, err := registry.NewRegistry(&crypto.NoOpEncryptor{}, registry.HTTPOptions{})
	require.NoError(t, err)

	emptyTemplate := ""
	node := &models.CanvasNode{
		Type:             models.NodeTypeTrigger,
		Ref:              datatypes.NewJSONType(models.NodeRef{Trigger: &models.TriggerRef{Name: triggerName}}),
		RunTitleTemplate: &emptyTemplate,
	}

	ctx := NewEventContext(nil, node, nil, reg)
	runTitle, err := ctx.resolveRunTitle(
		map[string]any{"data": map[string]any{"message": "hello"}},
		map[string]any{"type": "test", "data": map[string]any{"message": "hello"}},
	)
	require.NoError(t, err)
	require.Nil(t, runTitle)
}
