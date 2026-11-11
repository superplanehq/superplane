package prbody

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type publicationClient struct {
	onRead func()
	writes int
}

func (c *publicationClient) Read(context.Context, string, int64) (Description, error) {
	c.onRead()
	return Description{Open: true, HeadSHA: "abc", Body: "<!-- superplane:notes:start -->\nReport\n<!-- superplane:notes:end -->"}, nil
}

func (c *publicationClient) Write(context.Context, string, int64, string) error {
	c.writes++
	return nil
}

func TestPublishPropagatesEligibilityErrorAfterIdenticalRead(t *testing.T) {
	lookupError := errors.New("eligibility lookup failed")
	var eligibilityError error
	client := &publicationClient{onRead: func() { eligibilityError = lookupError }}
	_, err := Publish(t.Context(), client, Configuration{
		Repository: "acme/app", PullNumber: "42", SectionKey: "notes", Body: "Report", ExpectedHeadSHA: "abc",
	}, func() (bool, error) { return true, eligibilityError })
	require.ErrorIs(t, err, lookupError)
	assert.Zero(t, client.writes)
}
