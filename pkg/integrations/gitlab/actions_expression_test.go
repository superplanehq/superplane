package gitlab

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// actionSetup is implemented by every GitLab action component. Unlike the
// triggers, actions only read the project when they execute, so they defer
// project validation to runtime and must accept an expression-based project.
type actionSetup interface {
	Setup(core.SetupContext) error
}

// Test__MergeRequestActions__AcceptExpressionProject guards the behaviour
// requested in issue #6108: the GitLab merge request components must accept an
// expression-based project during Setup (deferring resolution to runtime)
// instead of raising a validation error. AcceptMergeRequest already has its own
// case; this locks the invariant in across every merge request action so a
// future strict-validation regression can't silently reintroduce the bug.
func Test__MergeRequestActions__AcceptExpressionProject(t *testing.T) {
	const projectExpression = "{{ $['On Merge Request'].data.project.id }}"

	cases := []struct {
		name       string
		action     actionSetup
		additional map[string]any
	}{
		{
			name:       "AcceptMergeRequest",
			action:     &AcceptMergeRequest{},
			additional: map[string]any{"mergeRequestIid": "42"},
		},
		{
			name:       "ApproveMergeRequest",
			action:     &ApproveMergeRequest{},
			additional: map[string]any{"mergeRequestIid": "42"},
		},
		{
			name:       "CreateMergeComment",
			action:     &CreateMergeComment{},
			additional: map[string]any{"mergeRequestIid": "42", "body": "looks good"},
		},
		{
			name:   "AddReaction",
			action: &AddReaction{},
			additional: map[string]any{
				"mergeRequestIid": "42",
				"content":         "thumbsup",
				"target":          ReactionTargetMergeRequest,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configuration := map[string]any{"project": projectExpression}
			for k, v := range tc.additional {
				configuration[k] = v
			}

			err := tc.action.Setup(core.SetupContext{
				Configuration: configuration,
				// No project in the integration metadata: an expression project
				// must be accepted without an accessibility check.
				Integration: &contexts.IntegrationContext{},
				Metadata:    &contexts.MetadataContext{},
			})

			require.NoError(t, err, "expression-based project should be accepted and validated at runtime")
		})
	}
}
