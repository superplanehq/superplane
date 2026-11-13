package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrPublicationUnchanged = errors.New("publication unchanged")

// PublicationOrder is resolved from persisted, authorized run and root-event records.
// Components must never decode these timestamps from configuration or agent output.
type PublicationOrder struct {
	SourceRunID        uuid.UUID `json:"sourceRunId"`
	SourceCreatedAt    time.Time `json:"sourceCreatedAt"`
	AttemptRootEventID uuid.UUID `json:"attemptRootEventId"`
	AttemptCreatedAt   time.Time `json:"attemptCreatedAt"`
}

func (o PublicationOrder) Compare(other PublicationOrder) int {
	if c := o.SourceCreatedAt.Compare(other.SourceCreatedAt); c != 0 {
		return c
	}
	if c := bytes.Compare(o.SourceRunID[:], other.SourceRunID[:]); c != 0 {
		return c
	}
	if c := o.AttemptCreatedAt.Compare(other.AttemptCreatedAt); c != 0 {
		return c
	}
	return bytes.Compare(o.AttemptRootEventID[:], other.AttemptRootEventID[:])
}

// OrderedPublication stores an opaque component-owned payload. A disabled record
// remains as a fence until an explicit new activation.
type OrderedPublication struct {
	Key                  string           `json:"key"`
	Order                PublicationOrder `json:"order"`
	Generation           int64            `json:"generation"`
	ActivationGeneration int64            `json:"activationGeneration"`
	Revision             int64            `json:"revision"`
	Disabled             bool             `json:"disabled"`
	OwnerCanvasID        uuid.UUID        `json:"ownerCanvasId"`
	OwnerNodeID          string           `json:"ownerNodeId"`
	OwnerDeletedAt       *time.Time       `json:"ownerDeletedAt,omitempty"`
	Payload              json.RawMessage  `json:"payload"`
	Hook                 string           `json:"hook"`
	WakeAt               *time.Time       `json:"wakeAt,omitempty"`
}

// CanReplace fences callbacks from a disabled activation even if they carry a
// newer source. Re-activation must explicitly use a greater activation generation.
func (p OrderedPublication) CanReplace(order PublicationOrder, activation int64, disabled bool) bool {
	if p.Generation == 0 {
		return activation >= 1
	}
	if disabled {
		return activation == p.ActivationGeneration && order.Compare(p.Order) >= 0
	}
	if p.Disabled {
		return activation > p.ActivationGeneration && order.Compare(p.Order) >= 0
	}
	return activation == p.ActivationGeneration && order.Compare(p.Order) >= 0
}

type PublicationExecution struct {
	Component     string
	Configuration any
	Output        any
}

type PublicationOwnerStatus struct {
	Available bool
	Removed   bool
}

// PublicationNode exposes saved node configuration without component policy.
type PublicationNode struct {
	Component     string
	Configuration map[string]any
}

// PublicationStore serializes one integration/key and persists the next hook in
// the same transaction. Reads and provider lookups belong outside Update.
type PublicationStore interface {
	// NodeSnapshot reads the live node, or the version admitted by the attempt.
	// A missing or removed node returns nil. Components own eligibility policy.
	NodeSnapshot(canvasID uuid.UUID, nodeID string, attemptRootEventID uuid.UUID) (*PublicationNode, error)
	Get(key string) (*OrderedPublication, error)
	List(prefix string) ([]OrderedPublication, error)
	Update(key string, apply func(*OrderedPublication) error) error
	AttemptFinished(rootEventID uuid.UUID) (bool, error)
	// AttemptEnded reports termination, even if a running attempt has errors.
	AttemptEnded(rootEventID uuid.UUID) (bool, error)
	OwnerStatus(canvasID uuid.UUID, nodeID string) (PublicationOwnerStatus, error)
}

type PublicationContext interface {
	PublicationStore
	ResolveOrder(sourceRunID, attemptRootEventID uuid.UUID) (PublicationOrder, error)
	ExecutionOutput(executionID uuid.UUID) (PublicationExecution, error)
}

// PublicationIntegration is optional: integrations that own durable publications
// use it from hooks without keeping component executions open for monitoring.
type PublicationIntegration interface {
	Publications() PublicationStore
}
