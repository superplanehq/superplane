package prbody

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
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

type memoryPublications struct {
	records map[string]core.OrderedPublication
}

func (m *memoryPublications) Get(key string) (*core.OrderedPublication, error) {
	record, ok := m.records[key]
	if !ok {
		return nil, nil
	}
	return &record, nil
}
func (m *memoryPublications) List(string) ([]core.OrderedPublication, error) { return nil, nil }
func (m *memoryPublications) Update(key string, apply func(*core.OrderedPublication) error) error {
	record := m.records[key]
	record.Key = key
	if err := apply(&record); err != nil {
		return err
	}
	m.records[key] = record
	return nil
}
func (m *memoryPublications) AttemptFinished(uuid.UUID) (bool, error) { return true, nil }
func (m *memoryPublications) AttemptEnded(uuid.UUID) (bool, error)    { return true, nil }
func (m *memoryPublications) OwnerStatus(uuid.UUID, string) (core.PublicationOwnerStatus, error) {
	return core.PublicationOwnerStatus{Available: true}, nil
}
func (m *memoryPublications) NodeSnapshot(uuid.UUID, string, uuid.UUID) (*core.PublicationNode, error) {
	return nil, nil
}

type descriptionHost struct {
	body   string
	onRead func()
}

func (h *descriptionHost) Read(context.Context, string, int64) (Description, error) {
	body := h.body
	if h.onRead != nil {
		onRead := h.onRead
		h.onRead = nil
		onRead()
	}
	return Description{Open: true, HeadSHA: "abc", Body: body}, nil
}

func (h *descriptionHost) Write(_ context.Context, _ string, _ int64, body string) error {
	h.body = body
	return nil
}

// Two sections that read the same description must not both write, or the
// last write removes the other section.
func TestPublishSerializedKeepsConcurrentSections(t *testing.T) {
	store := &memoryPublications{records: map[string]core.OrderedPublication{}}
	host := &descriptionHost{body: "Intro"}
	section := func(key, body string) Configuration {
		return Configuration{Repository: "acme/app", PullNumber: "42", SectionKey: key, Body: body, ExpectedHeadSHA: "abc"}
	}
	eligible := func() (bool, error) { return true, nil }

	var concurrentErr error
	host.onRead = func() {
		_, concurrentErr = PublishSerialized(t.Context(), host, store, section("merge-confidence", "Score"), eligible)
	}
	outcome, err := PublishSerialized(t.Context(), host, store, section("review-request", "Findings"), eligible)
	require.NoError(t, err)
	require.Equal(t, StatusPublished, outcome.Status)
	require.ErrorIs(t, concurrentErr, core.ErrPublicationBusy)

	outcome, err = PublishSerialized(t.Context(), host, store, section("merge-confidence", "Score"), eligible)
	require.NoError(t, err)
	require.Equal(t, StatusPublished, outcome.Status)
	assert.Contains(t, host.body, "Findings")
	assert.Contains(t, host.body, "Score")
}
