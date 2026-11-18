package pulls

import (
	"context"
	"fmt"
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
	headSHA  string
	onList   func()
}

func (c *fakeCommentClient) Read(context.Context, string, int64) (prbody.Description, error) {
	if c.headSHA != "" {
		return prbody.Description{Open: true, HeadSHA: c.headSHA}, nil
	}
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
	if c.onList != nil {
		c.onList()
	}
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

// The comment lookup can read many pages. A change during the lookup must
// stop the create or edit.
func TestPublishCommentRechecksAfterLookup(t *testing.T) {
	canvas := uuid.New()
	config := PullRequestCommentConfiguration{Repository: "acme/app", PullNumber: "42", CommentKey: "review-request", Body: "New findings", ExpectedHeadSHA: "abc"}
	marker := config.Marker(canvas)
	for _, scenario := range []string{"publication stopped", "commit pushed"} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing=%t", scenario, existing), func(t *testing.T) {
				client := &fakeCommentClient{comments: map[int64]string{}}
				if existing {
					client.comments[1] = marker + "\nOld findings"
				}
				active := true
				client.onList = func() {
					if scenario == "publication stopped" {
						active = false
						return
					}
					client.headSHA = "def"
				}
				boundary := &fakeCommentBoundary{}
				outcome, err := PublishComment(t.Context(), client, config, marker, func() (bool, error) { return active, nil }, boundary)
				require.NoError(t, err)
				require.Equal(t, prbody.StatusSkipped, outcome.Status)
				require.Zero(t, client.created)
				require.Empty(t, client.edited)
				if existing {
					require.Equal(t, marker+"\nOld findings", client.comments[1])
					require.Equal(t, int64(1), boundary.finished)
				}
			})
		}
	}
}
