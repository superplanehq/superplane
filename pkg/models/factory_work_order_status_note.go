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
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	FactoryWorkOrderStatusNoteKindInfo = factory.StatusNoteKindInfo

	MaxFactoryWorkOrderStatusNoteKeyBytes      = 255
	MaxFactoryWorkOrderStatusNoteHeadlineBytes = 255

	// MaxFactoryWorkOrderStatusNoteBodyBytes caps the markdown body; a
	// note is a one-paragraph announcement, not a report.
	MaxFactoryWorkOrderStatusNoteBodyBytes = 16 * 1024

	MaxFactoryWorkOrderStatusNoteCtaLabelBytes = 255
	MaxFactoryWorkOrderStatusNoteCtaURLBytes   = 2048

	// MaxFactoryWorkOrderStatusNotes caps how many notes one order can
	// carry. Notes are latest-only per key; this is a payload-size bound.
	MaxFactoryWorkOrderStatusNotes = 20
)

var ErrFactoryWorkOrderStatusNoteInvalid = errors.New("invalid work order status note")

var factoryWorkOrderStatusNoteKinds = []string{
	FactoryWorkOrderStatusNoteKindInfo,
}

// FactoryWorkOrderStatusNote is one current-wait announcement. Notes live
// as a jsonb array on the work order row, keyed like checks: the first
// set of a key creates the note, a later set with the same key updates
// it in place, and a different key sits beside it. Kind is the payload
// shape (`info` today), not the identity. Any lifecycle transition
// clears the whole set (see FactoryWorkOrder.UpdateStatus).
type FactoryWorkOrderStatusNote struct {
	Key      string `json:"key"`
	Kind     string `json:"kind"`
	Headline string `json:"headline"`
	Body     string `json:"body,omitempty"`
	CtaLabel string `json:"ctaLabel,omitempty"`
	CtaURL   string `json:"ctaUrl,omitempty"`
	// ShowOnlyWhenWaiting hides the note on the work order page while a
	// line is running. The default is false: the note stays visible
	// for the whole open wait, including an active dispatch.
	ShowOnlyWhenWaiting bool `json:"showOnlyWhenWaiting,omitempty"`
	// Automation and Run snapshot who announced the wait, captured at
	// write time like check attribution.
	Automation *factory.AutomationRef `json:"automation,omitempty"`
	Run        *factory.RunRef        `json:"run,omitempty"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

type FactoryWorkOrderStatusNoteParams struct {
	Key                 string
	Kind                string
	Headline            string
	Body                string
	CtaLabel            string
	CtaURL              string
	ShowOnlyWhenWaiting bool
	Automation          *factory.AutomationRef
	Run                 *factory.RunRef
}

// StatusNotes decodes the notes stored on the row. Returns nil when the
// order has no status notes.
func (o *FactoryWorkOrder) StatusNotes() ([]FactoryWorkOrderStatusNote, error) {
	if len(o.StatusNote) == 0 {
		return nil, nil
	}

	var notes []FactoryWorkOrderStatusNote
	if err := json.Unmarshal(o.StatusNote, &notes); err != nil {
		return nil, err
	}

	return notes, nil
}

// SetStatusNote validates and upserts the note by Key. Only open orders
// can carry notes: a draft was never dispatched and a closed order is
// not waiting on anything. The write locks the row and reloads the
// stored list so concurrent sets with different keys keep both notes,
// and a close that already committed is not overwritten.
func (o *FactoryWorkOrder) SetStatusNote(
	tx *gorm.DB,
	params FactoryWorkOrderStatusNoteParams,
) (*FactoryWorkOrderStatusNote, error) {
	note, err := normalizeStatusNoteParams(params)
	if err != nil {
		return nil, err
	}

	err = tx.Transaction(func(tx *gorm.DB) error {
		if err := o.lockAndReload(tx); err != nil {
			return err
		}
		if !o.IsOpen() {
			return fmt.Errorf("%w: work order must be open, is %s", ErrFactoryWorkOrderStatusNoteInvalid, o.State)
		}

		notes, err := o.StatusNotes()
		if err != nil {
			return err
		}

		notes, err = upsertStatusNote(notes, *note)
		if err != nil {
			return err
		}

		return o.persistStatusNotes(tx, notes)
	})
	if err != nil {
		return nil, err
	}

	return note, nil
}

// ClearStatusNote removes the note with the given key. Unknown keys are
// a no-op. Lifecycle transitions clear the whole set in UpdateStatus.
func (o *FactoryWorkOrder) ClearStatusNote(tx *gorm.DB, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key is required", ErrFactoryWorkOrderStatusNoteInvalid)
	}

	return tx.Transaction(func(tx *gorm.DB) error {
		if err := o.lockAndReload(tx); err != nil {
			return err
		}

		notes, err := o.StatusNotes()
		if err != nil {
			return err
		}

		kept := slices.DeleteFunc(notes, func(note FactoryWorkOrderStatusNote) bool {
			return note.Key == key
		})
		if len(kept) == len(notes) {
			return nil
		}

		return o.persistStatusNotes(tx, kept)
	})
}

// ClearStatusNotes removes every note without a state transition, for
// waits that all resolve while the order stays open.
func (o *FactoryWorkOrder) ClearStatusNotes(tx *gorm.DB) error {
	return tx.Transaction(func(tx *gorm.DB) error {
		if err := o.lockAndReload(tx); err != nil {
			return err
		}
		return o.persistStatusNotes(tx, nil)
	})
}

// lockAndReload takes a row lock and replaces the in-memory state,
// result, and notes with the committed row. Callers must hold a
// transaction: the lock lasts only until that transaction ends.
func (o *FactoryWorkOrder) lockAndReload(tx *gorm.DB) error {
	var locked FactoryWorkOrder
	err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", o.ID).
		First(&locked).
		Error
	if err != nil {
		return err
	}

	o.State = locked.State
	o.Result = locked.Result
	o.StatusNote = locked.StatusNote
	return nil
}

func normalizeStatusNoteParams(params FactoryWorkOrderStatusNoteParams) (*FactoryWorkOrderStatusNote, error) {
	key := strings.TrimSpace(params.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: key is required", ErrFactoryWorkOrderStatusNoteInvalid)
	}
	if len(key) > MaxFactoryWorkOrderStatusNoteKeyBytes {
		return nil, fmt.Errorf(
			"%w: key exceeds %d bytes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNoteKeyBytes,
		)
	}

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
		Key:                 key,
		Kind:                kind,
		Headline:            headline,
		Body:                body,
		CtaLabel:            ctaLabel,
		CtaURL:              ctaURL,
		ShowOnlyWhenWaiting: params.ShowOnlyWhenWaiting,
		Automation:          params.Automation,
		Run:                 params.Run,
		UpdatedAt:           time.Now(),
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

func upsertStatusNote(notes []FactoryWorkOrderStatusNote, note FactoryWorkOrderStatusNote) ([]FactoryWorkOrderStatusNote, error) {
	for i := range notes {
		if notes[i].Key == note.Key {
			notes[i] = note
			return notes, nil
		}
	}

	if len(notes) >= MaxFactoryWorkOrderStatusNotes {
		return nil, fmt.Errorf(
			"%w: cannot store more than %d notes",
			ErrFactoryWorkOrderStatusNoteInvalid, MaxFactoryWorkOrderStatusNotes,
		)
	}

	return append(notes, note), nil
}

func (o *FactoryWorkOrder) persistStatusNotes(tx *gorm.DB, notes []FactoryWorkOrderStatusNote) error {
	var encoded datatypes.JSON
	if len(notes) > 0 {
		raw, err := json.Marshal(notes)
		if err != nil {
			return err
		}
		encoded = raw
	}

	// UpdateColumn: setting a note is metadata, not a lifecycle edit —
	// it must not move `updated_at` (the close-instant heuristic).
	if err := tx.Model(o).UpdateColumn("status_note", encoded).Error; err != nil {
		return err
	}

	o.StatusNote = encoded
	return nil
}

func IsValidWorkOrderStatusNoteKind(kind string) bool {
	return slices.Contains(factoryWorkOrderStatusNoteKinds, kind)
}
