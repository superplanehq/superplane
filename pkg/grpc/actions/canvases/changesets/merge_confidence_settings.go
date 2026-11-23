package changesets

import (
	"context"
	"errors"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func (p *CanvasPublisher) validateMergeConfidenceCheckPermissions(ctx context.Context) error {
	if p.canvas.FactoryID == nil || models.FactoryAppTemplateID(p.draft.Nodes) != "risk-score" {
		return nil
	}
	for _, node := range p.draft.Nodes {
		if node.ID != "start-assessment-check" || node.ComponentName() != "github.setPullRequestCheckRun" || node.Configuration["enabled"] == false {
			continue
		}
		live, exists := p.findLiveNode(node.ID)
		alreadyEnabled := exists && live.ComponentName() == node.ComponentName() && live.Configuration["enabled"] != false
		sameIntegration := live.IntegrationID == node.IntegrationID ||
			(live.IntegrationID != nil && node.IntegrationID != nil && *live.IntegrationID == *node.IntegrationID)
		if alreadyEnabled && sameIntegration {
			return nil
		}
		integrationID, err := p.getNodeIntegrationID(node)
		if err != nil {
			return err
		}
		if integrationID == nil {
			return grpcerrors.FailedPrecondition(nil, "Connect GitHub before you enable PR checks.")
		}
		integration, err := models.FindIntegrationInTransaction(p.tx, p.options.OrgID, *integrationID)
		if err != nil {
			return err
		}
		integrationCtx := contexts.NewIntegrationContext(p.tx, nil, integration, p.options.Encryptor, p.options.Registry, nil)
		if err := common.VerifyCheckRunPermissions(ctx, integrationCtx, p.options.Registry.HTTPContextInTransaction(p.tx)); err != nil {
			if errors.Is(err, common.ErrCheckRunPermissions) || errors.Is(err, common.ErrCheckRunAppRequired) {
				return grpcerrors.FailedPrecondition(err, err.Error())
			}
			return grpcerrors.FailedPrecondition(err, "Could not verify GitHub permissions. Try again.")
		}
		return nil
	}
	return nil
}

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
