package contexts

import (
	"encoding/json"
	"strings"

	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
)

// Results enter only through the authenticated runner completion for the bound
// execution. No agent credential can publish a check or record an approval.
func (s *ExecutionStateContext) prepareHumanReviewOutput(channel, payloadType string, payloads []any) error {
	component := strings.TrimSuffix(payloadType, ".finished")
	if !models.IsAgentRunnerComponent(component) || component == payloadType || (channel != "passed" && channel != "failed") || s.IsFinished() || s.IsCancelling() {
		return nil
	}
	decision, err := models.FindFactoryHumanReviewDecisionByRun(s.tx, s.execution.RunID)
	if err != nil || decision == nil {
		return err
	}
	workspace, err := models.FindFactory(s.tx, decision.OrganizationID, decision.FactoryID)
	if err != nil {
		return err
	}
	pr, err := workspace.FindPullRequest(s.tx, models.FactoryPullRequestLookup{ID: decision.PullRequestID})
	if err != nil {
		return err
	}
	if err := pr.LockForUpdate(s.tx); err != nil {
		return err
	}
	current, err := pr.CurrentHumanReview(s.tx)
	if err != nil {
		return err
	}
	if current == nil || current.ID != decision.ID || current.State != models.FactoryHumanReviewStateWaitingDecision || pr.CurrentRevisionID == nil || *pr.CurrentRevisionID != current.RevisionID {
		return nil
	}
	canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(s.tx, s.execution.WorkflowID)
	if err != nil {
		return err
	}
	if canvas.FactoryID == nil || *canvas.FactoryID != current.FactoryID || canvas.OrganizationID != current.OrganizationID || canvas.LiveVersionID == nil || canvas.LiveVersionID.String() != current.ConfigurationVersion {
		return nil
	}
	version, err := models.FindCanvasVersionInTransaction(s.tx, canvas.ID, *canvas.LiveVersionID)
	if err != nil {
		return err
	}
	settings, recognized := models.HumanReviewSettingsFromNodes(version.Nodes)
	if !recognized || !settings.Enabled || settings.AgentNodeID != s.execution.NodeID {
		return nil
	}
	matchedComponent := false
	for _, node := range version.Nodes {
		if node.ID == s.execution.NodeID && node.ComponentName() == component {
			matchedComponent = true
			break
		}
	}
	if !matchedComponent || (s.execution.FrozenComponentName() != "" && s.execution.FrozenComponentName() != component) {
		return nil
	}
	if pr.State != models.FactoryPullRequestStateOpen && pr.State != models.FactoryPullRequestStateDraft {
		return nil
	}
	current.State = models.FactoryHumanReviewStateFailed
	current.BlockingReason = "The Request review agent failed or returned an invalid decision. Rerun Request review."
	if channel == "passed" && len(payloads) == 1 {
		payload, _ := payloads[0].(map[string]any)
		result, _ := payload["result"].(map[string]any)
		encoded, marshalErr := json.Marshal(result["humanReview"])
		var output humanreview.Decision
		if marshalErr == nil && json.Unmarshal(encoded, &output) == nil && humanreview.ValidateDecision(output, current.AssessedCommit) == nil {
			var raw map[string]any
			if err := json.Unmarshal(encoded, &raw); err != nil {
				return err
			}
			current.Result = datatypes.JSONMap(raw)
			current.State = models.FactoryHumanReviewStateDecided
			current.ReviewRequired = *output.ReviewRequired
			current.Rationale = output.Rationale
			current.SelectedReviewers = datatypes.NewJSONSlice([]models.FactoryHumanReviewReviewer{})
			for _, reviewer := range output.SelectedReviewers {
				current.SelectedReviewers = append(current.SelectedReviewers, models.FactoryHumanReviewReviewer{ID: reviewer.ID, Login: reviewer.Login, Reason: reviewer.Reason})
			}
			current.BlockingReason = ""
		}
	}
	if current.State == models.FactoryHumanReviewStateFailed {
		run, err := models.FindUnscopedCanvasRun(s.tx, s.execution.RunID)
		if err != nil {
			return err
		}
		if err := run.AddError(s.tx, current.BlockingReason, config.MaxPayloadSize()); err != nil {
			return err
		}
	}
	return current.Save(s.tx)
}
