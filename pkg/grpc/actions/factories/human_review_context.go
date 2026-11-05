package factories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v84/github"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/factories/humanreview"
	"github.com/superplanehq/superplane/pkg/models"
)

func gatherHumanReviewContext(ctx context.Context, client humanReviewGitHub, pr *models.FactoryPullRequest, current *github.PullRequest, fallbackTeam string) (map[string]any, error) {
	files, err := client.HumanReviewChangedFiles(ctx, pr.Repository, int(pr.Number))
	if err != nil {
		return nil, fmt.Errorf("read changed files for Human review: %w", err)
	}
	changed := make([]map[string]any, 0, len(files))
	for _, file := range files {
		changed = append(changed, map[string]any{"path": file.GetFilename(), "previousPath": file.GetPreviousFilename(), "status": file.GetStatus(), "additions": file.GetAdditions(), "deletions": file.GetDeletions()})
	}
	ownership := map[string]string{}
	for _, path := range []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"} {
		content, err := client.HumanReviewOwnershipFile(ctx, pr.Repository, path, current.GetBase().GetSHA())
		if err != nil {
			return nil, fmt.Errorf("read target branch ownership: %w", err)
		}
		if content != "" {
			ownership[path] = content
			break
		}
	}
	ownershipTeams := map[string][]humanreview.Reviewer{}
	ownershipCandidates := []humanreview.Reviewer{}
	resolvedOwners := map[string]bool{}
	for _, content := range ownership {
		for line := range strings.SplitSeq(content, "\n") {
			line, _, _ = strings.Cut(line, "#")
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			for _, owner := range fields[1:] {
				if !strings.HasPrefix(owner, "@") {
					continue
				}
				reference := strings.TrimPrefix(owner, "@")
				if !strings.Contains(reference, "/") {
					login := strings.ToLower(reference)
					if resolvedOwners[login] {
						continue
					}
					if len(resolvedOwners) >= 100 {
						return nil, errors.New("target branch ownership contains more than 100 individual reviewers; reduce owner references before running Human review")
					}
					resolvedOwners[login] = true
					permission, _, err := client.GetRepositoryPermissionLevel(ctx, pr.Repository, login)
					if err != nil {
						log.WithError(err).WithField("owner", owner).Warn("Human review: skip unavailable CODEOWNERS hint")
						continue
					}
					if permission == nil || permission.GetUser() == nil {
						continue
					}
					user := permission.GetUser()
					eligible, err := humanReviewerEligible(ctx, client, pr.Repository, user, current.GetUser())
					if err != nil {
						log.WithError(err).WithField("owner", owner).Warn("Human review: skip unavailable CODEOWNERS hint")
						continue
					}
					if eligible {
						ownershipCandidates = append(ownershipCandidates, humanreview.Reviewer{ID: user.GetID(), Login: user.GetLogin(), Reason: "Target branch CODEOWNERS names this user."})
					}
					continue
				}
				team := reference
				if _, resolved := ownershipTeams[team]; resolved {
					continue
				}
				if len(ownershipTeams) >= 100 {
					return nil, errors.New("target branch ownership contains more than 100 teams; reduce team references before running Human review")
				}
				// Cache unavailable teams too, so repeated rules do not repeat failures.
				ownershipTeams[team] = []humanreview.Reviewer{}
				candidates, err := humanReviewFallbackCandidates(ctx, client, pr.Repository, team, current.GetUser())
				if err != nil {
					log.WithError(err).WithField("owner", owner).Warn("Human review: skip unavailable CODEOWNERS hint")
					continue
				}
				ownershipTeams[team] = candidates
			}
		}
	}
	candidates, err := humanReviewFallbackCandidates(ctx, client, pr.Repository, fallbackTeam, current.GetUser())
	if err != nil {
		return nil, err
	}
	existing := make([]map[string]any, 0, len(current.RequestedReviewers))
	for _, user := range current.RequestedReviewers {
		eligible, err := humanReviewerEligible(ctx, client, pr.Repository, user, current.GetUser())
		if err != nil {
			return nil, err
		}
		existing = append(existing, map[string]any{"id": user.GetID(), "login": user.GetLogin(), "eligible": eligible})
	}
	existingTeams := []map[string]any{}
	for _, team := range current.RequestedTeams {
		organization := team.GetOrganization().GetLogin()
		if organization == "" {
			organization, _, _ = strings.Cut(pr.Repository, "/")
		}
		candidates, err := humanReviewFallbackCandidates(ctx, client, pr.Repository, organization+"/"+team.GetSlug(), current.GetUser())
		if err != nil {
			return nil, fmt.Errorf("resolve existing team review request: %w", err)
		}
		existingTeams = append(existingTeams, map[string]any{"id": team.GetID(), "organization": organization, "slug": team.GetSlug(), "eligibleMembers": candidates})
	}
	// Read recent commits separately for affected files. This lets the agent weigh
	// ownership against actual contributions rather than repository-wide authors.
	contributions := []map[string]any{}
	for i, file := range files {
		if i >= 30 {
			break
		}
		commits, _, err := client.ListCommits(ctx, pr.Repository, &github.CommitsListOptions{SHA: current.GetBase().GetSHA(), Path: file.GetFilename(), Since: time.Now().AddDate(0, -3, 0), ListOptions: github.ListOptions{PerPage: 10}})
		if err != nil {
			return nil, fmt.Errorf("read affected-file contributions: %w", err)
		}
		for _, commit := range commits {
			if commit.GetAuthor() == nil {
				continue
			}
			contributions = append(contributions, map[string]any{"path": file.GetFilename(), "id": commit.GetAuthor().GetID(), "login": commit.GetAuthor().GetLogin(), "commit": commit.GetSHA()})
		}
	}
	return map[string]any{"changedFiles": changed, "existingRequests": existing, "existingTeamRequests": existingTeams, "ownershipFiles": ownership, "ownershipCandidates": ownershipCandidates, "ownershipTeamCandidates": ownershipTeams, "recentContributions": contributions, "contributionHistoryFileLimit": 30, "fallbackCandidates": candidates}, nil
}

func humanReviewFallbackCandidates(ctx context.Context, client humanReviewGitHub, repository, fallbackTeam string, author *github.User) ([]humanreview.Reviewer, error) {
	organization, slug, valid := strings.Cut(strings.TrimSpace(fallbackTeam), "/")
	if !valid || organization == "" || slug == "" {
		return nil, errors.New("select a valid fallback GitHub team in the Human review settings")
	}
	members, err := client.HumanReviewTeamMembers(ctx, organization, slug)
	if err != nil {
		return nil, fmt.Errorf("read fallback team members: %w", err)
	}
	candidates := make([]humanreview.Reviewer, 0, len(members))
	for _, member := range members {
		eligible, err := humanReviewerEligible(ctx, client, repository, member, author)
		if err != nil {
			return nil, err
		}
		if eligible {
			candidates = append(candidates, humanreview.Reviewer{ID: member.GetID(), Login: member.GetLogin(), Reason: "Eligible member of the configured fallback team."})
		}
	}
	return candidates, nil
}

func humanReviewerEligible(ctx context.Context, client humanReviewGitHub, repository string, user, author *github.User) (bool, error) {
	if user == nil || user.GetID() == 0 || user.GetType() != "User" || strings.HasSuffix(strings.ToLower(user.GetLogin()), "[bot]") || strings.EqualFold(user.GetLogin(), author.GetLogin()) || user.GetID() == author.GetID() {
		return false, nil
	}
	permission, _, err := client.GetRepositoryPermissionLevel(ctx, repository, user.GetLogin())
	if err != nil {
		return false, fmt.Errorf("check repository access for %s: %w", user.GetLogin(), err)
	}
	if permission == nil {
		return false, nil
	}
	switch permission.GetPermission() {
	case "admin", "maintain", "write":
		return true, nil
	default:
		return false, nil
	}
}
