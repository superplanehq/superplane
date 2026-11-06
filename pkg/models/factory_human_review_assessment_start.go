package models

import "gorm.io/gorm"

// Starting a new assessment retires any earlier decision immediately, including
// a successful decision for the same commit. Publication then queues that gate.
func (p *FactoryPullRequest) queueHumanReviewAssessment(tx *gorm.DB, event *CanvasEvent, run *CanvasRun) error {
	if p.Provider != FactoryPullRequestProviderGitHub || (p.State != FactoryPullRequestStateOpen && p.State != FactoryPullRequestStateDraft) {
		return nil
	}
	if err := p.LockForUpdate(tx); err != nil {
		return err
	}
	previous, err := p.CurrentHumanReview(tx)
	if err != nil || previous == nil || previous.State == FactoryHumanReviewStateDisabled {
		return err
	}
	if previous.SourceRunID != nil && *previous.SourceRunID == run.ID {
		return nil
	}
	if p.CurrentRevisionID == nil || *p.CurrentRevisionID != previous.RevisionID {
		return nil
	}
	payload, _ := objectMap(RootEventSourcePayload(event.Data.Data()))
	pr, _ := objectMap(payload["pull_request"])
	head, _ := objectMap(pr["head"])
	sha, _ := head["sha"].(string)
	if sha == "" || sha != previous.AssessedCommit {
		return nil
	}
	if previous.SourceRunID != nil {
		source, err := FindUnscopedCanvasRun(tx, *previous.SourceRunID)
		if err != nil {
			return err
		}
		if source.CreatedAt != nil && run.CreatedAt != nil {
			if source.CreatedAt.After(*run.CreatedAt) || (source.CreatedAt.Equal(*run.CreatedAt) && source.ID.String() > run.ID.String()) {
				return nil
			}
		}
	}
	_, err = p.NewHumanReviewDecision(tx, FactoryHumanReviewDecision{
		RevisionID:           previous.RevisionID,
		SourceRunID:          &run.ID,
		AssessedCommit:       sha,
		ConfigurationVersion: previous.ConfigurationVersion,
		State:                FactoryHumanReviewStateWaitingAssessment,
		BlockingReason:       "Wait for the new merge-confidence assessment to finish.",
		CheckRunID:           previous.CheckRunID,
		CheckURL:             previous.CheckURL,
	})
	return err
}
