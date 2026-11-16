package pulls

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// A later submitted decision ends ownership, so a manual re-request survives.
type ownedReviewerRequest struct {
	Reviewer ReviewerIdentity `json:"reviewer"`
	ReviewID int64            `json:"reviewId"`
}
type managedReviewerState struct {
	Selected []ReviewerIdentity     `json:"selected"`
	Owned    []ownedReviewerRequest `json:"owned"`
	Adding   *ownedReviewerRequest  `json:"adding,omitempty"`
}

// This envelope is shared with core.BeginOrderedWrite.
type managedReviewerPayload struct {
	Lease *core.PublicationLease `json:"lease,omitempty"`
	State managedReviewerState   `json:"state"`
}
type reviewerReconciliation struct {
	execution       core.ExecutionContext
	client          *common.Client
	config          ensureReviewersConfiguration
	number          int
	write           core.OrderedWrite
	lease           core.PublicationLease
	providerContext context.Context
	state           managedReviewerState
	outcome         ensureReviewersOutcome
}

func managedReviewersKey(canvasID, repository string, number int) string {
	return fmt.Sprintf("github.review-requests/%s/%s/%d", canvasID, strings.ToLower(repository), number)
}
func previousManagedReviewers(ctx core.ExecutionContext, client *common.Client, repository string, number int) ([]ReviewerIdentity, error) {
	selected := []ReviewerIdentity{}
	if ctx.Publications == nil {
		return selected, nil
	}
	record, err := ctx.Publications.Get(managedReviewersKey(ctx.WorkflowID, client.RepositoryFullName(repository), number))
	if err != nil {
		return nil, fmt.Errorf("read previous reviewer selection: %w", err)
	}
	if record == nil || len(record.Payload) == 0 {
		return selected, nil
	}
	var payload managedReviewerPayload
	if err := json.Unmarshal(record.Payload, &payload); err != nil {
		return nil, err
	}
	if payload.State.Selected != nil {
		selected = payload.State.Selected
	}
	return selected, nil
}

func reconcileManagedReviewers(ctx core.ExecutionContext, client *common.Client, config ensureReviewersConfiguration, number int) error {
	if ctx.Publications == nil {
		return fmt.Errorf("publication context is required for managed reviewer requests")
	}
	if independent, ok := ctx.Publications.(core.IndependentPublicationContext); ok {
		ctx.Publications = independent.Independent()
	}
	canvasID, err := uuid.Parse(ctx.WorkflowID)
	if err != nil {
		return err
	}
	if config.ExpectedHeadSHA == "" {
		return fmt.Errorf("expected head SHA is required for managed reviewer requests")
	}
	key := managedReviewersKey(ctx.WorkflowID, client.RepositoryFullName(config.Repository), number)
	write, accepted, err := core.AcceptOrderedWrite(ctx.Publications, key, canvasID, ctx.NodeID)
	if err != nil {
		return err
	}
	r := reviewerReconciliation{execution: ctx, client: client, config: config, number: number, write: write, outcome: ensureReviewersOutcome{
		ExecutionID: ctx.ID.String(), Requested: []ReviewerIdentity{}, AlreadyCovered: []ReviewerIdentity{}, Removed: []ReviewerIdentity{}, Selected: []ReviewerIdentity{},
	}}
	if !accepted {
		return r.emitSuperseded()
	}
	lease, raw, err := core.BeginOrderedWrite(ctx.Publications, write, time.Minute)
	if errors.Is(err, core.ErrPublicationSuperseded) {
		return r.emitSuperseded()
	}
	if err != nil {
		return err
	}
	r.lease = lease
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &r.state); err != nil {
			return errors.Join(err, core.FinishOrderedWrite(ctx.Publications, write, lease, nil))
		}
	}
	providerContext, cancel := context.WithDeadline(context.Background(), lease.ExpiresAt.Add(-15*time.Second))
	defer cancel()
	r.providerContext = providerContext
	reconcileErr := r.reconcile()
	// Check the lease ID even at finish: an expired writer must not replace a
	// successor's ownership state or emit a successful completion.
	if finishErr := r.persist(nil); finishErr != nil {
		return errors.Join(reconcileErr, finishErr)
	}
	if reconcileErr != nil && reconcileErr != core.ErrPublicationSuperseded {
		return reconcileErr
	}
	current, err := core.IsCurrentOrderedWrite(ctx.Publications, write)
	if err != nil {
		return err
	}
	if !current {
		return r.emitSuperseded()
	}
	return ctx.ExecutionState.Emit("default", "github.pullRequestReviewersEnsured", []any{r.outcome})
}
func (r *reviewerReconciliation) emitSuperseded() error {
	r.outcome.Status, r.outcome.Reason = "skipped", "superseded"
	r.outcome.Selected = []ReviewerIdentity{}
	return r.execution.ExecutionState.Emit("default", "github.pullRequestReviewersEnsured", []any{r.outcome})
}
func (r *reviewerReconciliation) persist(lease *core.PublicationLease) error {
	return r.execution.Publications.Update(r.write.Key, func(record *core.OrderedPublication) error {
		var payload managedReviewerPayload
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return err
		}
		if payload.Lease == nil || payload.Lease.ID != r.lease.ID {
			return core.ErrPublicationSuperseded
		}
		payload.State, payload.Lease = r.state, lease
		encoded, err := json.Marshal(payload)
		record.Payload = encoded
		return err
	})
}
func (r *reviewerReconciliation) readCurrent() (*github.PullRequest, error) {
	pr, _, err := r.client.GetPullRequest(r.providerContext, r.config.Repository, r.number)
	if err != nil {
		return nil, fmt.Errorf("read pull request: %w", err)
	}
	current, err := core.IsCurrentOrderedWrite(r.execution.Publications, r.write)
	if err != nil {
		return nil, err
	}
	r.outcome.HeadSHA = pr.GetHead().GetSHA()
	switch {
	case !current:
		r.outcome.Reason = "superseded"
	case pr.GetState() != "open" || pr.GetMerged():
		r.outcome.Reason = "closed"
	case pr.GetHead().GetSHA() != r.config.ExpectedHeadSHA:
		r.outcome.Reason = "head_mismatch"
	default:
		return pr, nil
	}
	r.outcome.Status = "skipped"
	return nil, core.ErrPublicationSuperseded
}
func (r *reviewerReconciliation) reconcile() (resultErr error) {
	// All automations on this integration share this boundary. Two automations
	// must not both claim a request that was absent before either added it.
	sharedKey := fmt.Sprintf("github.review-requests-lock/%s/%d", strings.ToLower(r.client.RepositoryFullName(r.config.Repository)), r.number)
	sharedLease, err := core.BeginPublicationLease(r.execution.Publications, sharedKey, time.Minute)
	if err != nil {
		return err
	}
	defer func() {
		if finishErr := core.FinishPublicationLease(r.execution.Publications, sharedKey, sharedLease); finishErr != nil {
			resultErr = errors.Join(resultErr, finishErr)
		}
	}()
	pr, err := r.readCurrent()
	if err != nil {
		return err
	}
	if len(r.config.Reviewers) > 0 {
		if err := validateReviewerEligibility(r.providerContext, r.client, r.config, pr); err != nil {
			return err
		}
	}
	reviews, err := r.client.ListReviews(r.providerContext, r.config.Repository, r.number)
	if err != nil {
		return fmt.Errorf("read pull request reviews: %w", err)
	}
	pr, err = r.readCurrent()
	if err != nil {
		return err
	}
	r.recoverOwnership(pr, reviews)
	if err := r.persist(&r.lease); err != nil {
		return err
	}
	if err := r.requestMissing(pr, reviews); err != nil {
		return err
	}
	// Add every replacement before withdrawing any old request.
	if err := r.withdrawUnselected(); err != nil {
		return err
	}
	if _, err := r.readCurrent(); err != nil {
		return err
	}
	r.state.Selected = append([]ReviewerIdentity{}, r.config.Reviewers...)
	r.outcome.Selected = r.state.Selected
	r.outcome.Status = "already_covered"
	if len(r.outcome.Requested) > 0 {
		r.outcome.Status = "requested"
	}
	return nil
}
func (r *reviewerReconciliation) recoverOwnership(pr *github.PullRequest, reviews []*github.PullRequestReview) {
	// Recover an uncertain POST only for an identity that was absent before it.
	if r.state.Adding != nil {
		if requestedReviewer(pr, r.state.Adding.Reviewer) {
			r.state.Owned = append(r.state.Owned, *r.state.Adding)
		}
		r.state.Adding = nil
	}
	r.state.Owned = slices.DeleteFunc(r.state.Owned, func(owned ownedReviewerRequest) bool {
		return !requestedReviewer(pr, owned.Reviewer) || reviewerDecisionID(reviews, owned.Reviewer) != owned.ReviewID
	})
}
func (r *reviewerReconciliation) requestMissing(pr *github.PullRequest, reviews []*github.PullRequestReview) error {
	for _, reviewer := range r.config.Reviewers {
		if requestedReviewer(pr, reviewer) || hasCurrentApproval(reviews, reviewer, r.outcome.HeadSHA) {
			r.outcome.AlreadyCovered = append(r.outcome.AlreadyCovered, reviewer)
			continue
		}
		// Re-read head and requests at every mutation, including after partial adds.
		current, err := r.readCurrent()
		if err != nil {
			return err
		}
		if requestedReviewer(current, reviewer) {
			r.outcome.AlreadyCovered = append(r.outcome.AlreadyCovered, reviewer)
			continue
		}
		r.state.Adding = &ownedReviewerRequest{Reviewer: reviewer, ReviewID: reviewerDecisionID(reviews, reviewer)}
		if err := r.persist(&r.lease); err != nil {
			return err
		}
		_, _, addErr := r.client.AddPullRequestReviewers(r.providerContext, r.config.Repository, r.number, github.ReviewersRequest{Reviewers: []string{reviewer.Login}})
		if addErr == nil {
			r.state.Owned = append(r.state.Owned, *r.state.Adding)
			r.state.Adding = nil
		} else if code := common.StatusCode(addErr); code >= 400 && code < 500 {
			r.state.Adding = nil
		}
		if err := r.persist(&r.lease); err != nil {
			return errors.Join(addErr, err)
		}
		if addErr != nil {
			return fmt.Errorf("request pull request reviewer: %w", addErr)
		}
		r.outcome.Requested = append(r.outcome.Requested, reviewer)
	}
	return nil
}
func (r *reviewerReconciliation) withdrawUnselected() error {
	for _, owned := range slices.Clone(r.state.Owned) {
		if containsReviewer(r.config.Reviewers, owned.Reviewer) {
			continue
		}
		reviews, err := r.client.ListReviews(r.providerContext, r.config.Repository, r.number)
		if err != nil {
			return err
		}
		pr, err := r.readCurrent()
		if err != nil {
			return err
		}
		if requestedReviewer(pr, owned.Reviewer) && reviewerDecisionID(reviews, owned.Reviewer) == owned.ReviewID {
			if _, err := r.client.RemovePullRequestReviewers(r.providerContext, r.config.Repository, r.number, github.ReviewersRequest{Reviewers: []string{owned.Reviewer.Login}}); err != nil {
				return fmt.Errorf("withdraw pull request reviewer: %w", err)
			}
			r.outcome.Removed = append(r.outcome.Removed, owned.Reviewer)
		}
		r.state.Owned = slices.DeleteFunc(r.state.Owned, func(other ownedReviewerRequest) bool { return sameReviewer(other.Reviewer, owned.Reviewer) })
		if err := r.persist(&r.lease); err != nil {
			return err
		}
	}
	return nil
}
func sameReviewer(a, b ReviewerIdentity) bool {
	return a.ID == b.ID && strings.EqualFold(a.Login, b.Login)
}
func containsReviewer(reviewers []ReviewerIdentity, reviewer ReviewerIdentity) bool {
	return slices.ContainsFunc(reviewers, func(other ReviewerIdentity) bool { return sameReviewer(other, reviewer) })
}
func requestedReviewer(pr *github.PullRequest, reviewer ReviewerIdentity) bool {
	return slices.ContainsFunc(pr.RequestedReviewers, func(user *github.User) bool {
		return user.GetID() == reviewer.ID && strings.EqualFold(user.GetLogin(), reviewer.Login)
	})
}
func reviewerDecisionID(reviews []*github.PullRequestReview, reviewer ReviewerIdentity) int64 {
	return latestReviewerDecision(reviews, reviewer).GetID()
}
