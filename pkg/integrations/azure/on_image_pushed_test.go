package azure

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	workercontexts "github.com/superplanehq/superplane/pkg/workers/contexts"
)

func TestOnImagePushed_DefaultRunTitle(t *testing.T) {
	trigger := &OnImagePushed{}
	builder := workercontexts.NewNodeConfigurationBuilder(nil, uuid.Nil).WithRootPayload(map[string]any{
		"type": "azure.image.pushed",
		"data": map[string]any{
			"subject": "/registries/example/repositories/api",
			"data": map[string]any{
				"target": map[string]any{
					"repository": "api",
					"tag":        "v1.2.3",
				},
			},
		},
	})

	out, err := builder.ResolveTemplateExpressions(trigger.DefaultRunTitle())
	require.NoError(t, err)
	require.Equal(t, "api:v1.2.3", out)
}

func TestOnImagePushed_DefaultRunTitle_FallsBackToSubject(t *testing.T) {
	trigger := &OnImagePushed{}
	builder := workercontexts.NewNodeConfigurationBuilder(nil, uuid.Nil).WithRootPayload(map[string]any{
		"type": "azure.image.pushed",
		"data": map[string]any{
			"subject": "/registries/example/repositories/api",
			"data": map[string]any{
				"target": map[string]any{},
			},
		},
	})

	out, err := builder.ResolveTemplateExpressions(trigger.DefaultRunTitle())
	require.NoError(t, err)
	require.Equal(t, "/registries/example/repositories/api", out)
}
