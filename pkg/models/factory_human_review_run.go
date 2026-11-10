package models

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrFactoryHumanReviewRunRejected = errors.New("human review run rejected")

// MaybeLinkHumanReviewRun gives explicit reruns new decision authority. It
// rejects stale root payloads before the agent can receive an authorized run.
func MaybeLinkHumanReviewRun(tx *gorm.DB, canvas *Canvas, event *CanvasEvent, run *CanvasRun) error {
	if canvas == nil || event == nil || run == nil || canvas.FactoryID == nil {
		return nil
	}
	version, err := FindCanvasVersionInTransaction(tx, canvas.ID, run.VersionID)
	if err != nil {
		return err
	}
	settings, matches := HumanReviewSettingsFromNodes(version.Nodes)
	if !matches {
		return nil
	}
	if !settings.Enabled {
		return fmt.Errorf("%w: Request review enforcement is disabled", ErrFactoryHumanReviewRunRejected)
	}
	payload, ok := objectMap(RootEventSourcePayload(event.Data.Data()))
	if !ok {
		return fmt.Errorf("%w: Request review run has no authorized context", ErrFactoryHumanReviewRunRejected)
	}
	review, _ := objectMap(payload["humanReview"])
	raw, _ := review["decisionId"].(string)
	id, err := uuid.Parse(raw)
	if err != nil {
		return fmt.Errorf("%w: Request review run has no valid decision: %v", ErrFactoryHumanReviewRunRejected, err)
	}
	original, err := FindFactoryHumanReviewDecision(tx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: Request review decision was not found", ErrFactoryHumanReviewRunRejected)
	}
	if err != nil {
		return err
	}
	factory, err := FindFactory(tx, canvas.OrganizationID, *canvas.FactoryID)
	if err != nil {
		return err
	}
	pr, err := factory.FindPullRequest(tx, FactoryPullRequestLookup{ID: original.PullRequestID})
	if err != nil {
		return err
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "human-review:"+pr.ID.String()).Error; err != nil {
		return err
	}
	canvas, err = FindCanvasWithoutOrgScopeInTransaction(tx, canvas.ID)
	if err != nil {
		return err
	}
	if err := pr.LockForUpdate(tx); err != nil {
		return err
	}
	current, err := pr.CurrentHumanReview(tx)
	if err != nil {
		return err
	}
	if current == nil || current.ID != original.ID || pr.State != FactoryPullRequestStateOpen || pr.CurrentRevisionID == nil || *pr.CurrentRevisionID != original.RevisionID || original.ConfigurationVersion != run.VersionID.String() || canvas.LiveVersionID == nil || *canvas.LiveVersionID != run.VersionID {
		return fmt.Errorf("%w: Request review run is disabled or superseded", ErrFactoryHumanReviewRunRejected)
	}
	source, _, err := pr.LatestMergeConfidenceAssessment(tx, original.AssessedCommit)
	if err != nil {
		return err
	}
	if source == nil || original.SourceRunID == nil || source.ID != *original.SourceRunID {
		return fmt.Errorf("%w: Request review assessment is superseded", ErrFactoryHumanReviewRunRejected)
	}
	decision := original
	if original.RunID != nil && *original.RunID != run.ID {
		decision, err = pr.NewHumanReviewDecision(tx, FactoryHumanReviewDecision{
			RevisionID: original.RevisionID, SourceRunID: original.SourceRunID, RunID: &run.ID,
			ConfigurationVersion: original.ConfigurationVersion, State: FactoryHumanReviewStateWaitingDecision,
			AssessedCommit: original.AssessedCommit, Assessment: original.Assessment,
			CheckRunID: original.CheckRunID, CheckURL: original.CheckURL,
		})
		if err != nil {
			return err
		}
		review["decisionId"] = decision.ID.String()
		event.Data = NewJSONValue(event.Data.Data())
		if err := tx.Model(event).Update("data", event.Data).Error; err != nil {
			return err
		}
	}
	if decision.RunID == nil {
		decision.RunID = &run.ID
		decision.State = FactoryHumanReviewStateWaitingDecision
		if err := decision.Save(tx); err != nil {
			return err
		}
	}
	// Lifecycle startup already attaches automatic runs. Reruns arrive through
	// the router and need the same task authority and activity history.
	return maybeLinkFactoryColumnAppRun(tx, canvas, event, run, FactoryAppTemplateHumanReviewID, "Request review")
}
