package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/superplanehq/superplane/pkg/models/factory"
	"gorm.io/gorm"
)

const (
	FactoryWorkOrderStatusNoteKindInfo = factory.StatusNoteKindInfo

	MaxFactoryWorkOrderStatusNoteHeadlineBytes = 255

	// MaxFactoryWorkOrderStatusNoteBodyBytes caps the markdown body; a
	// note is a one-paragraph announcement, not a report.
	MaxFactoryWorkOrderStatusNoteBodyBytes = 16 * 1024

	MaxFactoryWorkOrderStatusNoteCtaLabelBytes = 255
	MaxFactoryWorkOrderStatusNoteCtaURLBytes   = 2048
)

var ErrFactoryWorkOrderStatusNoteInvalid = errors.New("invalid work order status note")

var factoryWorkOrderStatusNoteKinds = []string{
	FactoryWorkOrderStatusNoteKindInfo,
}

// FactoryWorkOrderStatusNote is the current-wait announcement stored on
// the work order row (jsonb column, latest wins). It explains what a
// Waiting order is blocked on and what resolves it — e.g. a PR watcher
// announcing that merging the tracked pull request completes the order.
// Any lifecycle transition clears it (see FactoryWorkOrder.UpdateStatus),
// so it always describes the current wait.
type FactoryWorkOrderStatusNote struct {
	Kind     string `json:"kind"`
	Headline string `json:"headline"`
	Body     string `json:"body,omitempty"`
	CtaLabel string `json:"ctaLabel,omitempty"`
	CtaURL   string `json:"ctaUrl,omitempty"`
	// Automation and Run snapshot who announced the wait, captured at
	// write time like check attribution.
	Automation *factory.AutomationRef `json:"automation,omitempty"`
	Run        *factory.RunRef        `json:"run,omitempty"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type FactoryWorkOrderStatusNoteParams struct {
	Kind       string
	Headline   string
	Body       string
	CtaLabel   string
	CtaURL     string
	Automation *factory.AutomationRef
	Run        *factory.RunRef
}

// StatusNoteRef decodes the note stored on the row. Returns nil when the
// order has no status note.
func (o *FactoryWorkOrder) StatusNoteRef() (*FactoryWorkOrderStatusNote, error) {
	if len(o.StatusNote) == 0 {
		return nil, nil
	}

	var note FactoryWorkOrderStatusNote
	if err := json.Unmarshal(o.StatusNote, &note); err != nil {
		return nil, err
	}

	return &note, nil
}

// SetStatusNote validates and stores the note, replacing any previous
// one. Only open orders can carry a note: a draft was never dispatched
// and a closed order is not waiting on anything.
func (o *FactoryWorkOrder) SetStatusNote(
	tx *gorm.DB,
	params FactoryWorkOrderStatusNoteParams,
) (*FactoryWorkOrderStatusNote, error) {
	if !o.IsOpen() {
		return nil, fmt.Errorf("%w: work order must be open, is %s", ErrFactoryWorkOrderStatusNoteInvalid, o.State)
	}

	note, err := normalizeStatusNoteParams(params)
	if err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(note)
	if err != nil {
		return nil, err
	}

	// UpdateColumn: setting a note is metadata, not a lifecycle edit —
	// it must not move `updated_at` (the close-instant heuristic).
	if err := tx.Model(o).UpdateColumn("status_note", encoded).Error; err != nil {
		return nil, err
	}

	o.StatusNote = encoded
	return note, nil
}

// ClearStatusNote removes the note without a state transition, for waits
// that resolve while the order stays open. Lifecycle transitions clear
// the note themselves in UpdateStatus.
func (o *FactoryWorkOrder) ClearStatusNote(tx *gorm.DB) error {
	if err := tx.Model(o).UpdateColumn("status_note", nil).Error; err != nil {
		return err
	}

	o.StatusNote = nil
	return nil
}

func normalizeStatusNoteParams(params FactoryWorkOrderStatusNoteParams) (*FactoryWorkOrderStatusNote, error) {
	kind := params.Kind
	if kind == "" {
		kind = FactoryWorkOrderStatusNoteKindInfo
	}
	if !IsValidWorkOrderStatusNoteKind(kind) {
		return nil, fmt.Errorf("%w: unknown kind %q", ErrFactoryWorkOrderStatusNoteInvalid, kind)
	}

	headline := strings.TrimSpace(params.Headline)
	if headline == "" {
		return nil, fmt.Errorf("%w: headline is required", ErrFactoryWorkOrderStatusNoteInvalid)
	}
	if len(headline) > MaxFactoryWorkOrderStatusNoteHeadlineBytes {
		return nil, fmt.Errorf(
			"%w: headline exceeds %d bytes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNoteHeadlineBytes,
		)
	}

	body := strings.TrimSpace(params.Body)
	if len(body) > MaxFactoryWorkOrderStatusNoteBodyBytes {
		return nil, fmt.Errorf(
			"%w: body exceeds %d bytes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNoteBodyBytes,
		)
	}

	ctaLabel, ctaURL, err := normalizeStatusNoteCta(params.CtaLabel, params.CtaURL)
	if err != nil {
		return nil, err
	}

	return &FactoryWorkOrderStatusNote{
		Kind:       kind,
		Headline:   headline,
		Body:       body,
		CtaLabel:   ctaLabel,
		CtaURL:     ctaURL,
		Automation: params.Automation,
		Run:        params.Run,
		UpdatedAt:  time.Now(),
	}, nil
}

func normalizeStatusNoteCta(label, rawURL string) (string, string, error) {
	label = strings.TrimSpace(label)
	rawURL = strings.TrimSpace(rawURL)
	if label == "" && rawURL == "" {
		return "", "", nil
	}
	if label == "" || rawURL == "" {
		return "", "", fmt.Errorf(
			"%w: cta label and cta url must be set together",
			ErrFactoryWorkOrderStatusNoteInvalid,
		)
	}
	if len(label) > MaxFactoryWorkOrderStatusNoteCtaLabelBytes {
		return "", "", fmt.Errorf(
			"%w: cta label exceeds %d bytes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNoteCtaLabelBytes,
		)
	}
	if len(rawURL) > MaxFactoryWorkOrderStatusNoteCtaURLBytes {
		return "", "", fmt.Errorf(
			"%w: cta url exceeds %d bytes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNoteCtaURLBytes,
		)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", "", fmt.Errorf("%w: cta url must be an absolute http(s) URL", ErrFactoryWorkOrderStatusNoteInvalid)
	}

	return label, rawURL, nil
}

func IsValidWorkOrderStatusNoteKind(kind string) bool {
	return slices.Contains(factoryWorkOrderStatusNoteKinds, kind)
}
