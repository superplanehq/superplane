package factories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/go-github/v84/github"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func evaluateHumanReviewDecision(ctx context.Context, db *gorm.DB, client humanReviewGitHub, pr *models.FactoryPullRequest, current *github.PullRequest, automation *humanReviewAutomation, decision *models.FactoryHumanReviewDecision) error {
	priorRationale, priorBlock := decision.Rationale, decision.BlockingReason
	var result humanreview.Decision
	encoded, err := json.Marshal(decision.Result)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(encoded, &result); err == nil {
		err = humanreview.ValidateDecision(result, decision.AssessedCommit)
	}
	if err != nil {
		decision.State = models.FactoryHumanReviewStateFailed
		decision.BlockingReason = "The Request review decision is invalid: " + err.Error()
		if err := decision.Save(db); err != nil {
			return err
		}
		if priorBlock != decision.BlockingReason {
			notifyHumanReviewDecision(db, pr, decision)
		}
		return publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: "completed", Conclusion: "failure", Reason: decision.BlockingReason})
	}
	reviews, err := client.ListReviews(ctx, pr.Repository, int(pr.Number))
	if err != nil {
		return fmt.Errorf("read current GitHub reviews: %w", err)
	}
	eligible := map[int64]bool{}
	users := map[int64]*github.User{}
	for _, review := range reviews {
		users[review.GetUser().GetID()] = review.GetUser()
	}
	for _, requested := range current.RequestedReviewers {
		users[requested.GetID()] = requested
	}
	selected := make([]humanreview.Reviewer, 0, len(result.SelectedReviewers))
	for _, reviewer := range result.SelectedReviewers {
		permission, _, err := client.GetRepositoryPermissionLevel(ctx, pr.Repository, reviewer.Login)
		if err != nil {
			return fmt.Errorf("validate selected reviewer %s: %w", reviewer.Login, err)
		}
		if permission == nil || permission.GetUser() == nil {
			continue
		}
		user := permission.GetUser()
		if reviewer.ID != 0 && reviewer.ID != user.GetID() {
			continue
		}
		allowed, err := humanReviewerEligible(ctx, client, pr.Repository, user, current.GetUser())
		if err != nil {
			return err
		}
		users[user.GetID()] = user
		eligible[user.GetID()] = allowed
		if !allowed {
			continue
		}
		reviewer.ID = user.GetID()
		reviewer.Login = user.GetLogin()
		selected = append(selected, reviewer)
	}
	if *result.ReviewRequired && len(selected) == 0 {
		fallback, err := humanReviewConfiguredFallbackCandidates(ctx, client, pr.Repository, automation.settings, current.GetUser())
		if err != nil {
			return err
		}
		if len(fallback) > 0 {
			selected = append(selected, fallback[0])
			eligible[fallback[0].ID] = true
		}
	}
	result.SelectedReviewers = selected
	for id, user := range users {
		if _, known := eligible[id]; known {
			continue
		}
		allowed, err := humanReviewerEligible(ctx, client, pr.Repository, user, current.GetUser())
		if err != nil {
			return err
		}
		eligible[id] = allowed
	}
	decisions := make([]models.FactoryHumanReviewReviewer, 0, len(selected))
	for _, reviewer := range selected {
		decisions = append(decisions, models.FactoryHumanReviewReviewer{ID: reviewer.ID, Login: reviewer.Login, Reason: reviewer.Reason})
	}
	decision.SelectedReviewers = datatypes.NewJSONSlice(decisions)
	decision.ReviewRequired = *result.ReviewRequired
	decision.Rationale = result.Rationale
	decision.State = models.FactoryHumanReviewStateDecided
	providerReviews := make([]humanreview.Review, 0, len(reviews))
	for _, review := range reviews {
		providerReviews = append(providerReviews, humanreview.Review{ID: review.GetID(), UserID: review.GetUser().GetID(), State: review.GetState(), CommitID: review.GetCommitID(), SubmittedAt: review.GetSubmittedAt().Time})
	}
	gate := humanreview.EvaluateGate(humanreview.GateInput{Enabled: true, AssessmentValid: true, DecisionValid: true, HeadSHA: decision.AssessedCommit, Author: current.GetUser().GetLogin(), Decision: result, Eligible: eligible, Reviews: providerReviews})
	decision.ApprovalIDs = datatypes.NewJSONSlice(gate.ApprovalIDs)
	if *result.ReviewRequired && len(selected) == 0 {
		gate.Reason = "The fallback GitHub team has no eligible human reviewer. Add a human member with repository write access, then rerun Request review."
		if automation.settings.DefaultReviewer != "" {
			gate.Reason = "The default reviewer cannot approve this pull request. Select another human with repository write access, then rerun Request review."
		}
	}
	decision.BlockingReason = gate.Reason
	if gate.Conclusion == "success" {
		decision.BlockingReason = ""
	}
	if err := decision.Save(db); err != nil {
		return err
	}
	if priorRationale != decision.Rationale || priorBlock != decision.BlockingReason {
		notifyHumanReviewDecision(db, pr, decision)
	}
	publicationErr := publishHumanReviewCheck(ctx, db, client, pr, decision, humanReviewPublication{Status: gate.Status, Conclusion: gate.Conclusion, Reason: gate.Reason})
	if *result.ReviewRequired {
		missing := humanReviewMissingRequests(selected, current.RequestedReviewers, reviews, decision.AssessedCommit)
		if len(missing) > 0 {
			latest, _, err := client.GetPullRequest(ctx, pr.Repository, int(pr.Number))
			if err != nil {
				return err
			}
			if latest == nil || latest.GetState() != "open" || latest.GetMerged() || latest.GetHead().GetSHA() != decision.AssessedCommit {
				return nil
			}
			active, err := humanReviewPublicationCurrent(db, pr, decision)
			if err != nil || !active {
				return err
			}
			if _, _, err := client.AddPullRequestReviewers(ctx, pr.Repository, int(pr.Number), github.ReviewersRequest{Reviewers: missing}); err != nil {
				return errors.Join(publicationErr, recordHumanReviewPublicationError(db, decision, fmt.Errorf("request human reviewers: %w", err)))
			}
		}
	}
	return publicationErr
}

func humanReviewMissingRequests(selected []humanreview.Reviewer, requested []*github.User, reviews []*github.PullRequestReview, sha string) []string {
	existing := map[int64]bool{}
	for _, user := range requested {
		existing[user.GetID()] = true
	}
	latest := map[int64]*github.PullRequestReview{}
	for _, review := range reviews {
		state := strings.ToUpper(review.GetState())
		if state != "APPROVED" && state != "CHANGES_REQUESTED" && state != "DISMISSED" {
			continue
		}
		id := review.GetUser().GetID()
		prior := latest[id]
		if prior == nil || review.GetSubmittedAt().After(prior.GetSubmittedAt().Time) || (review.GetSubmittedAt().Equal(prior.GetSubmittedAt()) && review.GetID() > prior.GetID()) {
			latest[id] = review
		}
	}
	for id, review := range latest {
		if strings.EqualFold(review.GetState(), "APPROVED") && review.GetCommitID() == sha {
			existing[id] = true
		}
	}

	var missing []string
	for _, reviewer := range selected {
		if !existing[reviewer.ID] {
			missing = append(missing, reviewer.Login)
		}
	}
	return missing
}
