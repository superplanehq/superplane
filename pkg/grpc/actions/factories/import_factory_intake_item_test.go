package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestWorkOrderOriginFromIntakeItem_UsesSentryIssueTitle(t *testing.T) {
	assert.Equal(t, models.WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/123/",
		Label: "TypeError: boom",
	}, workOrderOriginFromIntakeItem(IntakeItem{
		Title: "  TypeError: boom  ",
		URL:   "https://acme.sentry.io/issues/123/",
	}))
}

func TestWorkOrderOriginFromIntakeItem_FallsBackWhenSentryTitleEmpty(t *testing.T) {
	assert.Equal(t, models.WorkOrderOrigin{
		URL:   "https://acme.sentry.io/issues/123/",
		Label: "123",
	}, workOrderOriginFromIntakeItem(IntakeItem{
		URL: "https://acme.sentry.io/issues/123/",
	}))
}

func TestWorkOrderOriginFromIntakeItem_KeepsGitHubLabel(t *testing.T) {
	assert.Equal(t, models.WorkOrderOrigin{
		URL:   "https://github.com/acme/payments/issues/12",
		Label: "acme/payments#12",
	}, workOrderOriginFromIntakeItem(IntakeItem{
		Title: "Handle duplicate refunds",
		URL:   "https://github.com/acme/payments/issues/12",
	}))
}
