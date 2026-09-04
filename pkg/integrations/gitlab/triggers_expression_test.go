package gitlab

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// triggerSetup is implemented by every GitLab trigger. All of them register a
// project webhook during Setup, so none can accept an expression-based project.
type triggerSetup interface {
	Setup(core.TriggerContext) error
}

// Test__Triggers__RejectExpressionProject guards the invariant behind the fix
// for issue #6108: triggers provision a project webhook at setup time, so an
// expression-based project must be rejected up front rather than reaching
// RequestWebhook with an unresolved value.
func Test__Triggers__RejectExpressionProject(t *testing.T) {
	metadata := Metadata{
		Projects: []ProjectMetadata{
			{ID: 123, Name: "group/example", URL: "https://gitlab.com/group/example"},
		},
	}

	triggers := map[string]triggerSetup{
		"OnIssue":         &OnIssue{},
		"OnMergeComment":  &OnMergeComment{},
		"OnMergeRequest":  &OnMergeRequest{},
		"OnMilestone":     &OnMilestone{},
		"OnPipeline":      &OnPipeline{},
		"OnRelease":       &OnRelease{},
		"OnTag":           &OnTag{},
		"OnVulnerability": &OnVulnerability{},
	}

	for name, trigger := range triggers {
		t.Run(name, func(t *testing.T) {
			integrationCtx := &contexts.IntegrationContext{Metadata: metadata}
			err := trigger.Setup(core.TriggerContext{
				Integration:   integrationCtx,
				Metadata:      &contexts.MetadataContext{},
				Configuration: map[string]any{"project": "{{ root().data.project.id }}"},
			})

			require.ErrorContains(t, err, "project does not support expressions")
			require.Empty(t, integrationCtx.WebhookRequests, "no webhook should be requested when the project is rejected")
		})
	}
}
