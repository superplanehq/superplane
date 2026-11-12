package reviewrequest

import (
	"slices"

	"github.com/superplanehq/superplane/pkg/factories/lifecycle"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
)

// validatePublication blocks new automations while the organization feature
// is off. Installed automations stay editable after the feature is disabled.
func validatePublication(publication lifecycle.Publication) error {
	proposed := slices.ContainsFunc(publication.Draft.Nodes, func(node models.Node) bool {
		metadata, _ := node.Metadata[models.FactoryAppTemplateMetadataKey].(map[string]any)
		return metadata["id"] == TemplateID
	})
	if !proposed {
		return nil
	}
	exists, err := installed(publication.Tx, publication.Canvas)
	if err != nil || exists {
		return err
	}
	enabled, err := models.OrganizationHasExperimentalFeatures(publication.Tx, publication.OrganizationID, features.FeatureFactoryReviewRequest)
	if err != nil {
		return err
	}
	if !enabled {
		return grpcerrors.FailedPrecondition(nil, "Review request is not enabled for this organization.")
	}
	return nil
}
