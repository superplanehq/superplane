package pulls

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/integrations/prbody"
)

type fakeCommentClient struct {
	comments map[int64]string
	edited   []int64
	created  int
}

func (c *fakeCommentClient) Read(context.Context, string, int64) (prbody.Description, error) {
	return prbody.Description{Open: true, HeadSHA: "abc"}, nil
}
func (c *fakeCommentClient) Get(_ context.Context, _ string, id int64) (*ManagedComment, error) {
	body, ok := c.comments[id]
	if !ok {
		return nil, nil
	}
	return &ManagedComment{ID: id, Body: body}, nil
}
func (c *fakeCommentClient) List(context.Context, string, int64) ([]ManagedComment, error) {
	comments := []ManagedComment{}
	for id := int64(1); id <= int64(len(c.comments))+1; id++ {
		if body, ok := c.comments[id]; ok {
			comments = append(comments, ManagedComment{ID: id, Body: body})
		}
	}
	return comments, nil
}
func (c *fakeCommentClient) Create(_ context.Context, _ string, _ int64, body string) (int64, error) {
	c.created++
	id := int64(len(c.comments) + 1)
	c.comments[id] = body
	return id, nil
}
func (c *fakeCommentClient) Edit(_ context.Context, _ string, id int64, body string) error {
	c.edited = append(c.edited, id)
	c.comments[id] = body
	return nil
}

type fakeCommentBoundary struct {
	saved    int64
	finished int64
}

func (b *fakeCommentBoundary) Begin() (int64, time.Time, error) {
	return b.saved, time.Now().Add(time.Minute), nil
}
func (b *fakeCommentBoundary) Finish(id int64) error { b.finished = id; return nil }

// A saved ID must not authorize edits after someone removes the ownership marker.
func TestPublishCommentEditsOnlyMarkedComments(t *testing.T) {
	canvas := uuid.New()
	config := PullRequestCommentConfiguration{Repository: "acme/app", PullNumber: "42", CommentKey: "review-request", Body: "New findings", ExpectedHeadSHA: "abc"}
	marker := config.Marker(canvas)
	for _, scenario := range []string{"marker removed", "other automation"} {
		t.Run(scenario, func(t *testing.T) {
			client := &fakeCommentClient{comments: map[int64]string{1: "User reply quoting the old findings", 2: marker + "\nOld findings"}}
			if scenario == "other automation" {
				client.comments[2] = config.Marker(uuid.New()) + "\nOther findings"
			}
			boundary := &fakeCommentBoundary{saved: 1}
			outcome, err := PublishComment(t.Context(), client, config, marker, func() (bool, error) { return true, nil }, boundary)
			require.NoError(t, err)
			require.Equal(t, prbody.StatusPublished, outcome.Status)
			require.Equal(t, "User reply quoting the old findings", client.comments[1])
			if scenario == "marker removed" {
				require.Equal(t, []int64{2}, client.edited)
				require.Zero(t, client.created)
			} else {
				require.Empty(t, client.edited)
				require.Equal(t, 1, client.created)
				require.True(t, strings.HasSuffix(client.comments[2], "\nOther findings"))
			}
			require.Equal(t, outcome.CommentID, boundary.finished)
			require.Equal(t, marker+"\nNew findings", client.comments[outcome.CommentID])
		})
	}
}
