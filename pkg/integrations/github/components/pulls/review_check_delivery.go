package pulls

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

// RefreshReviewCheck runs as an integration hook, after the owning execution
// transaction commits. Lookups do not hold the per-requirement publication lock.
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
	state, err := decodeReviewCheckState(snapshot.Payload)
	if err != nil {
		return err
	}
	if snapshot.Disabled && state.Delivered != "" && state.Delivered == state.fingerprint() {
		return nil
	}
	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return err
	}
	monitor, lookupErr := evaluateReviewCheck(client, store, snapshot, &state)
	if lookupErr != nil {
		if snapshot.Disabled {
			return lookupErr
		}
		state.State, state.Description = "error", "Could not verify pull request reviews"
		if state.SHA == "" {
			state.SHA = state.Configuration.AssessedSHA
		}
		monitor = true
	}
	// Commit desired state and recovery work before attempting external delivery.
	var revision int64
	err = store.Update(parameters.Key, func(current *core.OrderedPublication) error {
		if current.Revision != snapshot.Revision {
			return core.ErrPublicationUnchanged
		}
		current.Payload, err = json.Marshal(state)
		if err != nil {
			return err
		}
		current.WakeAt = reviewCheckWake(5 * time.Minute)
		revision = current.Revision + 1
		return nil
	})
	if err != nil || revision == 0 {
		return err
	}
	err = store.Update(parameters.Key, func(current *core.OrderedPublication) error {
		if current.Revision != revision {
			return core.ErrPublicationUnchanged
		}
		if state.Delivered != state.fingerprint() {
			// Bound the only provider write held under this requirement's lock.
			writeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			description := string([]rune(state.Description)[:min(len([]rune(state.Description)), 140)])
			_, _, err := client.CreateStatus(writeCtx, state.Configuration.Repository, state.SHA, github.RepoStatus{State: &state.State, Context: &state.Configuration.Context, Description: &description, TargetURL: &state.Configuration.ResultURL})
			if err != nil {
				return fmt.Errorf("publish review check: %w", err)
			}
		}
		state.Delivered = state.fingerprint()
		current.Payload, err = json.Marshal(state)
		if err != nil {
			return err
		}
		if !monitor {
			current.WakeAt = nil
		}
		return nil
	})
	if err != nil {
		return err
	}
	return lookupErr
}

func evaluateReviewCheck(client *common.Client, store core.PublicationStore, record *core.OrderedPublication, state *reviewCheckState) (bool, error) {
	config := state.Configuration
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
		state.State, state.Description = "success", "All required reviewers approved the current commit"
		for _, reviewer := range config.Reviewers {
			latest := latestReviewerDecision(reviews, reviewer)
			if latest != nil && latest.GetState() == "CHANGES_REQUESTED" {
				state.State, state.Description = "failure", "A required reviewer requested changes"
				break
			}
			if !hasCurrentApproval(reviews, reviewer, config.AssessedSHA) {
				state.State, state.Description = "pending", "Waiting for all required approvals"
			}
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
		}
	default:
		return monitor, fmt.Errorf("invalid persisted review check phase")
	}
	return monitor, nil
}

func reviewCheckStore(integration core.IntegrationContext) (core.PublicationStore, error) {
	owner, ok := integration.(core.PublicationIntegration)
	if !ok {
		return nil, fmt.Errorf("integration does not support durable publications")
	}
	return owner.Publications(), nil
}

// WakeReviewChecks is called only after signature verification by the app or node
// webhook handler. It schedules refresh; it never interprets review payload states.
func WakeReviewChecks(integration core.IntegrationContext, eventType string, payload []byte) error {
	if eventType != "pull_request" && eventType != "pull_request_review" {
		return nil
	}
	var event struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		PullRequest struct {
			Number int `json:"number"`
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
			current.WakeAt = reviewCheckWake(time.Second)
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
