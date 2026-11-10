package canvases

import (
	"context"

	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// AuthorizeHumanReviewMutation also covers drafts and editable canvas metadata.
// Publishing repeats this check against the policy currently stored in the DB.
func AuthorizeHumanReviewMutation(ctx context.Context, db *gorm.DB, canvas *models.Canvas, authService authorization.Authorization) error {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
	if err != nil {
		return err
	}
	if models.FactoryAppTemplateID(version.Nodes) != models.FactoryAppTemplateHumanReviewID {
		return nil
	}
	return changesets.RequireHumanReviewAdministrator(ctx, authService, canvas.OrganizationID)
}
