package contexts

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type NotificationContext struct {
	tx         *gorm.DB
	orgID      uuid.UUID
	workflowID uuid.UUID
}

func NewNotificationContext(tx *gorm.DB, orgID, workflowID uuid.UUID) *NotificationContext {
	return &NotificationContext{
		tx:         tx,
		orgID:      orgID,
		workflowID: workflowID,
	}
}

func (c *NotificationContext) Send(title, body, url, urlLabel string, receivers core.NotificationReceivers) error {
	orgID := c.orgID
	if orgID == uuid.Nil {
		if c.workflowID == uuid.Nil {
			return fmt.Errorf("notification context missing organization and workflow IDs")
		}

		workflow, err := models.FindWorkflowWithoutOrgScopeInTransaction(c.tx, c.workflowID)
		if err != nil {
			return fmt.Errorf("failed to resolve workflow organization: %w", err)
		}

		orgID = workflow.OrganizationID
	}

	message := messages.NewNotificationEmailRequestedMessage(
		orgID.String(),
		title,
		body,
		url,
		urlLabel,
		receivers.Emails,
		receivers.Groups,
		receivers.Roles,
	)

	return message.Publish()
}
