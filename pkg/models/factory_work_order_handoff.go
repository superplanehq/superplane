package models

import (
	"slices"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PendingHandOff struct {
	LineID         uuid.UUID `json:"line_id"`
	Column         string    `json:"column"`
	PullRequestURL string    `json:"pull_request_url,omitempty"`
	// PullRequestURLs holds the pull requests after the first one. A task
	// can hand off one pull request in each workspace repository.
	PullRequestURLs []string `json:"pull_request_urls,omitempty"`
}

// NewPendingHandOff saves the first URL in PullRequestURL so older readers
// still find it.
func NewPendingHandOff(lineID uuid.UUID, column string, urls []string) *PendingHandOff {
	urls = NormalizeHandOffPullRequestURLs(urls)
	pending := &PendingHandOff{LineID: lineID, Column: column}
	if len(urls) == 0 {
		return pending
	}
	pending.PullRequestURL = urls[0]
	if len(urls) > 1 {
		pending.PullRequestURLs = slices.Clone(urls[1:])
	}
	return pending
}

// URLs returns every saved pull request URL, first URL first.
func (p PendingHandOff) URLs() []string {
	return NormalizeHandOffPullRequestURLs(append([]string{p.PullRequestURL}, p.PullRequestURLs...))
}

func (p PendingHandOff) Equal(other PendingHandOff) bool {
	return p.LineID == other.LineID &&
		p.Column == other.Column &&
		slices.Equal(p.URLs(), other.URLs())
}

// NormalizeHandOffPullRequestURLs trims the URLs and drops empty and
// repeated values. It keeps the input order.
func NormalizeHandOffPullRequestURLs(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, raw := range urls {
		value := strings.TrimSpace(raw)
		if value == "" || slices.Contains(out, value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func (o *FactoryWorkOrder) SetPendingHandOff(tx *gorm.DB, pending *PendingHandOff) error {
	var value *datatypes.JSONType[PendingHandOff]
	if pending != nil {
		saved := datatypes.NewJSONType(*pending)
		value = &saved
	}
	if err := tx.Model(o).Omit("Assignees").Update("pending_handoff", value).Error; err != nil {
		return err
	}
	o.PendingHandOff = value
	return nil
}
