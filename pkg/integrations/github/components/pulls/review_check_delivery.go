package pulls

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

const reviewCheckDeliveryLease = time.Minute
const reviewCheckWriteTimeout = 20 * time.Second

// RefreshReviewCheck runs as an integration hook, after the owning execution
// transaction commits. All provider requests run outside publication updates.
func RefreshReviewCheck(ctx core.IntegrationHookContext) error {
	var parameters struct{ Key string }
	if err := mapstructure.Decode(ctx.Parameters, &parameters); err != nil {
		return err
	}
	if !strings.HasPrefix(parameters.Key, reviewCheckPrefix) {
		return fmt.Errorf("invalid review check key")
	}
	store, err := reviewCheckStore(ctx.Integration)
	if err != nil {
		return err
	}
	snapshot, err := store.Get(parameters.Key)
	if err != nil || snapshot == nil {
		return err
	}
	if !snapshot.Disabled {
		state, err := decodeReviewCheckState(snapshot.Payload)
		if err != nil {
			return err
		}
		if state.Configuration.ActivationNodeID != "" {
			eligible, err := reviewCheckStillActive(store, snapshot)
			if err != nil {
				return err
			}
			if !eligible {
				return store.Update(parameters.Key, func(current *core.OrderedPublication) error {
					if current.Revision != snapshot.Revision {
						return core.ErrPublicationUnchanged
					}
					return disableReviewCheck(current)
				})
			}
		}
	}
	state, err := decodeReviewCheckState(snapshot.Payload)
	if err != nil {
		return err
	}
	if state.Configuration.State != "" {
		if state.Error != "" {
			return nil
		}
		owner, err := store.OwnerStatus(snapshot.OwnerCanvasID, snapshot.OwnerNodeID)
		if err != nil {
			return err
		}
		if !owner.Available {
			return stopPullRequestCheck(store, snapshot, "The organization or integration is unavailable.")
		}
		if owner.Removed {
			state.markOwnerRemoved()
		}
		if state.Configuration.State != "pending" && state.State == state.Configuration.State && state.Delivered != "" && state.Delivered == state.fingerprint() && len(state.Retired) == 0 {
			return store.Update(parameters.Key, func(current *core.OrderedPublication) error {
				if current.Revision != snapshot.Revision || current.WakeAt == nil {
					return core.ErrPublicationUnchanged
				}
				current.WakeAt = nil
				return nil
			})
		}
	}
	if state.Delivery != nil && time.Now().Before(state.Delivery.ExpiresAt) {
		return store.Update(parameters.Key, func(current *core.OrderedPublication) error {
			if current.Revision != snapshot.Revision {
				return core.ErrPublicationUnchanged
			}
			// A replacement or webhook can consume the original recovery request.
			current.WakeAt = &state.Delivery.ExpiresAt
			return nil
		})
	}
	if snapshot.Disabled && state.Delivered != "" && state.Delivered == state.fingerprint() {
		return store.Update(parameters.Key, func(current *core.OrderedPublication) error {
			if current.Revision != snapshot.Revision || current.WakeAt == nil {
				return core.ErrPublicationUnchanged
			}
			current.WakeAt = nil
			return nil
		})
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		if state.Configuration.State != "" {
			return retryPullRequestCheckSetup(store, snapshot, err)
		}
		return err
	}
	state.SetupFailures = 0
	monitor, lookupErr := evaluateReviewCheck(client, store, snapshot, &state)
	if lookupErr != nil {
		if state.Configuration.State != "" && permanentCheckError(lookupErr) {
			return stopPullRequestCheck(store, snapshot, "Could not access the pull request. Check the integration permissions and retry this step.")
		}
		if snapshot.Disabled {
			return lookupErr
		}
		state.State, state.Description = "error", "Could not verify pull request reviews"
		if state.Configuration.State != "" {
			state.Description = "Could not verify the pull request."
			if state.CheckRun != nil && state.Configuration.State == "pending" {
				state.State = "pending"
			}
		}
		if state.SHA == "" {
			state.SHA = state.Configuration.AssessedSHA
		}
		monitor = true
	}
	// Provider reads can overlap organization, integration, or canvas removal.
	if state.Configuration.State != "" {
		owner, err := store.OwnerStatus(snapshot.OwnerCanvasID, snapshot.OwnerNodeID)
		if err != nil {
			return err
		}
		if !owner.Available {
			return stopPullRequestCheck(store, snapshot, "The organization or integration is unavailable.")
		}
		if owner.Removed {
			state.markOwnerRemoved()
			monitor = len(state.Retired) > 0
		}
	}
	// Recheck activation after provider reads, outside the publication lock.
	eligible := true
	if !snapshot.Disabled && state.Configuration.ActivationNodeID != "" {
		eligible, err = reviewCheckStillActive(store, snapshot)
		if err != nil {
			return err
		}
	}
	// Claim delivery and commit recovery work before attempting external delivery.
	var revision int64
	err = store.Update(parameters.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		if !eligible {
			return disableReviewCheck(current)
		}
		current.WakeAt = nil
		if monitor {
			current.WakeAt = reviewCheckWake(5 * time.Minute)
		}
		if state.Delivery != nil || state.Delivered != state.fingerprint() {
			state.Delivery = &reviewCheckDelivery{ID: uuid.New(), ExpiresAt: time.Now().Add(reviewCheckDeliveryLease)}
			state.Delivered = ""
			current.WakeAt = &state.Delivery.ExpiresAt
			revision = current.Revision + 1
		}
		current.Payload, err = json.Marshal(state)
		return err
	})
	if err != nil {
		return err
	}
	if revision == 0 {
		return lookupErr
	}

	writeErr := publishReviewCheck(client, &state)
	err = store.Update(parameters.Key, func(current *core.OrderedPublication) error {
		latest, err := decodeReviewCheckState(current.Payload)
		if err != nil {
			return err
		}
		// Creation can finish after a new attempt replaces the publication.
		// Save the receipt on its own target, including a retired target.
		retainDeliveredCheckRun(&latest, &state)
		ownsDelivery := latest.Delivery != nil && latest.Delivery.ID == state.Delivery.ID
		if ownsDelivery {
			latest.Delivery = nil
		}
		current.WakeAt = reviewCheckWake(5 * time.Minute)
		if ownsDelivery && writeErr == nil && current.Generation == snapshot.Generation && latest.fingerprint() == state.fingerprint() {
			latest.Delivered = state.fingerprint()
			if !monitor {
				current.WakeAt = nil
			}
			if len(latest.Retired) > 0 {
				latest.Retired = latest.Retired[1:]
				latest.Delivered = ""
				current.WakeAt = reviewCheckWake(time.Second)
			}
		}
		if ownsDelivery && current.Generation == snapshot.Generation && state.Configuration.State != "" && permanentCheckError(writeErr) {
			latest.Error = "Could not publish the check. Check the integration permissions and retry this step."
			current.WakeAt = nil
		}
		if !ownsDelivery {
			// A late response can change GitHub after recovery has already run.
			latest.Delivered = ""
		}
		if !ownsDelivery || current.Revision != revision {
			current.WakeAt = reviewCheckWake(time.Second)
		}
		current.Payload, err = json.Marshal(latest)
		return err
	})
	if err != nil {
		return err
	}
	if writeErr != nil {
		if state.Configuration.State != "" && permanentCheckError(writeErr) {
			return nil
		}
		return writeErr
	}
	return lookupErr
}

func publishReviewCheck(client *common.Client, state *reviewCheckState) error {
	// Anchor the HTTP deadline to the claim so a paused worker cannot start a
	// status request after its lease expires.
	deadline := state.Delivery.ExpiresAt.Add(reviewCheckWriteTimeout - reviewCheckDeliveryLease)
	writeCtx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	if state.checkRunTarget() != nil {
		return publishPullRequestCheckRun(writeCtx, client, state)
	}
	description := string([]rune(state.Description)[:min(len([]rune(state.Description)), 140)])
	resultURL := state.resultURL()
	_, _, err := client.CreateStatus(writeCtx, state.Configuration.Repository, state.SHA, github.RepoStatus{State: &state.State, Context: &state.Configuration.Context, Description: &description, TargetURL: &resultURL})
	if err != nil {
		return fmt.Errorf("publish review check: %w", err)
	}
	return nil
}

func (s *reviewCheckState) checkRunTarget() *pullRequestCheckRun {
	if len(s.Retired) > 0 {
		return s.Retired[0].CheckRun
	}
	return s.CheckRun
}

func retainDeliveredCheckRun(latest, delivered *reviewCheckState) {
	target := delivered.checkRunTarget()
	if target == nil {
		return
	}
	matches := func(saved *pullRequestCheckRun) bool {
		return saved != nil && saved.ExternalID == target.ExternalID && (saved.ID == 0 || saved.ID == target.ID)
	}
	if matches(latest.CheckRun) {
		latest.CheckRun.ID = target.ID
		return
	}
	for i := range latest.Retired {
		if matches(latest.Retired[i].CheckRun) {
			latest.Retired[i].CheckRun.ID = target.ID
			return
		}
	}
	// Recovery may have retired this target before a paused writer returned.
	// Retain even a missing receipt so a lost creation response can be found.
	latest.Retired = append(latest.Retired, retiredCommitCheck{SHA: delivered.SHA, ResultURL: delivered.resultURL(), CheckRun: target})
}

func evaluateReviewCheck(client *common.Client, store core.PublicationStore, record *core.OrderedPublication, state *reviewCheckState) (bool, error) {
	config := state.Configuration
	if record.Disabled && config.CheckRun && len(state.Retired) == 0 {
		state.SHA, state.State, state.Description = config.AssessedSHA, "success", config.Reason
		return false, nil
	}
	if config.State != "" && len(state.Retired) > 0 {
		retired := state.Retired[0]
		state.SHA, state.State, state.Description = retired.SHA, "error", "A newer attempt superseded this check."
		if retired.State != "" {
			state.State, state.Description = retired.State, retired.Description
		}
		return true, nil
	}
	number, err := parsePullNumber(config.PullNumber)
	if err != nil {
		return false, err
	}
	readCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pr, _, err := client.GetPullRequest(readCtx, config.Repository, number)
	if err != nil {
		return true, err
	}
	if pr.GetHead().GetSHA() == "" {
		return true, fmt.Errorf("pull request head is missing")
	}
	state.SHA = pr.GetHead().GetSHA()
	if config.State != "" {
		// Commit status results stay attached to the assessed commit, including
		// cancellation after a push. Never move an old result onto the new head.
		state.SHA, state.State, state.Description = config.AssessedSHA, config.State, config.Reason
		if pr.GetHead().GetSHA() != config.AssessedSHA || pr.GetState() != "open" || pr.GetMerged() {
			state.State, state.Description = "error", "The pull request is closed or its assessed commit changed."
		}
		if state.State == "pending" {
			// Recorded run errors do not end the attempt. The final step can
			// still publish failure, so wait for the run to stop.
			ended, err := store.AttemptEnded(record.Order.AttemptRootEventID)
			if err != nil {
				return true, err
			}
			if ended {
				state.State, state.Description = "error", "The attempt ended without a final check result."
			}
		}
		if state.State != "pending" {
			state.Configuration.State, state.Configuration.Reason = state.State, state.Description
		}
		return state.State == "pending", nil
	}
	if record.Disabled {
		state.State, state.Description = "success", "Review request check disabled"
		return false, nil
	}
	monitor := pr.GetState() == "open" && !pr.GetMerged()
	if state.SHA != config.AssessedSHA {
		state.State, state.Description = "pending", "Waiting for a decision on the current commit"
		return monitor, nil
	}
	switch config.Phase {
	case "pending_selection":
		finished, err := store.AttemptFinished(record.Order.AttemptRootEventID)
		if err != nil {
			return monitor, err
		}
		if finished {
			state.State, state.Description = "error", "Reviewer selection ended without a valid decision"
			return monitor, nil
		}
		state.State, state.Description = "pending", "Selecting required reviewers"
	case "selection_error":
		state.State, state.Description = "error", config.Reason
	case "no_review":
		state.State, state.Description = "success", config.Reason
	case "required_reviewers":
		reviews, err := client.ListReviews(readCtx, config.Repository, number)
		if err != nil {
			return monitor, err
		}
		users, err := client.ListHumanReviewers(readCtx, config.Repository)
		if err != nil {
			return monitor, err
		}
		if pr.GetUser().GetLogin() == "" {
			return monitor, fmt.Errorf("pull request author is missing")
		}
		approved, changesRequested := false, false
		for _, user := range users {
			if user.GetID() == pr.GetUser().GetID() || strings.EqualFold(user.GetLogin(), pr.GetUser().GetLogin()) {
				continue
			}
			reviewer := ReviewerIdentity{ID: user.GetID(), Login: user.GetLogin()}
			latest := latestReviewerDecision(reviews, reviewer)
			if latest == nil {
				continue
			}
			permission, _, err := client.GetRepositoryPermissionLevel(readCtx, config.Repository, reviewer.Login)
			if err != nil {
				return monitor, fmt.Errorf("read reviewer permission: %w", err)
			}
			if !slices.Contains([]string{"write", "maintain", "admin"}, permission.GetPermission()) {
				continue
			}
			if hasCurrentApproval(reviews, reviewer, config.AssessedSHA) && state.corroborateReviews(reviews, reviewer.ID) == nil {
				approved = true
				break
			}
			changesRequested = changesRequested || latest.GetState() == "CHANGES_REQUESTED"
		}
		// Review pagination can overlap a push. Recheck before persisting success.
		pr, _, err = client.GetPullRequest(readCtx, config.Repository, number)
		if err != nil {
			return monitor, err
		}
		monitor = pr.GetState() == "open" && !pr.GetMerged()
		if pr.GetHead().GetSHA() == "" {
			return monitor, fmt.Errorf("pull request head is missing")
		}
		state.SHA = pr.GetHead().GetSHA()
		if state.SHA != config.AssessedSHA {
			state.State, state.Description = "pending", "Waiting for a decision on the current commit"
			return monitor, nil
		}
		if !approved {
			if err := state.corroborateReviews(reviews, 0); err != nil {
				return monitor, err
			}
		}
		state.observeRequests(pr, users, reviews)
		state.State, state.Description = "pending", "Waiting for one qualifying human approval"
		switch {
		case approved:
			state.State, state.Description = "success", "A qualifying human approved the current commit"
		case changesRequested:
			state.State, state.Description = "failure", "A human reviewer requested changes"
		case len(state.Requests) == 0 && len(state.RemovedRequests) > 0:
			state.State, state.Description = "failure", "The last human review request was removed without approval"
		}

		state.Description += ". " + config.Reason
	default:
		return monitor, fmt.Errorf("invalid persisted review check phase")
	}
	return monitor, nil
}

// Verified activity fences provider propagation delays. The paginated provider
// history still supplies the effective decision; webhook order cannot undo dismissal.
func (s *reviewCheckState) corroborateReviews(reviews []*github.PullRequestReview, reviewerID int64) error {
	for _, activity := range s.ReviewActivity {
		if activity.SHA != s.Configuration.AssessedSHA || (reviewerID != 0 && activity.Reviewer.ID != reviewerID) {
			continue
		}
		index := slices.IndexFunc(reviews, func(review *github.PullRequestReview) bool {
			return review.GetID() == activity.ID && review.GetUser().GetID() == activity.Reviewer.ID && review.GetCommitID() == activity.SHA
		})
		if index < 0 || (reviews[index].GetState() != activity.State && reviews[index].GetState() != "DISMISSED") {
			return fmt.Errorf("GitHub review history has not confirmed verified review activity %d", activity.ID)
		}
	}
	return nil
}

// Keep request history with the review watermark from the same fresh provider
// read. A new submission can clear a request without releasing required review.
func (s *reviewCheckState) observeRequests(pr *github.PullRequest, users []*github.User, reviews []*github.PullRequestReview) {
	current := []observedReviewRequest{}
	for _, user := range pr.RequestedReviewers {
		if user.GetID() <= 0 || strings.EqualFold(user.GetLogin(), pr.GetUser().GetLogin()) {
			continue
		}
		human := strings.EqualFold(user.GetType(), "User") || slices.ContainsFunc(users, func(candidate *github.User) bool { return candidate.GetID() == user.GetID() })
		if !human {
			continue
		}
		reviewer := ReviewerIdentity{ID: user.GetID(), Login: user.GetLogin()}
		current = append(current, observedReviewRequest{Reviewer: reviewer, ReviewID: reviewerSubmittedReviewID(reviews, reviewer)})
	}
	for _, previous := range s.Requests {
		if slices.ContainsFunc(current, func(request observedReviewRequest) bool { return request.Reviewer.ID == previous.Reviewer.ID }) {
			continue
		}
		if reviewerSubmittedReviewID(reviews, previous.Reviewer) == previous.ReviewID {
			s.recordRemovedRequest(previous)
		}
	}
	for _, activity := range s.RequestActivity {
		if activity.SHA != s.Configuration.AssessedSHA || activity.Action != "review_request_removed" || activity.At.Before(s.ObservedAt.Truncate(time.Second)) {
			continue
		}
		if slices.ContainsFunc(current, func(request observedReviewRequest) bool { return request.Reviewer.ID == activity.Reviewer.ID }) {
			continue
		}
		// A delayed removal notification must not hide a newer submission.
		lastSubmission := time.Time{}
		for _, review := range reviews {
			if review.GetUser().GetID() == activity.Reviewer.ID && review.GetState() != "PENDING" && review.GetSubmittedAt().After(lastSubmission) {
				lastSubmission = review.GetSubmittedAt().Time
			}
		}
		// GitHub can emit removal after the submission timestamp. Only a
		// subsequent assignment proves that this removal was manual.
		if !lastSubmission.IsZero() && !lastSubmission.Before(activity.RequestedAt) {
			continue
		}
		if lastSubmission.Before(activity.At) {
			s.recordRemovedRequest(observedReviewRequest{Reviewer: activity.Reviewer, ReviewID: reviewerSubmittedReviewID(reviews, activity.Reviewer)})
		}
	}
	s.RemovedRequests = slices.DeleteFunc(s.RemovedRequests, func(removed observedReviewRequest) bool {
		latest := reviewerSubmittedReviewID(reviews, removed.Reviewer)
		return latest != 0 && latest != removed.ReviewID
	})
	if len(current) > 0 {
		s.RemovedRequests = nil
	}
	s.Requests, s.ObservedAt = current, time.Now().UTC()
}

func (s *reviewCheckState) recordRemovedRequest(request observedReviewRequest) {
	if slices.ContainsFunc(s.RemovedRequests, func(removed observedReviewRequest) bool { return removed.Reviewer.ID == request.Reviewer.ID }) {
		return
	}
	s.RemovedRequests = append(s.RemovedRequests, request)
}

func permanentCheckError(err error) bool {
	switch common.StatusCode(err) {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}

func retryPullRequestCheckSetup(store core.PublicationStore, snapshot *core.OrderedPublication, setupErr error) error {
	// Each publication update replaces the hook request and its retry counter.
	// Keep setup failures with the desired state to bound missing-credential retries.
	stopped := false
	err := store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		state, err := decodeReviewCheckState(current.Payload)
		if err != nil {
			return err
		}
		state.SetupFailures++
		current.WakeAt = reviewCheckWake(time.Minute)
		if state.SetupFailures >= 10 {
			state.Error = "The GitHub integration is unavailable. Check its configuration and retry this step."
			current.WakeAt, stopped = nil, true
		}
		current.Payload, err = json.Marshal(state)
		return err
	})
	if err != nil || stopped {
		return err
	}
	return setupErr
}

func stopPullRequestCheck(store core.PublicationStore, snapshot *core.OrderedPublication, message string) error {
	return store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		state, err := decodeReviewCheckState(current.Payload)
		if err != nil {
			return err
		}
		state.Error = message
		current.WakeAt = nil
		current.Payload, err = json.Marshal(state)
		return err
	})
}

func reviewCheckStore(integration core.IntegrationContext) (core.PublicationStore, error) {
	owner, ok := integration.(core.PublicationIntegration)
	if !ok {
		return nil, fmt.Errorf("integration does not support durable publications")
	}
	return owner.Publications(), nil
}

// WakeReviewChecks is called only after signature verification by the app or node
// webhook handler. Provider reads remain authoritative for review decisions.
func WakeReviewChecks(integration core.IntegrationContext, eventType string, payload []byte) error {
	if eventType != "pull_request" && eventType != "pull_request_review" {
		return nil
	}
	var event struct {
		Review            *github.PullRequestReview `json:"review"`
		Action            string                    `json:"action"`
		RequestedReviewer *github.User              `json:"requested_reviewer"`
		Repository        struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		PullRequest struct {
			Number int `json:"number"`
			Head   struct {
				SHA string `json:"sha"`
			} `json:"head"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"pull_request"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}
	if event.Repository.FullName == "" || event.PullRequest.Number <= 0 {
		return nil
	}
	store, err := reviewCheckStore(integration)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%s%s/%d/", reviewCheckPrefix, strings.ToLower(event.Repository.FullName), event.PullRequest.Number)
	records, err := store.List(prefix)
	if err != nil {
		return err
	}
	for _, record := range records {
		if err := store.Update(record.Key, func(current *core.OrderedPublication) error {
			if current.Disabled {
				return core.ErrPublicationUnchanged
			}
			if eventType == "pull_request_review" && event.Review.GetID() > 0 && event.Review.GetUser().GetID() > 0 && event.Review.GetCommitID() != "" && slices.Contains([]string{"APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED"}, event.Review.GetState()) {
				state, err := decodeReviewCheckState(current.Payload)
				if err != nil {
					return err
				}
				activity := verifiedReviewActivity{ID: event.Review.GetID(), Reviewer: ReviewerIdentity{ID: event.Review.GetUser().GetID(), Login: event.Review.GetUser().GetLogin()}, SHA: event.Review.GetCommitID(), State: event.Review.GetState()}
				index := slices.IndexFunc(state.ReviewActivity, func(previous verifiedReviewActivity) bool { return previous.ID == activity.ID })
				if index < 0 {
					state.ReviewActivity = append(state.ReviewActivity, activity)
				} else if activity.State == "DISMISSED" {
					state.ReviewActivity[index] = activity
				}
				current.Payload, err = json.Marshal(state)
				if err != nil {
					return err
				}
			}
			if eventType == "pull_request" && slices.Contains([]string{"review_requested", "review_request_removed"}, event.Action) && event.RequestedReviewer.GetID() > 0 && !event.PullRequest.UpdatedAt.IsZero() {
				state, err := decodeReviewCheckState(current.Payload)
				if err != nil {
					return err
				}
				activity := reviewRequestActivity{Reviewer: ReviewerIdentity{ID: event.RequestedReviewer.GetID(), Login: event.RequestedReviewer.GetLogin()}, SHA: event.PullRequest.Head.SHA, Action: event.Action, At: event.PullRequest.UpdatedAt}
				index := slices.IndexFunc(state.RequestActivity, func(previous reviewRequestActivity) bool {
					return previous.Reviewer.ID == activity.Reviewer.ID && previous.SHA == activity.SHA
				})
				if activity.Action == "review_requested" {
					activity.RequestedAt = activity.At
				}
				if index < 0 {
					state.RequestActivity = append(state.RequestActivity, activity)
				} else {
					previous := &state.RequestActivity[index]
					if activity.RequestedAt.After(previous.RequestedAt) {
						previous.RequestedAt = activity.RequestedAt
					}
					if activity.At.After(previous.At) {
						activity.RequestedAt = previous.RequestedAt
						*previous = activity
					}
				}
				current.Payload, err = json.Marshal(state)
				if err != nil {
					return err
				}
			}
			current.WakeAt = reviewCheckWake(time.Second)
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
