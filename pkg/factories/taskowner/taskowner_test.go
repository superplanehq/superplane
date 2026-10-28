package taskowner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
)

type fakeHistory map[string][]Commit

func (f fakeHistory) ListPathCommits(_ context.Context, path string, _ time.Time) ([]Commit, error) {
	return f[path], nil
}

type failingHistory struct{}

func (failingHistory) ListPathCommits(context.Context, string, time.Time) ([]Commit, error) {
	return nil, errors.New("rate limited")
}

func TestSuggest(t *testing.T) {
	now := time.Now()
	ana := candidate("Ana", 4, "ana-gh", "ana-old")
	bruno := candidate("Bruno", 0, "bruno")
	carla := candidate("Carla", 1, "carla")
	candidates := []models.TaskOwnerCandidate{ana, bruno, carla}

	t.Run("ranks recent frequent human authors and ignores bots and agent commits", func(t *testing.T) {
		history := fakeHistory{
			"pkg/payments": {
				{SHA: "a1", AuthorLogin: "ANA-OLD", CommittedAt: now.Add(-2 * 24 * time.Hour)},
				{SHA: "a2", AuthorLogin: "ana-gh", CommittedAt: now.Add(-5 * 24 * time.Hour)},
				{SHA: "b1", AuthorLogin: "bruno", CommittedAt: now.Add(-170 * 24 * time.Hour)},
				{SHA: "d1", AuthorLogin: "dependabot[bot]", CommittedAt: now},
				{SHA: "x1", AuthorLogin: "superplane-app", AuthorIsBot: true, CommittedAt: now},
				{SHA: "c1", AuthorLogin: "carla", CommittedAt: now, Message: "Fix\n\nCo-authored-by: SuperPlane Agent <superplaneagent@superplane.com>"},
				{SHA: "s1", AuthorLogin: "stranger", CommittedAt: now},
			},
			"web_src/checkout.tsx": {
				{SHA: "a1", AuthorLogin: "ana-old", CommittedAt: now.Add(-2 * 24 * time.Hour)},
			},
		}

		suggestion, err := Suggest(context.Background(), history, []string{"pkg/payments", "web_src/checkout.tsx"}, candidates, now)
		require.NoError(t, err)

		require.Len(t, suggestion.CodeOwners, 2)
		assert.Equal(t, ana.UserID, suggestion.CodeOwners[0].UserID)
		assert.Equal(t, 2, suggestion.CodeOwners[0].Commits, "a commit that changed two paths counts once")
		assert.Equal(t, []string{"pkg/payments", "web_src/checkout.tsx"}, suggestion.CodeOwners[0].Paths)
		assert.Equal(t, bruno.UserID, suggestion.CodeOwners[1].UserID)

		require.Len(t, suggestion.Members, 3)
		assert.Equal(t, []uuid.UUID{bruno.UserID, carla.UserID, ana.UserID}, memberIDs(suggestion.Members))
	})

	t.Run("lists members without GitHub but never ranks them as code owners", func(t *testing.T) {
		unlinked := models.TaskOwnerCandidate{UserID: uuid.New(), Name: "Dana", Roles: []string{"Operator"}}
		linked := candidate("Eva", 2, "eva")
		linked.Roles = []string{"Admin"}
		history := fakeHistory{"pkg/payments": {
			{SHA: "e1", AuthorLogin: "eva", CommittedAt: now},
			{SHA: "u1", AuthorID: "123", CommittedAt: now},
			{SHA: "u2", CommittedAt: now},
		}}

		suggestion, err := Suggest(context.Background(), history, []string{"pkg/payments"}, []models.TaskOwnerCandidate{unlinked, linked}, now)
		require.NoError(t, err)

		require.Len(t, suggestion.CodeOwners, 1)
		assert.Equal(t, linked.UserID, suggestion.CodeOwners[0].UserID)
		assert.Equal(t, []string{"Admin"}, suggestion.CodeOwners[0].Roles)
		assert.Equal(t, []uuid.UUID{unlinked.UserID, linked.UserID}, memberIDs(suggestion.Members))
		assert.Equal(t, []string{"Operator"}, suggestion.Members[0].Roles)
		assert.Empty(t, suggestion.Members[0].GitHubLogin)
	})

	t.Run("breaks score ties with the fewest open tasks", func(t *testing.T) {
		committedAt := now.Add(-time.Hour)
		history := fakeHistory{"api": {
			{SHA: "1", AuthorLogin: "ana-gh", CommittedAt: committedAt},
			{SHA: "2", AuthorLogin: "carla", CommittedAt: committedAt},
		}}

		suggestion, err := Suggest(context.Background(), history, []string{"api"}, candidates, now)
		require.NoError(t, err)
		require.Len(t, suggestion.CodeOwners, 2)
		assert.Equal(t, carla.UserID, suggestion.CodeOwners[0].UserID)
	})

	t.Run("matches authors by GitHub ID when the login was renamed or reused", func(t *testing.T) {
		renamed := candidate("Dora", 0, "dora-old")
		renamed.GitHubIDs = []string{"42"}
		reused := candidate("Eli", 0, "dora-new")
		reused.GitHubIDs = []string{"77"}
		history := fakeHistory{"api": {
			{SHA: "1", AuthorID: "42", AuthorLogin: "dora-new", CommittedAt: now},
		}}

		suggestion, err := Suggest(context.Background(), history, []string{"api"}, []models.TaskOwnerCandidate{renamed, reused}, now)
		require.NoError(t, err)
		require.Len(t, suggestion.CodeOwners, 1)
		assert.Equal(t, renamed.UserID, suggestion.CodeOwners[0].UserID)
	})

	t.Run("returns members when the history cannot be read", func(t *testing.T) {
		suggestion, err := Suggest(context.Background(), failingHistory{}, []string{"api"}, candidates, now)
		require.Error(t, err)
		assert.Empty(t, suggestion.CodeOwners)
		assert.Len(t, suggestion.Members, 3)
	})
}

func TestNormalizePaths(t *testing.T) {
	paths := NormalizePaths([]string{" ./pkg/api/ ", "/pkg/api", "", "../etc/passwd", ".", "web\\src\\app.tsx"})
	assert.Equal(t, []string{"pkg/api", "web/src/app.tsx"}, paths)

	many := make([]string, 0, MaxPaths+5)
	for i := 0; i < MaxPaths+5; i++ {
		many = append(many, uuid.NewString())
	}
	assert.Len(t, NormalizePaths(many), MaxPaths)
}

func candidate(name string, openTasks int, logins ...string) models.TaskOwnerCandidate {
	return models.TaskOwnerCandidate{
		UserID:       uuid.New(),
		Name:         name,
		GitHubLogin:  logins[0],
		GitHubLogins: logins,
		OpenTasks:    openTasks,
	}
}

func memberIDs(members []Member) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}
