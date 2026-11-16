package changesets

import (
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
)

// Validate the whole managed graph before setup can change participation.
func validateMergeConfidencePublication(canvas *models.Canvas, version *models.CanvasVersion) error {
	if canvas.FactoryID == nil || models.FactoryAppTemplateID(version.Nodes) != "risk-score" {
		return nil
	}
	var settings map[string]any
	var commentAvailable bool
	publish := true
	for _, node := range version.Nodes {
		if node.ID == "publish-merge-confidence" {
			publish = node.Configuration["enabled"] != false
		}
		if node.ID == "assess-risk" {
			settings = node.Configuration
		}
		if node.ID == "comment-merge-confidence" && node.ComponentName() == "github.publishPullRequestComment" {
			commentAvailable = true
		}
	}
	if saved, present := settings["publishResults"]; present {
		publish = saved != false
	}
	if !publish {
		return nil
	}
	if settings["resultsBody"] == false && settings["resultsComment"] != true {
		return grpcerrors.InvalidArgument(nil, "Select at least one destination for Merge confidence results.")
	}
	if settings["resultsComment"] == true && !commentAvailable {
		return grpcerrors.InvalidArgument(nil, "PR comment publication is unavailable for this automation.")
	}
	return nil
}
