package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
)

// Integration hooks own monitoring after the selection execution has finished.
func refreshReviewCheck(ctx core.IntegrationHookContext) error {
	var parameters struct{ Key string }
	if err := mapstructure.Decode(ctx.Parameters, &parameters); err != nil {
		return err
	}
	if !strings.HasPrefix(parameters.Key, reviewCheckPrefix) {
		return fmt.Errorf("invalid Bitbucket review check key")
	}
	store, err := bitbucketReviewStore(ctx.Integration)
	if err != nil {
		return err
	}
	snapshot, err := store.Get(parameters.Key)
	if err != nil || snapshot == nil {
		return err
	}
	var state reviewCheckState
	if err := json.Unmarshal(snapshot.Payload, &state); err != nil {
		return err
	}
	if !snapshot.Disabled {
		active, err := reviewCheckStillActive(store, snapshot)
		if err != nil {
			return err
		}
		if !active {
			return store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
				if current.Revision != snapshot.Revision {
					return core.ErrPublicationUnchanged
				}
				return disableReviewCheck(current)
			})
		}
	}
	if state.Error != "" {
		return nil
	}
	if state.Delivery != nil && time.Now().Before(state.Delivery.ExpiresAt) {
		return store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
			if current.Revision != snapshot.Revision {
				return core.ErrPublicationUnchanged
			}
			current.WakeAt = &state.Delivery.ExpiresAt
			return nil
		})
	}
	if snapshot.Disabled && state.Delivered != "" && state.Delivered == state.fingerprint() {
		return nil
	}
	client, err := newIntegrationClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return reviewCheckFailure(store, snapshot, err)
	}
	monitor, lookupErr := evaluateReviewCheck(client, store, snapshot, &state)
	if lookupErr != nil {
		state.State, state.Reason, state.Description = "FAILED", "lookup_error", "Could not verify pull request reviews"
		if state.SHA == "" {
			state.SHA = state.Configuration.AssessedSHA
		}
		monitor = true
		state.Failures++
	}
	// Provider reads run without the lock. A second live check fences removal
	// or reactivation while those reads are in flight.
	active := true
	if !snapshot.Disabled {
		active, err = reviewCheckStillActive(store, snapshot)
		if err != nil {
			return err
		}
	}
	var claimed bool
	var revision int64
	err = store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		if !active {
			return disableReviewCheck(current)
		}
		current.WakeAt = nil
		if monitor {
			current.WakeAt = reviewWake(5 * time.Minute)
		}
		if lookupErr != nil && state.Failures >= 10 && state.Delivered == state.fingerprint() {
			state.Error = "Could not verify the check. Check the Bitbucket integration and retry this step."
			current.WakeAt = nil
		}
		if state.Delivery != nil || state.Delivered != state.fingerprint() {
			state.Delivery = &core.PublicationLease{ID: uuid.New(), ExpiresAt: time.Now().Add(time.Minute)}
			state.Delivered = ""
			current.WakeAt = &state.Delivery.ExpiresAt
			claimed = true
			revision = current.Revision + 1
		}
		current.Payload, err = json.Marshal(state)
		return err
	})
	if err != nil || !claimed {
		return err
	}
	// The deadline starts at the claim, so a paused worker cannot start late.
	writeCtx, cancel := context.WithDeadline(context.Background(), state.Delivery.ExpiresAt.Add(-40*time.Second))
	defer cancel()
	writeErr := client.PublishBuildStatus(writeCtx, state.Configuration.apiRepository(), state.SHA, state.Refname, state.Configuration.ResultURL, state.State, state.Description)
	err = store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
		var latest reviewCheckState
		if err := json.Unmarshal(current.Payload, &latest); err != nil {
			return err
		}
		owns := latest.Delivery != nil && latest.Delivery.ID == state.Delivery.ID
		if owns {
			latest.Delivery = nil
		}
		current.WakeAt = reviewWake(5 * time.Minute)
		if owns && current.Generation == snapshot.Generation {
			if writeErr == nil && latest.fingerprint() == state.fingerprint() {
				latest.Delivered = state.fingerprint()
				if !monitor {
					current.WakeAt = nil
				}
			}
			if lookupErr != nil || writeErr != nil {
				if writeErr != nil && lookupErr == nil {
					latest.Failures++
				}
				if latest.Failures >= 10 {
					latest.Error = "Could not verify or publish the check. Check the Bitbucket integration and retry this step."
					current.WakeAt = nil
				}
			} else {
				latest.Failures = 0
			}
		}
		if !owns {
			latest.Delivered = ""
		}
		if !owns || current.Revision != revision {
			current.WakeAt = reviewWake(time.Second)
		}
		current.Payload, err = json.Marshal(latest)
		return err
	})
	if err != nil {
		return err
	}
	if writeErr != nil {
		return writeErr
	}
	return lookupErr
}

func evaluateReviewCheck(client *Client, store core.PublicationStore, record *core.OrderedPublication, state *reviewCheckState) (bool, error) {
	config := state.Configuration
	// Release uses saved commit and branch information, even after PR deletion.
	if record.Disabled && state.Refname != "" {
		state.State, state.Reason, state.Description = "SUCCESSFUL", "disabled", "Review request check disabled"
		return false, nil
	}
	id, err := parsePullRequestID(config.PullRequestID)
	if err != nil {
		return false, err
	}
	readCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pr, err := client.GetPullRequestContext(readCtx, config.apiRepository(), id)
	if err != nil {
		return true, err
	}
	if !IsValidFullCommitSHA(pr.SourceHash) || pr.SourceBr == "" {
		return true, fmt.Errorf("pull request source commit or branch is missing")
	}
	state.SHA, state.Refname = pr.SourceHash, pr.SourceBr
	monitor := pr.State == PullRequestStateOpen
	if record.Disabled {
		state.State, state.Reason, state.Description = "SUCCESSFUL", "disabled", "Review request check disabled"
		return false, nil
	}
	state.Reviewers = nil
	if state.SHA != config.AssessedSHA {
		state.State, state.Reason, state.Description = "INPROGRESS", "head_changed", "Waiting for a decision on the current commit"
		return monitor, nil
	}
	switch config.Phase {
	case "pending_selection":
		finished, err := store.AttemptFinished(record.Order.AttemptRootEventID)
		if err != nil {
			return monitor, err
		}
		state.State, state.Reason, state.Description = "INPROGRESS", "pending_selection", "Selecting required reviewers"
		if finished {
			state.State, state.Reason, state.Description = "FAILED", "attempt_ended", "Reviewer selection ended without a valid decision"
		}
	case "selection_error":
		state.State, state.Reason, state.Description = "FAILED", "selection_error", config.Reason
	case "no_review":
		state.State, state.Reason, state.Description = "SUCCESSFUL", "no_review", config.Reason
	case "required_reviewers":
		evidence, err := ReadApprovalEvidence(store, config.RepositoryUUID, config.PullRequestID)
		if err != nil {
			return monitor, err
		}
		reviews, err := client.ReadPullRequestReviewState(readCtx, config.apiRepository(), id, config.AssessedSHA, config.Reviewers, evidence)
		if err != nil {
			return monitor, err
		}
		state.Reviewers = reviews.Reviewers
		state.State, state.Reason, state.Description = "SUCCESSFUL", "approved", "All required reviewers approved the current commit"
		for _, reviewer := range reviews.Reviewers {
			if reviewer.Reason == "changes_requested" {
				state.State, state.Reason, state.Description = "FAILED", "changes_requested", "A required reviewer requested changes"
				break
			}
			if reviewer.Reason != "approved" {
				state.State, state.Reason, state.Description = "INPROGRESS", reviewer.Reason, "Waiting for approvals with verified commit evidence"
			}
		}
		// Pagination can overlap pushes or revocation. A changed snapshot must
		// be reconciled again, never published as successful.
		current, err := client.GetPullRequestContext(readCtx, config.apiRepository(), id)
		if err != nil {
			return monitor, err
		}
		monitor = current.State == PullRequestStateOpen
		if current.SourceHash != config.AssessedSHA {
			state.SHA, state.Refname = current.SourceHash, current.SourceBr
			state.State, state.Reason, state.Description = "INPROGRESS", "head_changed", "Waiting for a decision on the current commit"
		}
		if !sameReviewParticipants(current.Participants, reviews.PullRequest.Participants) {
			state.State, state.Reason, state.Description = "INPROGRESS", "review_changed", "Waiting for current review state"
		}
	default:
		return monitor, fmt.Errorf("invalid persisted review phase")
	}
	return monitor, nil
}

func sameReviewParticipants(left, right []ReviewParticipant) bool {
	if len(left) != len(right) {
		return false
	}
	for _, a := range left {
		found := false
		for _, b := range right {
			if bitbucketUUID(a.User.UUID) == bitbucketUUID(b.User.UUID) && a.State == b.State && a.Approved == b.Approved {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func reviewCheckFailure(store core.PublicationStore, snapshot *core.OrderedPublication, cause error) error {
	err := store.Update(snapshot.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		var state reviewCheckState
		if err := json.Unmarshal(current.Payload, &state); err != nil {
			return err
		}
		state.Failures++
		current.WakeAt = reviewWake(time.Minute)
		if state.Failures >= 10 {
			state.Error = "The Bitbucket integration is unavailable. Check its configuration and retry this step."
			current.WakeAt = nil
		}
		var err error
		current.Payload, err = json.Marshal(state)
		return err
	})
	if err != nil {
		return err
	}
	return cause
}
