package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

func TestManualTaskMarkerFromEventData(t *testing.T) {
	marker := NewManualTaskMarker()
	body := AppendManualTaskMarker("Stop double charges.", marker)

	found, ok := ManualTaskMarkerFromEventData(map[string]any{
		"type": IssueEventPayloadType,
		"data": map[string]any{
			"issue": map[string]any{
				"body": body,
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, marker, found)

	_, ok = ManualTaskMarkerFromBody("no marker here")
	assert.False(t, ok)
}

func TestPendingManualTaskLabelKeepsTheActorOutOfTheBodyMarker(t *testing.T) {
	marker := NewManualTaskMarker()
	label := PendingManualTaskLabel(marker, "superplane-bot")

	found, ok := ManualTaskMarkerFromLabel(label)
	require.True(t, ok)
	assert.Equal(t, marker, found)
	assert.Equal(t, "superplane-bot", ManualTaskActorFromLabel(label))
	assert.NotContains(t, AppendManualTaskMarker("Stop double charges.", marker), "superplane-bot")
}

func TestIssueAuthorFromEventData(t *testing.T) {
	login, ok := IssueAuthorFromEventData(map[string]any{
		"type": IssueEventPayloadType,
		"data": map[string]any{
			"issue": map[string]any{
				"user": map[string]any{"login": "superplane-bot"},
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, "superplane-bot", login)

	_, ok = IssueAuthorFromEventData(map[string]any{
		"type": IssueEventPayloadType,
		"data": map[string]any{
			"issue": map[string]any{"body": "copied marker"},
		},
	})
	assert.False(t, ok)
}

func TestAppBotLogin(t *testing.T) {
	assert.Equal(t, "", AppBotLogin(nil))

	pat := &models.Integration{
		Properties: datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
			{Name: common.PropertyAppSlug, Value: "superplane"},
		}),
	}
	assert.Equal(t, "", AppBotLogin(pat))

	app := &models.Integration{
		Properties: datatypes.NewJSONSlice([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyAuthMethod, Value: common.AuthMethodApp},
			{Name: common.PropertyAppSlug, Value: "superplane"},
		}),
	}
	assert.Equal(t, "superplane[bot]", AppBotLogin(app))
}

func TestIssueURLFromEventData_NormalizesOwnerAndRepositoryCase(t *testing.T) {
	issueURL, ok := IssueURLFromEventData(map[string]any{
		"type": IssueEventPayloadType,
		"data": map[string]any{
			"issue": map[string]any{
				"html_url": "https://GitHub.com/Acme/Payments/issues/12/",
			},
		},
	})
	require.True(t, ok)
	assert.Equal(t, "https://github.com/acme/payments/issues/12", issueURL)
}
