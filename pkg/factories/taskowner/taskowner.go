// Package taskowner ranks the members who should own a task. It reads the
// code history of the paths the task touches and maps commit authors to
// members by their associated GitHub account.
package taskowner

import (
	"cmp"
	"context"
	"math"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
)

const (
	MaxPaths      = 10
	HistoryWindow = 180 * 24 * time.Hour

	// A commit loses half of its weight every 30 days, so the person who
	// works on the code now outranks the person who wrote it long ago.
	recencyHalfLife = 30 * 24 * time.Hour

	// agentCoAuthorEmail marks commits the SuperPlane agent wrote. The
	// person who merged such a commit did not write the code.
	agentCoAuthorEmail = "superplaneagent@superplane.com"
)

// Commit is one commit that changed a path. AuthorID is the numeric GitHub
// user ID. It is empty when the history source does not know the account.
type Commit struct {
	SHA         string
	AuthorID    string
	AuthorLogin string
	AuthorIsBot bool
	Message     string
	CommittedAt time.Time
}

// History lists the recent commits that changed a repository path.
type History interface {
	ListPathCommits(ctx context.Context, path string, since time.Time) ([]Commit, error)
}

// CodeOwner is a member who changed the paths, ranked by Score.
type CodeOwner struct {
	UserID       uuid.UUID `json:"user_id"`
	Name         string    `json:"name"`
	GitHubLogin  string    `json:"github_login"`
	Roles        []string  `json:"roles"`
	Commits      int       `json:"commits"`
	LastCommitAt time.Time `json:"last_commit_at"`
	Paths        []string  `json:"paths"`
	OpenTasks    int       `json:"open_tasks"`
	Score        float64   `json:"score"`
}

// Member is a member who can own the task, with their open task count.
// GitHubLogin is empty when the member has no GitHub account.
type Member struct {
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	GitHubLogin string    `json:"github_login"`
	Roles       []string  `json:"roles"`
	OpenTasks   int       `json:"open_tasks"`
}

// Suggestion is the ranked result for one task.
type Suggestion struct {
	CodeOwners []CodeOwner `json:"code_owners"`
	Members    []Member    `json:"members"`
	// HistoryNote explains why CodeOwners is empty when the code history
	// could not be read.
	HistoryNote string `json:"history_note,omitempty"`
}

// NormalizePaths cleans the repository paths the agent sends. It drops empty
// paths and paths that leave the repository, removes duplicates, and keeps at
// most MaxPaths.
func NormalizePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, raw := range paths {
		cleaned := strings.ReplaceAll(strings.TrimSpace(raw), "\\", "/")
		if cleaned == "" || slices.Contains(strings.Split(cleaned, "/"), "..") {
			continue
		}
		cleaned = strings.TrimPrefix(path.Clean("/"+cleaned), "/")
		if cleaned == "" {
			continue
		}
		if slices.Contains(normalized, cleaned) {
			continue
		}
		normalized = append(normalized, cleaned)
		if len(normalized) == MaxPaths {
			break
		}
	}
	return normalized
}

// Members lists the candidates by open task count, fewest first.
func Members(candidates []models.TaskOwnerCandidate) []Member {
	members := make([]Member, 0, len(candidates))
	for _, candidate := range candidates {
		members = append(members, MemberOf(candidate))
	}
	slices.SortStableFunc(members, func(left, right Member) int {
		if byLoad := cmp.Compare(left.OpenTasks, right.OpenTasks); byLoad != 0 {
			return byLoad
		}
		return cmp.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
	})
	return members
}

// MemberOf describes one candidate.
func MemberOf(candidate models.TaskOwnerCandidate) Member {
	return Member{
		UserID:      candidate.UserID,
		Name:        candidate.Name,
		Email:       candidate.Email,
		GitHubLogin: candidate.GitHubLogin,
		Roles:       nonNilRoles(candidate.Roles),
		OpenTasks:   candidate.OpenTasks,
	}
}

func nonNilRoles(roles []string) []string {
	if roles == nil {
		return []string{}
	}
	return roles
}

// Suggest ranks the candidates who changed the paths. A commit that changed
// several paths counts once. Bots, agent commits, and authors who are not
// candidates do not count.
func Suggest(ctx context.Context, history History, paths []string, candidates []models.TaskOwnerCandidate, now time.Time) (Suggestion, error) {
	suggestion := Suggestion{CodeOwners: []CodeOwner{}, Members: Members(candidates)}
	if history == nil || len(paths) == 0 || len(candidates) == 0 {
		return suggestion, nil
	}

	owners := map[uuid.UUID]*CodeOwner{}
	seen := map[string]bool{}
	since := now.Add(-HistoryWindow)
	for _, repositoryPath := range paths {
		commits, err := history.ListPathCommits(ctx, repositoryPath, since)
		if err != nil {
			return suggestion, err
		}
		for _, commit := range commits {
			candidate, ok := commitCandidate(commit, candidates)
			if !ok {
				continue
			}
			owner := owners[candidate.UserID]
			if owner == nil {
				owner = &CodeOwner{
					UserID:      candidate.UserID,
					Name:        candidate.Name,
					GitHubLogin: candidate.GitHubLogin,
					Roles:       nonNilRoles(candidate.Roles),
					OpenTasks:   candidate.OpenTasks,
					Paths:       []string{},
				}
				owners[candidate.UserID] = owner
			}
			if !slices.Contains(owner.Paths, repositoryPath) {
				owner.Paths = append(owner.Paths, repositoryPath)
			}
			if commit.SHA != "" {
				if seen[commit.SHA] {
					continue
				}
				seen[commit.SHA] = true
			}
			owner.Commits++
			owner.Score += recencyWeight(commit.CommittedAt, now)
			if commit.CommittedAt.After(owner.LastCommitAt) {
				owner.LastCommitAt = commit.CommittedAt
			}
		}
	}

	for _, owner := range owners {
		owner.Score = math.Round(owner.Score*100) / 100
		suggestion.CodeOwners = append(suggestion.CodeOwners, *owner)
	}
	slices.SortFunc(suggestion.CodeOwners, compareCodeOwners)
	return suggestion, nil
}

func commitCandidate(commit Commit, candidates []models.TaskOwnerCandidate) (models.TaskOwnerCandidate, bool) {
	if commit.AuthorIsBot || isBotLogin(commit.AuthorLogin) || hasAgentCoAuthor(commit.Message) {
		return models.TaskOwnerCandidate{}, false
	}
	// A login can be renamed and then taken by another person, so the
	// login only identifies the author when the account ID is unknown.
	for _, candidate := range candidates {
		if commit.AuthorID != "" && candidate.HasGitHubID(commit.AuthorID) {
			return candidate, true
		}
		if commit.AuthorID == "" && candidate.HasGitHubLogin(commit.AuthorLogin) {
			return candidate, true
		}
	}
	return models.TaskOwnerCandidate{}, false
}

func compareCodeOwners(left, right CodeOwner) int {
	if byScore := cmp.Compare(right.Score, left.Score); byScore != 0 {
		return byScore
	}
	if byLoad := cmp.Compare(left.OpenTasks, right.OpenTasks); byLoad != 0 {
		return byLoad
	}
	return cmp.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name))
}

func recencyWeight(committedAt, now time.Time) float64 {
	age := now.Sub(committedAt)
	if age < 0 {
		age = 0
	}
	return math.Pow(0.5, float64(age)/float64(recencyHalfLife))
}

func isBotLogin(login string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(login)), "[bot]")
}

func hasAgentCoAuthor(message string) bool {
	for _, line := range strings.Split(message, "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(line, "co-authored-by:") && strings.Contains(line, agentCoAuthorEmail) {
			return true
		}
	}
	return false
}
