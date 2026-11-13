package core

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrPublicationSuperseded = errors.New("a newer source or attempt owns this publication")
	ErrPublicationBusy       = errors.New("another publication write is in progress")
)

// OrderedWrite identifies the accepted claim of one attempt on one publication key.
// Components persist it with the execution and present it at the mutation boundary.
type OrderedWrite struct {
	Key        string `json:"key"`
	Generation int64  `json:"generation"`
}

// PublicationLease serializes provider writes for one key. Writes must finish
// before ExpiresAt so an expired owner cannot overwrite a successor.
type PublicationLease struct {
	ID        uuid.UUID `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// orderedWriteState is the payload of ordered writes. State is component owned,
// for example the ID of a managed provider resource.
type orderedWriteState struct {
	Lease *PublicationLease `json:"lease,omitempty"`
	State json.RawMessage   `json:"state,omitempty"`
}

// AcceptOrderedWrite resolves the trusted order of the current execution and
// makes it the current owner of the key unless a newer source or attempt owns it.
// Ordered writes use one activation generation. Components fence activation
// through their own saved configuration.
func AcceptOrderedWrite(publications PublicationContext, key string, ownerCanvasID uuid.UUID, ownerNodeID string) (OrderedWrite, bool, error) {
	order, err := publications.ResolveOrder(uuid.Nil, uuid.Nil)
	if err != nil {
		return OrderedWrite{}, false, err
	}
	var accepted OrderedWrite
	err = publications.Update(key, func(record *OrderedPublication) error {
		if !record.CanReplace(order, 1, false) {
			return ErrPublicationUnchanged
		}
		record.Generation++
		record.Order, record.ActivationGeneration, record.Disabled = order, 1, false
		record.OwnerCanvasID, record.OwnerNodeID = ownerCanvasID, ownerNodeID
		accepted = OrderedWrite{Key: key, Generation: record.Generation}
		return nil
	})
	if err != nil {
		return OrderedWrite{}, false, err
	}
	return accepted, accepted.Generation > 0, nil
}

// IsCurrentOrderedWrite reports whether no newer source or attempt was accepted.
func IsCurrentOrderedWrite(store PublicationStore, write OrderedWrite) (bool, error) {
	record, err := store.Get(write.Key)
	if err != nil {
		return false, err
	}
	return record != nil && record.Generation == write.Generation, nil
}

// BeginOrderedWrite claims the mutation boundary immediately before a provider
// write. It returns the component state saved by earlier writes.
func BeginOrderedWrite(store PublicationStore, write OrderedWrite, duration time.Duration) (PublicationLease, json.RawMessage, error) {
	var lease PublicationLease
	var state json.RawMessage
	err := store.Update(write.Key, func(record *OrderedPublication) error {
		if record.Generation != write.Generation {
			return ErrPublicationSuperseded
		}
		payload, err := decodeOrderedWriteState(record.Payload)
		if err != nil {
			return err
		}
		if payload.Lease != nil && time.Now().Before(payload.Lease.ExpiresAt) {
			return ErrPublicationBusy
		}
		// PostgreSQL JSON keeps the full value, but truncate for stable comparison.
		lease = PublicationLease{ID: uuid.New(), ExpiresAt: time.Now().Add(duration).Truncate(time.Microsecond)}
		payload.Lease = &lease
		state = payload.State
		record.Payload, err = json.Marshal(payload)
		return err
	})
	return lease, state, err
}

// FinishOrderedWrite releases the lease and saves component state. The state
// describes provider resources that exist, so it is saved even when a newer
// attempt was accepted during the write.
func FinishOrderedWrite(store PublicationStore, write OrderedWrite, lease PublicationLease, state json.RawMessage) error {
	return store.Update(write.Key, func(record *OrderedPublication) error {
		payload, err := decodeOrderedWriteState(record.Payload)
		if err != nil {
			return err
		}
		if payload.Lease != nil && payload.Lease.ID == lease.ID {
			payload.Lease = nil
		}
		if state != nil {
			payload.State = state
		}
		record.Payload, err = json.Marshal(payload)
		return err
	})
}

func decodeOrderedWriteState(raw json.RawMessage) (orderedWriteState, error) {
	var state orderedWriteState
	if len(raw) == 0 || string(raw) == "null" {
		return state, nil
	}
	err := json.Unmarshal(raw, &state)
	return state, err
}

// BeginPublicationLease claims a lease on a key that has no ordered owner.
// Use it to serialize a provider read and write that several keys share.
func BeginPublicationLease(store PublicationStore, key string, duration time.Duration) (PublicationLease, error) {
	lease, _, err := BeginOrderedWrite(store, OrderedWrite{Key: key}, duration)
	return lease, err
}

// FinishPublicationLease releases a lease from BeginPublicationLease.
func FinishPublicationLease(store PublicationStore, key string, lease PublicationLease) error {
	return FinishOrderedWrite(store, OrderedWrite{Key: key}, lease, nil)
}
