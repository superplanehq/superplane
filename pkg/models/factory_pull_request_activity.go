package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FactoryPullRequestActivityParams struct {
	RunID             uuid.UUID
	Description       string
	RevisionSHA       string
	Access            string
	FeedbackHandlerID *uuid.UUID
}

type FactoryPullRequestActivityOutcome string

const (
	FactoryPullRequestActivityOutcomeReady        FactoryPullRequestActivityOutcome = "ready"
	FactoryPullRequestActivityOutcomeWaiting      FactoryPullRequestActivityOutcome = "waiting"
	FactoryPullRequestActivityOutcomeLimitReached FactoryPullRequestActivityOutcome = "limit_reached"
)

type FactoryPullRequestActivityResult struct {
	Activity        *FactoryPullRequestRun
	Revision        *FactoryPullRequestRevision
	CurrentRevision *FactoryPullRequestRevision
	Outcome         FactoryPullRequestActivityOutcome
}

type FactoryPullRequestAccessResult struct {
	Activity        *FactoryPullRequestRun
	CurrentRevision *FactoryPullRequestRevision
	Outcome         FactoryPullRequestActivityOutcome
}

func (p *FactoryPullRequest) CreateActivity(tx *gorm.DB, params FactoryPullRequestActivityParams) (*FactoryPullRequestActivityResult, error) {
	if params.RunID == uuid.Nil {
		return nil, fmt.Errorf("%w: run id is required", ErrFactoryPullRequestInvalid)
	}

	access, err := normalizeActivityAccess(params.Access)
	if err != nil {
		return nil, err
	}

	if err := p.lock(tx); err != nil {
		return nil, err
	}

	existing, err := FindPullRequestActivityByRunID(tx, params.RunID)
	if err == nil {
		if existing.PullRequestID != p.ID {
			return nil, ErrFactoryPullRequestRunAlreadyLinked
		}
		return p.existingActivityResult(tx, existing, access)
	}
	if !errors.Is(err, ErrFactoryPullRequestActivityNotFound) {
		return nil, err
	}

	result := &FactoryPullRequestActivityResult{}
	var revision *FactoryPullRequestRevision
	if sha := strings.TrimSpace(params.RevisionSHA); sha != "" {
		observed, observeErr := p.ObserveRevision(tx, sha)
		if observeErr != nil {
			return nil, observeErr
		}
		revision = observed.Revision
		result.Revision = revision
	}

	if revision != nil && params.FeedbackHandlerID != nil {
		existingActivity, existingErr := findActiveHandlerRevisionActivity(tx, p.ID, revision.ID, *params.FeedbackHandlerID)
		if existingErr != nil {
			return nil, existingErr
		}
		if existingActivity != nil && existingActivity.RunID != params.RunID {
			return nil, ErrFactoryPullRequestActivityDuplicate
		}
	}

	activity, err := p.insertActivity(tx, params, revision, FactoryPullRequestActivityStateActive, access)
	if err != nil {
		return nil, err
	}

	result.Activity = activity
	result.Revision = revision
	result.CurrentRevision = currentRevisionOf(tx, p)

	if access != FactoryPullRequestAccessExclusive {
		result.Outcome = FactoryPullRequestActivityOutcomeReady
		return result, nil
	}

	accessResult, err := p.grantOrQueueExclusiveAccess(tx, activity)
	if err != nil {
		return nil, err
	}
	result.Activity = accessResult.Activity
	result.Outcome = accessResult.Outcome
	result.CurrentRevision = accessResult.CurrentRevision
	return result, nil
}

func (p *FactoryPullRequest) RequestExclusiveAccess(tx *gorm.DB, activity *FactoryPullRequestRun) (*FactoryPullRequestAccessResult, error) {
	if activity == nil {
		return nil, fmt.Errorf("%w: activity is required", ErrFactoryPullRequestInvalid)
	}
	if err := p.lock(tx); err != nil {
		return nil, err
	}
	return p.grantOrQueueExclusiveAccess(tx, activity)
}

func (a *FactoryPullRequestRun) RequestExclusiveAccess(tx *gorm.DB) (*FactoryPullRequestAccessResult, error) {
	var pullRequest FactoryPullRequest
	if err := tx.Where("id = ?", a.PullRequestID).First(&pullRequest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryPullRequestNotFound
		}
		return nil, err
	}
	return pullRequest.RequestExclusiveAccess(tx, a)
}

func (a *FactoryPullRequestRun) UpdateDescription(tx *gorm.DB, description string) error {
	now := time.Now()
	a.Description = strings.TrimSpace(description)
	a.UpdatedAt = now
	return tx.Model(a).
		Where("pull_request_id = ? AND run_id = ?", a.PullRequestID, a.RunID).
		Updates(map[string]any{
			"description": a.Description,
			"updated_at":  now,
		}).Error
}

func (a *FactoryPullRequestRun) Finalize(tx *gorm.DB, run *CanvasRun) error {
	if run == nil {
		return fmt.Errorf("%w: run is required", ErrFactoryPullRequestInvalid)
	}

	var pullRequest FactoryPullRequest
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", a.PullRequestID).
		First(&pullRequest).Error; err != nil {
		return err
	}

	now := time.Now()
	nextState := a.State
	if a.State == FactoryPullRequestActivityStateActive {
		nextState = FactoryPullRequestActivityStateFinished
	}

	nextAccess := a.Access
	if a.Access == FactoryPullRequestAccessExclusive || a.Access == FactoryPullRequestAccessWaiting {
		nextAccess = FactoryPullRequestAccessReleased
	}

	a.State = nextState
	a.Access = nextAccess
	a.UpdatedAt = now
	if err := tx.Model(a).
		Where("pull_request_id = ? AND run_id = ?", a.PullRequestID, a.RunID).
		Updates(map[string]any{
			"state":      nextState,
			"access":     nextAccess,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}

	if pullRequest.ActiveMutationRunID != nil && *pullRequest.ActiveMutationRunID == a.RunID {
		if err := tx.Model(&pullRequest).Update("active_mutation_run_id", nil).Error; err != nil {
			return err
		}
		pullRequest.ActiveMutationRunID = nil
	}

	return nil
}

func FindPullRequestActivityByRunID(tx *gorm.DB, runID uuid.UUID) (*FactoryPullRequestRun, error) {
	var activity FactoryPullRequestRun
	err := tx.Where("run_id = ?", runID).First(&activity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryPullRequestActivityNotFound
		}
		return nil, err
	}
	return &activity, nil
}

func (p *FactoryPullRequest) insertActivity(
	tx *gorm.DB,
	params FactoryPullRequestActivityParams,
	revision *FactoryPullRequestRevision,
	state, access string,
) (*FactoryPullRequestRun, error) {
	now := time.Now()
	activity := &FactoryPullRequestRun{
		PullRequestID:     p.ID,
		RunID:             params.RunID,
		FeedbackHandlerID: params.FeedbackHandlerID,
		Access:            access,
		State:             state,
		Description:       strings.TrimSpace(params.Description),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if revision != nil {
		activity.RevisionID = &revision.ID
	}

	if err := tx.Create(activity).Error; err != nil {
		return nil, mapFactoryPullRequestRunConstraintError(err)
	}
	return activity, nil
}

func (p *FactoryPullRequest) existingActivityResult(
	tx *gorm.DB,
	activity *FactoryPullRequestRun,
	requestedAccess string,
) (*FactoryPullRequestActivityResult, error) {
	result := &FactoryPullRequestActivityResult{
		Activity:        activity,
		CurrentRevision: currentRevisionOf(tx, p),
		Outcome:         outcomeForActivity(activity),
	}
	if activity.RevisionID != nil {
		revision, err := FindPullRequestRevision(tx, *activity.RevisionID)
		if err != nil {
			return nil, err
		}
		result.Revision = revision
	}
	if requestedAccess != FactoryPullRequestAccessExclusive {
		return result, nil
	}
	if activity.State != FactoryPullRequestActivityStateActive {
		return result, nil
	}

	accessResult, err := p.grantOrQueueExclusiveAccess(tx, activity)
	if err != nil {
		return nil, err
	}
	result.Activity = accessResult.Activity
	result.Outcome = accessResult.Outcome
	result.CurrentRevision = accessResult.CurrentRevision
	return result, nil
}

func (p *FactoryPullRequest) grantOrQueueExclusiveAccess(tx *gorm.DB, activity *FactoryPullRequestRun) (*FactoryPullRequestAccessResult, error) {
	if err := reloadActivity(tx, activity); err != nil {
		return nil, err
	}

	result := &FactoryPullRequestAccessResult{
		Activity:        activity,
		CurrentRevision: currentRevisionOf(tx, p),
	}

	if activity.State == FactoryPullRequestActivityStateLimitReached {
		result.Outcome = FactoryPullRequestActivityOutcomeLimitReached
		return result, nil
	}
	if activity.State != FactoryPullRequestActivityStateActive {
		result.Outcome = FactoryPullRequestActivityOutcomeReady
		return result, nil
	}

	if p.ActiveMutationRunID != nil && *p.ActiveMutationRunID == activity.RunID {
		result.Outcome = FactoryPullRequestActivityOutcomeReady
		return result, nil
	}

	now := time.Now()
	if activity.AccessRequestedAt == nil {
		activity.AccessRequestedAt = &now
	}
	activity.Access = FactoryPullRequestAccessWaiting
	activity.UpdatedAt = now
	if err := tx.Model(activity).
		Where("pull_request_id = ? AND run_id = ?", activity.PullRequestID, activity.RunID).
		Updates(map[string]any{
			"access":              FactoryPullRequestAccessWaiting,
			"access_requested_at": activity.AccessRequestedAt,
			"updated_at":          now,
		}).Error; err != nil {
		return nil, err
	}

	if p.ActiveMutationRunID != nil && *p.ActiveMutationRunID != activity.RunID {
		result.Outcome = FactoryPullRequestActivityOutcomeWaiting
		return result, nil
	}

	oldest, err := oldestWaitingActivity(tx, p.ID)
	if err != nil {
		return nil, err
	}
	if oldest == nil || oldest.RunID != activity.RunID {
		result.Outcome = FactoryPullRequestActivityOutcomeWaiting
		return result, nil
	}

	return p.grantExclusiveAccess(tx, activity)
}

func (p *FactoryPullRequest) grantExclusiveAccess(tx *gorm.DB, activity *FactoryPullRequestRun) (*FactoryPullRequestAccessResult, error) {
	result := &FactoryPullRequestAccessResult{
		Activity:        activity,
		CurrentRevision: currentRevisionOf(tx, p),
	}

	if activity.FeedbackHandlerID != nil {
		handler, err := FindPRFeedbackHandlerByID(tx, *activity.FeedbackHandlerID)
		if err != nil && !errors.Is(err, ErrFactoryPRFeedbackHandlerNotFound) {
			return nil, err
		}
		if handler != nil && handler.MaximumAttempts != nil {
			reserved, limitErr := p.reserveAttempt(tx, activity, *handler.MaximumAttempts)
			if limitErr != nil {
				return nil, limitErr
			}
			if !reserved {
				result.Outcome = FactoryPullRequestActivityOutcomeLimitReached
				return result, nil
			}
		}
	}

	now := time.Now()
	activity.Access = FactoryPullRequestAccessExclusive
	activity.AccessGrantedAt = &now
	activity.UpdatedAt = now
	if err := tx.Model(activity).
		Where("pull_request_id = ? AND run_id = ?", activity.PullRequestID, activity.RunID).
		Updates(map[string]any{
			"access":            FactoryPullRequestAccessExclusive,
			"access_granted_at": now,
			"updated_at":        now,
		}).Error; err != nil {
		return nil, err
	}

	p.ActiveMutationRunID = &activity.RunID
	if err := tx.Model(p).Update("active_mutation_run_id", activity.RunID).Error; err != nil {
		return nil, err
	}

	result.Outcome = FactoryPullRequestActivityOutcomeReady
	return result, nil
}

func (p *FactoryPullRequest) reserveAttempt(tx *gorm.DB, activity *FactoryPullRequestRun, maximumAttempts int) (bool, error) {
	if activity.Attempt != nil {
		return true, nil
	}

	count, err := p.countAttemptsSinceReset(tx, *activity.FeedbackHandlerID)
	if err != nil {
		return false, err
	}
	if count >= maximumAttempts {
		now := time.Now()
		activity.State = FactoryPullRequestActivityStateLimitReached
		activity.Access = FactoryPullRequestAccessReleased
		activity.AttemptLimit = &maximumAttempts
		activity.UpdatedAt = now
		if err := tx.Model(activity).
			Where("pull_request_id = ? AND run_id = ?", activity.PullRequestID, activity.RunID).
			Updates(map[string]any{
				"state":         FactoryPullRequestActivityStateLimitReached,
				"access":        FactoryPullRequestAccessReleased,
				"attempt_limit": maximumAttempts,
				"updated_at":    now,
			}).Error; err != nil {
			return false, err
		}
		return false, nil
	}

	attempt := count + 1
	activity.Attempt = &attempt
	activity.AttemptLimit = &maximumAttempts
	if err := tx.Model(activity).
		Where("pull_request_id = ? AND run_id = ?", activity.PullRequestID, activity.RunID).
		Updates(map[string]any{
			"attempt":       attempt,
			"attempt_limit": maximumAttempts,
			"updated_at":    time.Now(),
		}).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (p *FactoryPullRequest) countAttemptsSinceReset(tx *gorm.DB, handlerID uuid.UUID) (int, error) {
	type resetBoundary struct {
		UpdatedAt time.Time
	}

	var reset resetBoundary
	err := tx.Table("factory_pull_request_runs AS activities").
		Select("activities.updated_at").
		Joins("JOIN workflow_runs ON workflow_runs.id = activities.run_id").
		Where("activities.feedback_handler_id = ?", handlerID).
		Where("activities.pull_request_id = ?", p.ID).
		Where("activities.attempt IS NULL").
		Where("activities.access_granted_at IS NULL").
		Where("activities.state = ?", FactoryPullRequestActivityStateFinished).
		Where("workflow_runs.result = ?", CanvasRunResultPassed).
		Order("activities.updated_at DESC").
		Limit(1).
		Take(&reset).
		Error

	query := tx.Model(&FactoryPullRequestRun{}).
		Where("feedback_handler_id = ?", handlerID).
		Where("pull_request_id = ?", p.ID).
		Where("attempt IS NOT NULL")

	if err == nil {
		query = query.Where("access_granted_at > ?", reset.UpdatedAt)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func findActiveHandlerRevisionActivity(tx *gorm.DB, pullRequestID, revisionID, handlerID uuid.UUID) (*FactoryPullRequestRun, error) {
	var activity FactoryPullRequestRun
	err := tx.
		Where("pull_request_id = ?", pullRequestID).
		Where("revision_id = ?", revisionID).
		Where("feedback_handler_id = ?", handlerID).
		Where("state = ?", FactoryPullRequestActivityStateActive).
		First(&activity).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &activity, nil
}

func oldestWaitingActivity(tx *gorm.DB, pullRequestID uuid.UUID) (*FactoryPullRequestRun, error) {
	var activity FactoryPullRequestRun
	err := tx.
		Where("pull_request_id = ?", pullRequestID).
		Where("state = ?", FactoryPullRequestActivityStateActive).
		Where("access = ?", FactoryPullRequestAccessWaiting).
		Where("access_requested_at IS NOT NULL").
		Order("access_requested_at ASC").
		Order("created_at ASC").
		Order("run_id ASC").
		First(&activity).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &activity, nil
}

func reloadActivity(tx *gorm.DB, activity *FactoryPullRequestRun) error {
	reloaded, err := FindPullRequestActivityByRunID(tx, activity.RunID)
	if err != nil {
		return err
	}
	*activity = *reloaded
	return nil
}

func currentRevisionOf(tx *gorm.DB, pullRequest *FactoryPullRequest) *FactoryPullRequestRevision {
	if pullRequest.CurrentRevisionID == nil {
		return nil
	}
	revision, err := FindPullRequestRevision(tx, *pullRequest.CurrentRevisionID)
	if err != nil {
		return nil
	}
	return revision
}

func outcomeForActivity(activity *FactoryPullRequestRun) FactoryPullRequestActivityOutcome {
	switch activity.State {
	case FactoryPullRequestActivityStateLimitReached:
		return FactoryPullRequestActivityOutcomeLimitReached
	}
	if activity.Access == FactoryPullRequestAccessWaiting {
		return FactoryPullRequestActivityOutcomeWaiting
	}
	return FactoryPullRequestActivityOutcomeReady
}

func normalizeActivityAccess(access string) (string, error) {
	trimmed := strings.TrimSpace(access)
	if trimmed == "" {
		return FactoryPullRequestAccessConcurrent, nil
	}
	switch trimmed {
	case FactoryPullRequestAccessConcurrent, FactoryPullRequestAccessExclusive:
		return trimmed, nil
	default:
		return "", fmt.Errorf("%w: invalid access %q", ErrFactoryPullRequestInvalid, access)
	}
}
