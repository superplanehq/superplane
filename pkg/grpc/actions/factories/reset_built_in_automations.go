package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases"
	"github.com/superplanehq/superplane/pkg/grpc/actions/canvases/changesets"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const builtInAutomationResetMessage = "Reset built-in automation"

// BuiltInAutomationReset is how many built-in automations were saved.
type BuiltInAutomationReset struct {
	Reset int `json:"reset"`
}

// ResetOrganizationBuiltInAutomations replaces every built-in automation in
// the organization with its current template and saves that canvas. Custom
// automations are left unchanged. Builders carry supported user settings into
// the current templates; prompts and workflow structure come from those templates.
func ResetOrganizationBuiltInAutomations(
	ctx context.Context,
	organizationID uuid.UUID,
	deps IntakeDependencies,
) (BuiltInAutomationReset, error) {
	db := database.DB(ctx)
	factoryList, err := models.ListFactories(db, organizationID)
	if err != nil {
		return BuiltInAutomationReset{}, err
	}

	var result BuiltInAutomationReset
	err = db.Transaction(func(tx *gorm.DB) error {
		for i := range factoryList {
			canvasesForFactory, listErr := factoryList[i].ListCanvases(tx)
			if listErr != nil {
				return listErr
			}
			for j := range canvasesForFactory {
				saved, resetErr := resetBuiltInAutomation(ctx, tx, &factoryList[i], &canvasesForFactory[j], deps)
				if resetErr != nil {
					return fmt.Errorf("reset %q in factory %q: %w", canvasesForFactory[j].Name, factoryList[i].Name, resetErr)
				}
				if saved {
					result.Reset++
				}
			}
		}
		return nil
	})
	if err != nil {
		return BuiltInAutomationReset{}, err
	}
	return result, nil
}

func resetBuiltInAutomation(
	ctx context.Context,
	tx *gorm.DB,
	factory *models.Factory,
	canvas *models.Canvas,
	deps IntakeDependencies,
) (bool, error) {
	version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
	if err != nil {
		return false, err
	}
	materialized, err := materializeLiveAutomationDefaults(tx, factory, canvas, version, factoryAutomationSettingsFromVersion(version))
	if notBuiltInAutomation(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	doc, err := yaml.CanvasFromYAML([]byte(materialized.canvasYAML))
	if err != nil {
		return false, err
	}
	if err := validateBuiltInResetSettings(doc, deps.Registry); err != nil {
		return false, err
	}
	nodes, edges, err := doc.Parse(deps.Registry, canvas.OrganizationID.String())
	if err != nil {
		return false, err
	}
	err = canvases.PublishGeneratedCanvasNodesWithOwner(
		ctx,
		tx,
		canvas,
		nil,
		builtInAutomationResetMessage,
		nodes,
		edges,
		changesets.CanvasPublisherOptions{
			Registry:       deps.Registry,
			OrgID:          canvas.OrganizationID,
			Encryptor:      deps.Encryptor,
			AuthService:    deps.AuthService,
			WebhookBaseURL: deps.WebhookBaseURL,
		},
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

func notBuiltInAutomation(err error) bool {
	return errors.Is(err, errInvalidArgument) && strings.Contains(err.Error(), "no bundled defaults")
}
