package native

import (
	"context"
	"errors"
	"sync"

	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

var errNativeSessionNotFound = errors.New("native agent session not found")

type Store interface {
	Save(ctx context.Context, snapshot SessionSnapshot) error
	Load(ctx context.Context, providerSessionID string) (*SessionSnapshot, error)
	Delete(ctx context.Context, providerSessionID string) error
}

type SessionSnapshot struct {
	ID                 string
	History            []llm.Message
	Awaiting           bool
	Interrupted        bool
	Steps              int
	LastToolSignature  string
	RepeatedToolCalls  int
	PendingToolNames   map[string]string
	CompactionFailures int
	Summary            string
}

type memoryStore struct {
	mu       sync.Mutex
	sessions map[string]SessionSnapshot
}

func newMemoryStore() *memoryStore {
	return &memoryStore{sessions: map[string]SessionSnapshot{}}
}

func NewMemoryStore() Store {
	return newMemoryStore()
}

func (s *memoryStore) Save(_ context.Context, snapshot SessionSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[snapshot.ID] = cloneSnapshot(snapshot)
	return nil
}

func (s *memoryStore) Load(_ context.Context, providerSessionID string) (*SessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot, ok := s.sessions[providerSessionID]
	if !ok {
		return nil, errNativeSessionNotFound
	}
	cloned := cloneSnapshot(snapshot)
	return &cloned, nil
}

func (s *memoryStore) Delete(_ context.Context, providerSessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, providerSessionID)
	return nil
}

func cloneSnapshot(snapshot SessionSnapshot) SessionSnapshot {
	pending := map[string]string{}
	for key, value := range snapshot.PendingToolNames {
		pending[key] = value
	}
	return SessionSnapshot{
		ID:                 snapshot.ID,
		History:            cloneMessages(snapshot.History),
		Awaiting:           snapshot.Awaiting,
		Interrupted:        snapshot.Interrupted,
		Steps:              snapshot.Steps,
		LastToolSignature:  snapshot.LastToolSignature,
		RepeatedToolCalls:  snapshot.RepeatedToolCalls,
		PendingToolNames:   pending,
		CompactionFailures: snapshot.CompactionFailures,
		Summary:            snapshot.Summary,
	}
}
