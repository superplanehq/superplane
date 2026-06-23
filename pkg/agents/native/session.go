package native

import (
	"fmt"
	"strings"
	"sync"

	"github.com/superplanehq/superplane/pkg/agents"
	"github.com/superplanehq/superplane/pkg/agents/native/llm"
)

type sessionState struct {
	mu sync.Mutex

	id          string
	history     []llm.Message
	awaiting    bool
	interrupted bool
	steps       int

	lastToolSignature  string
	repeatedToolCalls  int
	pendingToolNames   map[string]string
	compactionFailures int
	summary            string
}

type turnSnapshot struct {
	history []llm.Message
}

func newSessionState(id, systemPrompt string) *sessionState {
	session := &sessionState{
		id:               id,
		pendingToolNames: map[string]string{},
	}
	systemPrompt = strings.TrimSpace(systemPrompt)
	if systemPrompt != "" {
		session.history = append(session.history, llm.NewSystemMessage(systemPrompt))
	}
	return session
}

func sessionFromSnapshot(snapshot SessionSnapshot) *sessionState {
	return &sessionState{
		id:                 snapshot.ID,
		history:            cloneMessages(snapshot.History),
		awaiting:           snapshot.Awaiting,
		interrupted:        snapshot.Interrupted,
		steps:              snapshot.Steps,
		lastToolSignature:  snapshot.LastToolSignature,
		repeatedToolCalls:  snapshot.RepeatedToolCalls,
		pendingToolNames:   clonePendingToolNames(snapshot.PendingToolNames),
		compactionFailures: snapshot.CompactionFailures,
		summary:            snapshot.Summary,
	}
}

func (s *sessionState) snapshot() SessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotLocked()
}

func (s *sessionState) snapshotLocked() SessionSnapshot {
	return SessionSnapshot{
		ID:                 s.id,
		History:            cloneMessages(s.history),
		Awaiting:           s.awaiting,
		Interrupted:        s.interrupted,
		Steps:              s.steps,
		LastToolSignature:  s.lastToolSignature,
		RepeatedToolCalls:  s.repeatedToolCalls,
		PendingToolNames:   clonePendingToolNames(s.pendingToolNames),
		CompactionFailures: s.compactionFailures,
		Summary:            s.summary,
	}
}

func (s *sessionState) enqueueUserMessage(message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.awaiting {
		return agents.ErrSessionBusy
	}
	s.history = append(s.history, llm.NewUserMessage(message))
	s.awaiting = false
	s.interrupted = false
	s.steps = 0
	s.lastToolSignature = ""
	s.repeatedToolCalls = 0
	return nil
}

func (s *sessionState) interrupt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interrupted = true
	s.awaiting = false
}

func (s *sessionState) nextTurn(maxSteps int) (turnSnapshot, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.interrupted {
		return turnSnapshot{}, false, agents.ErrSessionAlreadyTerminated
	}
	if s.awaiting {
		return turnSnapshot{}, false, nil
	}
	if s.steps >= maxSteps {
		return turnSnapshot{}, false, errMaxStepsReached
	}
	s.steps++

	return turnSnapshot{history: cloneMessages(s.history)}, true, nil
}

func (s *sessionState) completeTurn(assistantBlocks []llm.Block) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(assistantBlocks) > 0 {
		s.history = append(s.history, llm.NewAssistantMessage(assistantBlocks))
	}
	s.awaiting = false
}

func (s *sessionState) pauseForToolResults(assistantBlocks []llm.Block, toolCalls []llm.ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(toolCalls) == 0 {
		return nil
	}
	s.history = append(s.history, llm.NewAssistantMessage(assistantBlocks))
	s.pendingToolNames = map[string]string{}
	for _, toolCall := range toolCalls {
		s.pendingToolNames[toolCall.ID] = toolCall.Name
	}
	s.awaiting = true
	return nil
}

func (s *sessionState) appendToolResults(results []agents.CustomToolResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.awaiting {
		return nil
	}

	toolResults := make([]llm.ToolResult, 0, len(results))
	for _, result := range results {
		name := s.pendingToolNames[result.CustomToolUseID]
		toolResults = append(toolResults, llm.ToolResult{
			ToolCallID: result.CustomToolUseID,
			Name:       name,
			Content:    result.Content,
			IsError:    result.IsError,
		})
	}

	s.history = append(s.history, llm.NewToolResultMessage(toolResults))
	s.pendingToolNames = map[string]string{}
	s.awaiting = false
	return nil
}

func (s *sessionState) checkRepeatedToolCall(toolCall llm.ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	signature := fmt.Sprintf("%s:%s", toolCall.Name, strings.TrimSpace(toolCall.Input))
	if signature == s.lastToolSignature {
		s.repeatedToolCalls++
	} else {
		s.lastToolSignature = signature
		s.repeatedToolCalls = 1
	}
	if s.repeatedToolCalls >= doomLoopLimit {
		return fmt.Errorf("%w: %s", errDoomLoop, toolCall.Name)
	}
	return nil
}

func cloneMessages(messages []llm.Message) []llm.Message {
	cloned := make([]llm.Message, 0, len(messages))
	for _, message := range messages {
		blocks := make([]llm.Block, len(message.Blocks))
		copy(blocks, message.Blocks)
		cloned = append(cloned, llm.Message{
			Role:   message.Role,
			Blocks: blocks,
		})
	}
	return cloned
}

func clonePendingToolNames(names map[string]string) map[string]string {
	cloned := map[string]string{}
	for id, name := range names {
		cloned[id] = name
	}
	return cloned
}
