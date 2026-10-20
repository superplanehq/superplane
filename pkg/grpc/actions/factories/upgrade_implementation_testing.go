package factories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// This paragraph marks completion of the testing-guidance upgrade, independently
// of factoryTemplateVersion. Keep it aligned with guideline 3 in both templates.
const implementationTestingGuidance = "Follow the repository's testing instructions and use its existing test tools and conventions. For bug fixes, add a regression test that reproduces the failure when practical. For behavior changes, add or update tests that verify observable behavior. Avoid tests that only mirror the implementation or assert incidental presentation details. Run the relevant tests, lint, and build checks. Fix failures caused by your changes. Report any checks you could not run and explain why."

// UpgradeOrganizationImplementationTestingGuidance replaces stamped Implement
// canvases that lack the new paragraph. The first replacement drops customer
// edits; subsequent starts preserve canvases that already contain the paragraph.
func UpgradeOrganizationImplementationTestingGuidance(
	ctx context.Context,
	organizationID uuid.UUID,
	deps IntakeDependencies,
) (BuiltInAutomationReset, error) {
	db := database.DB(ctx)
	factories, err := models.ListFactories(db, organizationID)
	if err != nil {
		return BuiltInAutomationReset{}, err
	}

	var result BuiltInAutomationReset
	err = db.Transaction(func(tx *gorm.DB) error {
		for i := range factories {
			canvases, err := factories[i].ListCanvases(tx)
			if err != nil {
				return err
			}
			for j := range canvases {
				canvas := &canvases[j]
				version, err := models.FindLiveCanvasVersionByCanvasInTransaction(tx, canvas)
				if err != nil {
					return fmt.Errorf("load %q: %w", canvas.Name, err)
				}
				if models.FactoryAppTemplateID(version.Nodes) != "line-implementation" ||
					hasImplementationTestingGuidance(version.Nodes) {
					continue
				}
				saved, err := resetBuiltInAutomation(ctx, tx, &factories[i], canvas, deps)
				if err != nil {
					return fmt.Errorf("upgrade %q: %w", canvas.Name, err)
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

func hasImplementationTestingGuidance(nodes []models.Node) bool {
	for _, node := range nodes {
		steps, _ := node.Configuration["steps"].([]any)
		for _, step := range steps {
			item, _ := step.(map[string]any)
			if item["name"] != "Implementation" {
				continue
			}
			prompt, _ := item["prompt"].(string)
			if strings.Contains(prompt, implementationTestingGuidance) {
				return true
			}
		}
	}
	return false
}
