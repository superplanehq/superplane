package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHumanReviewSettingsAcceptIndividualReviewerWithoutTeam(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		configuration map[string]any
		valid         bool
	}{
		{name: "personal account reviewer", configuration: map[string]any{"enabled": true, "defaultReviewer": " mytmlt "}, valid: true},
		{name: "individual replaces stale team", configuration: map[string]any{"enabled": true, "defaultReviewer": "mytmlt", "fallbackTeam": "invalid old value"}, valid: true},
		{name: "legacy team", configuration: map[string]any{"enabled": true, "fallbackTeam": "acme/reviewers"}, valid: true},
		{name: "disabled without reviewer", configuration: map[string]any{"enabled": false}, valid: true},
		{name: "missing reviewer", configuration: map[string]any{"enabled": true}},
		{name: "blank reviewer", configuration: map[string]any{"enabled": true, "defaultReviewer": " "}},
		{name: "mention", configuration: map[string]any{"enabled": true, "defaultReviewer": "@mytmlt"}},
		{name: "repository", configuration: map[string]any{"enabled": true, "defaultReviewer": "mytmlt/app"}},
		{name: "profile URL", configuration: map[string]any{"enabled": true, "defaultReviewer": "https://github.com/mytmlt"}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			settings, present := HumanReviewSettingsFromNodes([]Node{{
				ID: HumanReviewTriggerNodeID, Type: NodeTypeTrigger,
				Metadata:      FactoryAppTemplateMetadataFor(FactoryAppTemplateHumanReviewID, 1, ProviderGitHub),
				Configuration: scenario.configuration,
			}})
			require.True(t, present)
			if scenario.valid {
				require.NoError(t, settings.Validate())
				return
			}
			require.Error(t, settings.Validate())
		})
	}
}

func TestHumanReviewSettingsPreferPreparationPolicy(t *testing.T) {
	settings, present := HumanReviewSettingsFromNodes([]Node{
		{
			ID: HumanReviewTriggerNodeID, Type: NodeTypeTrigger,
			Metadata:      FactoryAppTemplateMetadataFor(FactoryAppTemplateHumanReviewID, 1, ProviderGitHub),
			Configuration: map[string]any{"enabled": true, "defaultReviewer": "old-owner"},
		},
		{
			ID: HumanReviewPrepareNodeID, Type: NodeTypeComponent,
			Ref:           NodeRef{Component: &ComponentRef{Name: "prepareHumanReview"}},
			Configuration: map[string]any{"enabled": false, "defaultReviewer": "", "fallbackTeam": ""},
		},
	})
	require.True(t, present)
	require.False(t, settings.Enabled)
	require.Empty(t, settings.DefaultReviewer)
	require.Empty(t, settings.FallbackTeam)
}
