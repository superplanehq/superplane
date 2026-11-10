package factories

import (
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func configureHumanReviewSource(db *gorm.DB, factory *models.Factory, input *factoryTemplateInput) error {
	if input.integrations == nil {
		input.integrations = map[string]factoryTemplateIntegration{}
	}
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return err
	}
	for i := range canvases {
		canvas := &canvases[i]
		if canvas.LiveVersionID == nil {
			continue
		}
		version, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, canvas)
		if err != nil {
			return err
		}
		template, ok := resolveFactoryTemplate(version.Nodes)
		if !ok || template.id != "risk-score" || template.provider != models.ProviderGitHub {
			continue
		}
		input.mergeConfidenceAppID = canvas.ID.String()
		// Older Request review templates did not contain a GitHub trigger.
		// Use the source automation's integration when upgrading those versions.
		if _, configured := input.integrations[models.ProviderGitHub]; !configured {
			for _, node := range version.Nodes {
				if node.ComponentName() != "github.onPullRequest" || node.IntegrationID == nil {
					continue
				}
				input.integrations[models.ProviderGitHub] = factoryTemplateIntegration{id: *node.IntegrationID}
				break
			}
		}
		return nil
	}
	return invalidArgument("Install Merge confidence before Request review.")
}
