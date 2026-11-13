package factories

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/yaml"
	"gorm.io/gorm"
)

const (
	reviewRequestTemplateID    = "review-request"
	reviewRequestTriggerNodeID = "review-request-trigger"
	reviewRequestAgentNodeID   = "select-reviewers"
	reviewFindingsBodyNodeID   = "findings-body"
	reviewFindingsCommentNode  = "findings-comment"
)

func requireReviewRequest(organization *models.Organization) error {
	if !organization.HasExperimentalFeature(features.FeatureFactoryReviewRequest) {
		return grpcerrors.FailedPrecondition(nil, "Review request is not enabled for this organization.")
	}
	return nil
}

func configureReviewRequest(canvas *yaml.Canvas, params map[string]string) {
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if node.ID == reviewRequestTriggerNodeID {
			node.Configuration["app"] = params["mergeConfidenceAppId"]
		}
		if node.ID == reviewRequestAgentNodeID {
			node.Configuration["fallbackReviewer"] = strings.TrimSpace(params["fallbackReviewer"])
		}
	}
}

// The source is pinned at installation. Settings and resets keep that app even
// if another Merge confidence automation is added later.
func resolveReviewRequestSource(db *gorm.DB, factory *models.Factory, canvas *models.Canvas, version *models.CanvasVersion, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" && models.FactoryAppTemplateID(version.Nodes) == reviewRequestTemplateID {
		for _, node := range version.Nodes {
			if node.ID == reviewRequestTriggerNodeID {
				requested = configString(node.Configuration, "app")
			}
		}
	}
	canvases, err := factory.ListCanvases(db)
	if err != nil {
		return "", err
	}
	var source string
	for i := range canvases {
		candidate := &canvases[i]
		if candidate.ID == canvas.ID || requested != "" && candidate.ID.String() != requested {
			continue
		}
		live, err := models.FindLiveCanvasVersionByCanvasInTransaction(db, candidate)
		if err != nil {
			return "", err
		}
		template, ok := resolveFactoryTemplate(live.Nodes)
		if !ok || template.id != "risk-score" || template.provider != models.ProviderGitHub {
			continue
		}
		if source != "" {
			return "", invalidArgument("Choose a Merge confidence app before you add Review request.")
		}
		source = candidate.ID.String()
	}
	if source != "" {
		return source, nil
	}
	if requested != "" {
		return "", invalidArgument("The Merge confidence app must belong to this workspace.")
	}
	return "", invalidArgument("Add Merge confidence to this workspace before you add Review request.")
}

// configureReviewRequestFindings applies the findings settings saved on the
// agent node, which admin reset keeps, to the publication nodes. Absent
// settings use the template defaults.
func configureReviewRequestFindings(canvas *yaml.Canvas) error {
	var agent map[string]any
	for _, node := range canvas.Spec.Nodes {
		if node.ID == reviewRequestAgentNodeID {
			agent = node.Configuration
		}
	}
	publish, _ := agent["publishReviewFindings"].(bool)
	body := agent["reviewFindingsBody"] != false
	comment := agent["reviewFindingsComment"] == true
	if publish && !body && !comment {
		return fmt.Errorf("select at least one destination for review findings")
	}
	destinations := map[string]struct {
		enabled    bool
		generation int64
	}{
		reviewFindingsBodyNodeID:  {publish && body, savedActivationGeneration(agent["reviewFindingsBodyActivationGeneration"])},
		reviewFindingsCommentNode: {publish && comment, savedActivationGeneration(agent["reviewFindingsCommentActivationGeneration"])},
	}
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if destination, ok := destinations[node.ID]; ok {
			node.Configuration["enabled"] = destination.enabled
			node.Configuration["activationGeneration"] = destination.generation
		}
	}
	return nil
}

// Keep publication settings separate from the selection policy and instructions.
func configureReviewRequestCheck(canvas *yaml.Canvas) {
	var enabled bool
	var generation any = 1
	for _, node := range canvas.Spec.Nodes {
		if node.ID == reviewRequestAgentNodeID {
			enabled, _ = node.Configuration["publishPRCheck"].(bool)
			if saved, ok := node.Configuration["prCheckActivationGeneration"]; ok {
				generation = saved
			}
		}
	}
	for i := range canvas.Spec.Nodes {
		node := &canvas.Spec.Nodes[i]
		if node.ID == "pr-check-pending" {
			node.Configuration["enabled"] = enabled
			node.Configuration["activationGeneration"] = generation
		}
	}
}
