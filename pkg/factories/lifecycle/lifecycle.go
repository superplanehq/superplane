// Package lifecycle lets built-in automations take part in shared canvas
// operations. Shared code calls the operations in this package. It does not
// know which automation, if any, handles an operation. Each registered
// automation recognizes its own records and ignores all other records.
package lifecycle

import (
	"errors"

	"github.com/google/uuid"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Automation contains the optional operations of one built-in automation.
// An operation that does not recognize its input must return it unchanged.
type Automation struct {
	// Name labels run-link errors, for example "review request".
	Name string

	// ValidatePublication can reject a canvas version before it goes live.
	ValidatePublication func(Publication) error

	// AdmitEvent can transform, deduplicate, or suppress a new trigger event.
	AdmitEvent func(EventAdmission) (EventAdmission, error)

	// LinkRun connects a new run to records that the automation owns.
	// Wrap errors with Reject when a retry cannot fix them.
	LinkRun func(tx *gorm.DB, canvas *models.Canvas, event *models.CanvasEvent, run *models.CanvasRun) error

	// PrepareAction runs before an action executes. It can change the
	// execution context or complete the execution. Return true when the
	// execution is complete and the action must not run.
	PrepareAction func(Action) (bool, error)

	// PrepareOutput can validate or enrich output before it is saved. An
	// error fails the execution.
	PrepareOutput func(Output) ([]any, error)

	// RetryEventData can replace the data of an explicitly re-emitted
	// trigger event. Return false when the automation does not own the event.
	RetryEventData func(tx *gorm.DB, canvas *models.Canvas, trigger *models.CanvasNode, source *models.CanvasEvent) (any, bool, error)
}

// Publication is a canvas version that is about to go live.
type Publication struct {
	Tx             *gorm.DB
	Canvas         *models.Canvas
	Draft          *models.CanvasVersion
	OrganizationID uuid.UUID
}

// EventAdmission is a trigger event before it is saved.
type EventAdmission struct {
	Tx          *gorm.DB
	Node        *models.CanvasNode
	Run         *models.CanvasRun
	PayloadType string
	Payload     any
	// DedupKey replaces the caller's key when it is not empty.
	DedupKey string
	// Suppress drops the event without an error.
	Suppress bool
}

// Action is an action execution before the action runs.
type Action struct {
	Tx        *gorm.DB
	Canvas    *models.Canvas
	Execution *models.CanvasNodeExecution
	Context   *core.ExecutionContext
	// Integration opens an integration that the node does not reference.
	Integration func(*models.Integration) core.IntegrationContext
}

// Output is the output of an execution before it is saved.
type Output struct {
	Tx          *gorm.DB
	Execution   *models.CanvasNodeExecution
	Channel     string
	PayloadType string
	Payloads    []any
}

var automations []Automation

// Register adds an automation. Call it from an init function only.
func Register(automation Automation) {
	automations = append(automations, automation)
}

// ValidatePublication returns the first rejection of a canvas version.
func ValidatePublication(publication Publication) error {
	for _, automation := range automations {
		if automation.ValidatePublication == nil {
			continue
		}
		if err := automation.ValidatePublication(publication); err != nil {
			return err
		}
	}
	return nil
}

// AdmitEvent applies all automations to a new trigger event.
func AdmitEvent(event EventAdmission) (EventAdmission, error) {
	for _, automation := range automations {
		if automation.AdmitEvent == nil {
			continue
		}
		admitted, err := automation.AdmitEvent(event)
		if err != nil || admitted.Suppress {
			return admitted, err
		}
		event = admitted
	}
	return event, nil
}

// RunLink is the run-link operation of one automation.
type RunLink struct {
	Name string
	Link func(tx *gorm.DB, canvas *models.Canvas, event *models.CanvasEvent, run *models.CanvasRun) error
}

// RunLinks returns the run-link operations in registration order.
func RunLinks() []RunLink {
	links := []RunLink{}
	for _, automation := range automations {
		if automation.LinkRun != nil {
			links = append(links, RunLink{Name: automation.Name, Link: automation.LinkRun})
		}
	}
	return links
}

// PrepareAction returns true when an automation completed the execution.
func PrepareAction(action Action) (bool, error) {
	for _, automation := range automations {
		if automation.PrepareAction == nil {
			continue
		}
		handled, err := automation.PrepareAction(action)
		if err != nil || handled {
			return handled, err
		}
	}
	return false, nil
}

// PrepareOutput applies all automations to execution output.
func PrepareOutput(output Output) ([]any, error) {
	for _, automation := range automations {
		if automation.PrepareOutput == nil {
			continue
		}
		payloads, err := automation.PrepareOutput(output)
		if err != nil {
			return nil, err
		}
		output.Payloads = payloads
	}
	return output.Payloads, nil
}

// RetryEventData returns the data for an explicitly re-emitted trigger event.
// Without an owning automation, it returns the source event data.
func RetryEventData(tx *gorm.DB, canvas *models.Canvas, trigger *models.CanvasNode, source *models.CanvasEvent) (any, error) {
	for _, automation := range automations {
		if automation.RetryEventData == nil {
			continue
		}
		data, handled, err := automation.RetryEventData(tx, canvas, trigger, source)
		if err != nil || handled {
			return data, err
		}
	}
	return source.Data.Data(), nil
}

// RejectionError marks a run-link error that a retry cannot fix.
type RejectionError struct {
	err error
}

func (e *RejectionError) Error() string { return e.err.Error() }

func (e *RejectionError) Unwrap() error { return e.err }

// Reject marks err as permanent. The run fails and the event is routed.
func Reject(err error) error {
	if err == nil {
		return nil
	}
	return &RejectionError{err: err}
}

// IsRejection reports whether err is permanent.
func IsRejection(err error) bool {
	var rejection *RejectionError
	return errors.As(err, &rejection)
}
