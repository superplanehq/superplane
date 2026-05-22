package support

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/agents"
)

const (
	TestAgentProviderName = "test"
	agentStreamBufferSize = 1024
)

var (
	_ agents.Provider               = (*TestAgentProvider)(nil)
	_ agents.ProviderSessionCleaner = (*TestAgentProvider)(nil)
)

type AgentProviderCreateSessionCall struct {
	Options agents.CreateSessionOptions
}

type AgentProviderSendMessageCall struct {
	ProviderSessionID string
	Message           string
	Options           agents.SendMessageOptions
}

type AgentProviderInterruptSessionCall struct {
	ProviderSessionID string
}

type AgentProviderDefineOutcomeCall struct {
	ProviderSessionID string
	Options           agents.DefineOutcomeOptions
}

type AgentProviderDeleteSessionCall struct {
	ProviderSessionID string
}

type AgentProviderSession struct {
	ProviderSessionID string
	CreateOptions     agents.CreateSessionOptions
	Interrupted       bool
	Deleted           bool
}

type agentProviderStreamItem struct {
	event agents.ProviderEvent
	err   error
}

type agentProviderSessionState struct {
	mu      sync.Mutex
	session AgentProviderSession
	events  chan agentProviderStreamItem
	closed  bool
}

type TestAgentProvider struct {
	mu sync.Mutex

	name              string
	sessionIDProvider func() string
	sessions          map[string]*agentProviderSessionState

	createSessionCalls    []AgentProviderCreateSessionCall
	sendMessageCalls      []AgentProviderSendMessageCall
	interruptSessionCalls []AgentProviderInterruptSessionCall
	defineOutcomeCalls    []AgentProviderDefineOutcomeCall
	deleteSessionCalls    []AgentProviderDeleteSessionCall

	createSessionErr    error
	sendMessageErr      error
	interruptSessionErr error
	defineOutcomeErr    error
	streamErr           error
	deleteSessionErr    error

	sendMessageEvents    []agents.ProviderEvent
	defineOutcomeEvents  []agents.ProviderEvent
	sendMessageHandler   func(AgentProviderSendMessageCall) ([]agents.ProviderEvent, error)
	defineOutcomeHandler func(AgentProviderDefineOutcomeCall) ([]agents.ProviderEvent, error)
}

func NewAgentProvider() *TestAgentProvider {
	return &TestAgentProvider{
		name: TestAgentProviderName,
		sessionIDProvider: func() string {
			return "test-session-" + uuid.NewString()
		},
		sessions: map[string]*agentProviderSessionState{},
	}
}

func (p *TestAgentProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, session := range p.sessions {
		session.close()
	}

	p.sessions = map[string]*agentProviderSessionState{}

	p.createSessionCalls = nil
	p.sendMessageCalls = nil
	p.interruptSessionCalls = nil
	p.defineOutcomeCalls = nil
	p.deleteSessionCalls = nil

	p.createSessionErr = nil
	p.sendMessageErr = nil
	p.interruptSessionErr = nil
	p.defineOutcomeErr = nil
	p.streamErr = nil
	p.deleteSessionErr = nil

	p.sendMessageEvents = nil
	p.defineOutcomeEvents = nil
	p.sendMessageHandler = nil
	p.defineOutcomeHandler = nil
}

func (p *TestAgentProvider) Name() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.name
}

func (p *TestAgentProvider) SetName(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.name = name
}

func (p *TestAgentProvider) SetSessionIDProvider(provider func() string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sessionIDProvider = provider
}

func (p *TestAgentProvider) SetCreateSessionError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.createSessionErr = err
}

func (p *TestAgentProvider) SetSendMessageError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sendMessageErr = err
}

func (p *TestAgentProvider) SetInterruptSessionError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interruptSessionErr = err
}

func (p *TestAgentProvider) SetDefineOutcomeError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defineOutcomeErr = err
}

func (p *TestAgentProvider) SetStreamError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streamErr = err
}

func (p *TestAgentProvider) SetDeleteSessionError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deleteSessionErr = err
}

func (p *TestAgentProvider) SetSendMessageEvents(events ...agents.ProviderEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sendMessageEvents = cloneProviderEvents(events)
}

func (p *TestAgentProvider) SetDefineOutcomeEvents(events ...agents.ProviderEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defineOutcomeEvents = cloneProviderEvents(events)
}

func (p *TestAgentProvider) SetSendMessageHandler(
	handler func(AgentProviderSendMessageCall) ([]agents.ProviderEvent, error),
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sendMessageHandler = handler
}

func (p *TestAgentProvider) SetDefineOutcomeHandler(
	handler func(AgentProviderDefineOutcomeCall) ([]agents.ProviderEvent, error),
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defineOutcomeHandler = handler
}

func (p *TestAgentProvider) CreateSession(
	_ context.Context,
	opts agents.CreateSessionOptions,
) (*agents.CreateSessionResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.createSessionCalls = append(p.createSessionCalls, AgentProviderCreateSessionCall{Options: opts})
	if p.createSessionErr != nil {
		return nil, p.createSessionErr
	}

	providerSessionID := p.sessionIDProvider()
	session := &agentProviderSessionState{
		session: AgentProviderSession{
			ProviderSessionID: providerSessionID,
			CreateOptions:     opts,
		},
		events: make(chan agentProviderStreamItem, agentStreamBufferSize),
	}

	p.sessions[providerSessionID] = session
	return &agents.CreateSessionResult{ProviderSessionID: providerSessionID}, nil
}

func (p *TestAgentProvider) SendMessage(
	_ context.Context,
	providerSessionID string,
	message string,
	opts agents.SendMessageOptions,
) error {
	call := AgentProviderSendMessageCall{
		ProviderSessionID: providerSessionID,
		Message:           message,
		Options:           opts,
	}

	p.mu.Lock()
	session := p.sessions[providerSessionID]
	p.sendMessageCalls = append(p.sendMessageCalls, call)
	err := p.sendMessageErr
	handler := p.sendMessageHandler
	events := cloneProviderEvents(p.sendMessageEvents)
	p.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	if err != nil {
		return err
	}

	if handler != nil {
		handlerEvents, err := handler(call)
		if err != nil {
			return err
		}

		return p.QueueEvents(providerSessionID, handlerEvents...)
	}

	return p.QueueEvents(providerSessionID, events...)
}

func (p *TestAgentProvider) InterruptSession(_ context.Context, providerSessionID string) error {
	call := AgentProviderInterruptSessionCall{ProviderSessionID: providerSessionID}

	p.mu.Lock()
	session := p.sessions[providerSessionID]
	p.interruptSessionCalls = append(p.interruptSessionCalls, call)
	err := p.interruptSessionErr
	p.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	if err != nil {
		return err
	}

	session.mu.Lock()
	session.session.Interrupted = true
	session.mu.Unlock()
	return nil
}

func (p *TestAgentProvider) DefineOutcome(
	_ context.Context,
	providerSessionID string,
	opts agents.DefineOutcomeOptions,
) error {
	call := AgentProviderDefineOutcomeCall{
		ProviderSessionID: providerSessionID,
		Options:           opts,
	}

	p.mu.Lock()
	session := p.sessions[providerSessionID]
	p.defineOutcomeCalls = append(p.defineOutcomeCalls, call)
	err := p.defineOutcomeErr
	handler := p.defineOutcomeHandler
	events := cloneProviderEvents(p.defineOutcomeEvents)
	p.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	if err != nil {
		return err
	}

	if handler != nil {
		handlerEvents, err := handler(call)
		if err != nil {
			return err
		}

		return p.QueueEvents(providerSessionID, handlerEvents...)
	}

	return p.QueueEvents(providerSessionID, events...)
}

func (p *TestAgentProvider) StreamEvents(
	ctx context.Context,
	providerSessionID string,
	onEvent func(agents.ProviderEvent) error,
) error {
	p.mu.Lock()
	session := p.sessions[providerSessionID]
	err := p.streamErr
	p.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item, ok := <-session.events:
			if !ok {
				return nil
			}

			if item.err != nil {
				return item.err
			}

			if err := onEvent(item.event); err != nil {
				return err
			}

			if isTerminalAgentProviderEvent(item.event.Type) {
				return nil
			}
		}
	}
}

func (p *TestAgentProvider) DeleteSession(_ context.Context, providerSessionID string) error {
	call := AgentProviderDeleteSessionCall{ProviderSessionID: providerSessionID}

	p.mu.Lock()
	session := p.sessions[providerSessionID]
	p.deleteSessionCalls = append(p.deleteSessionCalls, call)
	err := p.deleteSessionErr
	p.mu.Unlock()

	if session == nil {
		return fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	if err != nil {
		return err
	}

	session.mu.Lock()
	session.session.Deleted = true
	session.mu.Unlock()

	p.CloseStream(providerSessionID)
	return nil
}

func (p *TestAgentProvider) QueueEvents(providerSessionID string, events ...agents.ProviderEvent) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := session.enqueue(agentProviderStreamItem{event: event}); err != nil {
			return err
		}
	}

	return nil
}

func (p *TestAgentProvider) QueueError(providerSessionID string, err error) error {
	session, sessionErr := p.session(providerSessionID)
	if sessionErr != nil {
		return sessionErr
	}

	return session.enqueue(agentProviderStreamItem{err: err})
}

func (p *TestAgentProvider) CloseStream(providerSessionID string) error {
	session, err := p.session(providerSessionID)
	if err != nil {
		return err
	}

	session.close()
	return nil
}

func (p *TestAgentProvider) Session(providerSessionID string) (AgentProviderSession, bool) {
	p.mu.Lock()
	session := p.sessions[providerSessionID]
	p.mu.Unlock()

	if session == nil {
		return AgentProviderSession{}, false
	}

	session.mu.Lock()
	defer session.mu.Unlock()
	return session.session, true
}

func (p *TestAgentProvider) CreateSessionCalls() []AgentProviderCreateSessionCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]AgentProviderCreateSessionCall(nil), p.createSessionCalls...)
}

func (p *TestAgentProvider) SendMessageCalls() []AgentProviderSendMessageCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]AgentProviderSendMessageCall(nil), p.sendMessageCalls...)
}

func (p *TestAgentProvider) InterruptSessionCalls() []AgentProviderInterruptSessionCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]AgentProviderInterruptSessionCall(nil), p.interruptSessionCalls...)
}

func (p *TestAgentProvider) DefineOutcomeCalls() []AgentProviderDefineOutcomeCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]AgentProviderDefineOutcomeCall(nil), p.defineOutcomeCalls...)
}

func (p *TestAgentProvider) DeleteSessionCalls() []AgentProviderDeleteSessionCall {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]AgentProviderDeleteSessionCall(nil), p.deleteSessionCalls...)
}

func (p *TestAgentProvider) session(providerSessionID string) (*agentProviderSessionState, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	session := p.sessions[providerSessionID]
	if session == nil {
		return nil, fmt.Errorf("agent provider session %q not found", providerSessionID)
	}

	return session, nil
}

func (s *agentProviderSessionState) enqueue(item agentProviderStreamItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return agents.ErrSessionAlreadyTerminated
	}

	select {
	case s.events <- item:
		return nil
	default:
		return fmt.Errorf("agent provider stream buffer is full")
	}
}

func (s *agentProviderSessionState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	close(s.events)
	s.closed = true
}

func cloneProviderEvents(events []agents.ProviderEvent) []agents.ProviderEvent {
	return append([]agents.ProviderEvent(nil), events...)
}

func isTerminalAgentProviderEvent(eventType agents.ProviderEventType) bool {
	return eventType == agents.ProviderEventTurnCompleted ||
		eventType == agents.ProviderEventSessionFailed
}
