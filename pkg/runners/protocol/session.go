package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	defaultReconnectMinimum = time.Second
	defaultReconnectMax     = 5 * time.Second
	defaultReadTimeout      = 60 * time.Second
	defaultWriteTimeout     = 10 * time.Second
	defaultHandshakeTimeout = 15 * time.Second
	defaultMessageLimit     = 2 * 1024 * 1024
)

var (
	errReconnect = errors.New("server requested reconnect")
	errShutdown  = errors.New("server requested shutdown")
)

type SessionConfig struct {
	BaseURL          string
	Registration     Registration
	Version          string
	ReconnectMin     time.Duration
	ReconnectMax     time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	HandshakeTimeout time.Duration
	MessageLimit     int64
	Dialer           *websocket.Dialer
}

type pendingCompletion struct {
	message CompleteMessage
	result  chan error
}

type websocketFrame struct {
	messageType int
	content     []byte
}

// Session owns reconnects and serializes all writes for each WebSocket.
type Session struct {
	config SessionConfig

	mu          sync.Mutex
	currentTask string
	completion  *pendingCompletion
	shutdown    *shutdownRequestMessage

	tasks             chan json.RawMessage
	cancellations     chan string
	completionChanged chan struct{}
	shutdownChanged   chan struct{}
}

func NewSession(config SessionConfig) (*Session, error) {
	config.BaseURL = strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	config.Version = strings.TrimSpace(config.Version)
	config.Registration.RunnerID = strings.TrimSpace(config.Registration.RunnerID)
	config.Registration.FleetID = strings.TrimSpace(config.Registration.FleetID)
	config.Registration.AccessToken = strings.TrimSpace(config.Registration.AccessToken)
	if _, err := connectURL(config.BaseURL); err != nil {
		return nil, err
	}
	switch {
	case config.Registration.RunnerID == "":
		return nil, errors.New("runner ID is required")
	case config.Registration.FleetID == "":
		return nil, errors.New("runner fleet ID is required")
	case config.Registration.AccessToken == "":
		return nil, errors.New("runner access token is required")
	case config.Version == "":
		return nil, errors.New("runner version is required")
	}
	if config.ReconnectMin <= 0 {
		config.ReconnectMin = defaultReconnectMinimum
	}
	if config.ReconnectMax < config.ReconnectMin {
		config.ReconnectMax = max(config.ReconnectMin, defaultReconnectMax)
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = defaultReadTimeout
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = defaultWriteTimeout
	}
	if config.HandshakeTimeout <= 0 {
		config.HandshakeTimeout = defaultHandshakeTimeout
	}
	if config.MessageLimit <= 0 {
		config.MessageLimit = defaultMessageLimit
	}
	if config.Dialer == nil {
		config.Dialer = &websocket.Dialer{HandshakeTimeout: config.HandshakeTimeout}
	}
	return &Session{
		config:            config,
		tasks:             make(chan json.RawMessage),
		cancellations:     make(chan string, 1),
		completionChanged: make(chan struct{}, 1),
		shutdownChanged:   make(chan struct{}, 1),
	}, nil
}

func (s *Session) Tasks() <-chan json.RawMessage {
	return s.tasks
}

func (s *Session) Cancellations() <-chan string {
	return s.cancellations
}

func (s *Session) RequestShutdown(reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = ShutdownReasonSignal
	}
	s.mu.Lock()
	if s.shutdown == nil {
		s.shutdown = &shutdownRequestMessage{
			Type:      messageTypeShutdownRequest,
			RequestID: uuid.NewString(),
			Reason:    reason,
		}
	}
	s.mu.Unlock()
	s.signalShutdownChanged()
}

func (s *Session) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := s.runConnection(ctx)
		if errors.Is(err, errShutdown) {
			return nil
		}
		var protocolErr *ProtocolError
		if errors.As(err, &protocolErr) {
			return err
		}
		switch {
		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			return err
		case errors.Is(err, errReconnect):
			continue
		}

		timer := time.NewTimer(s.reconnectDelay())
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (s *Session) Complete(ctx context.Context, message CompleteMessage) error {
	message.Type = messageTypeComplete
	if strings.TrimSpace(message.RequestID) == "" {
		return errors.New("completion request ID is required")
	}
	if strings.TrimSpace(message.TaskID) == "" {
		return errors.New("completion task ID is required")
	}

	pending := &pendingCompletion{
		message: message,
		result:  make(chan error, 1),
	}
	s.mu.Lock()
	if s.completion != nil {
		s.mu.Unlock()
		return errors.New("a task completion is already pending")
	}
	s.completion = pending
	s.mu.Unlock()
	s.signalCompletionChanged()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-pending.result:
		return err
	}
}

func (s *Session) runConnection(ctx context.Context) error {
	endpoint, err := connectURL(s.config.BaseURL)
	if err != nil {
		return err
	}
	headers := http.Header{
		"Authorization": []string{"Bearer " + s.config.Registration.AccessToken},
	}
	socket, response, err := s.config.Dialer.DialContext(ctx, endpoint, headers)
	if response != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("connect runner WebSocket: %w", err)
	}
	defer socket.Close()
	connectionDone := make(chan struct{})
	defer close(connectionDone)
	go func() {
		select {
		case <-ctx.Done():
			_ = socket.Close()
		case <-connectionDone:
		}
	}()
	socket.SetReadLimit(s.config.MessageLimit)
	_ = socket.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

	outgoing := make(chan any, 16)
	writerDone := make(chan error, 1)
	writerContext, stopWriter := context.WithCancel(ctx)
	defer stopWriter()
	socket.SetPingHandler(func(content string) error {
		_ = socket.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		return queueMessage(ctx, outgoing, websocketFrame{
			messageType: websocket.PongMessage,
			content:     []byte(content),
		})
	})
	go s.writeLoop(writerContext, socket, outgoing, writerDone)

	for {
		_, raw, err := socket.ReadMessage()
		if err != nil {
			select {
			case writerErr := <-writerDone:
				if writerErr != nil {
					return writerErr
				}
			default:
			}
			return fmt.Errorf("read runner WebSocket: %w", err)
		}
		_ = socket.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))

		var envelope messageEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("decode runner WebSocket message: %w", err)
		}
		switch envelope.Type {
		case messageTypeTask:
			if err := s.receiveTask(ctx, raw); err != nil {
				return err
			}
		case messageTypeCancel:
			if err := s.receiveCancellation(ctx, raw); err != nil {
				return err
			}
		case messageTypePing:
			if err := queueMessage(ctx, outgoing, typedMessage{Type: messageTypePong}); err != nil {
				return err
			}
		case messageTypeAck:
			if err := s.receiveAcknowledgement(raw); err != nil {
				return err
			}
		case messageTypeError:
			return s.receiveProtocolError(raw)
		case messageTypeReconnect:
			return errReconnect
		case messageTypeShutdown:
			return errShutdown
		default:
			return fmt.Errorf("unexpected runner WebSocket message type %q", envelope.Type)
		}
	}
}

func (s *Session) writeLoop(
	ctx context.Context,
	socket *websocket.Conn,
	outgoing <-chan any,
	done chan<- error,
) {
	write := func(message any) error {
		_ = socket.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
		if frame, ok := message.(websocketFrame); ok {
			return socket.WriteMessage(frame.messageType, frame.content)
		}
		return socket.WriteJSON(message)
	}
	if err := write(s.hello()); err != nil {
		done <- fmt.Errorf("write runner hello: %w", err)
		_ = socket.Close()
		return
	}
	if completion := s.pendingCompletion(); completion != nil {
		if err := write(completion.message); err != nil {
			done <- fmt.Errorf("write task completion: %w", err)
			_ = socket.Close()
			return
		}
	}
	if shutdown := s.pendingShutdown(); shutdown != nil {
		if err := write(shutdown); err != nil {
			done <- fmt.Errorf("write runner shutdown request: %w", err)
			_ = socket.Close()
			return
		}
		select {
		case <-s.shutdownChanged:
		default:
		}
	}

	for {
		select {
		case <-ctx.Done():
			done <- ctx.Err()
			return
		case message := <-outgoing:
			if err := write(message); err != nil {
				done <- fmt.Errorf("write runner WebSocket: %w", err)
				_ = socket.Close()
				return
			}
		case <-s.completionChanged:
			completion := s.pendingCompletion()
			if completion == nil {
				continue
			}
			if err := write(completion.message); err != nil {
				done <- fmt.Errorf("write task completion: %w", err)
				_ = socket.Close()
				return
			}
		case <-s.shutdownChanged:
			shutdown := s.pendingShutdown()
			if shutdown == nil {
				continue
			}
			if err := write(shutdown); err != nil {
				done <- fmt.Errorf("write runner shutdown request: %w", err)
				_ = socket.Close()
				return
			}
		}
	}
}

func (s *Session) receiveTask(ctx context.Context, raw []byte) error {
	var message taskMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("decode runner task: %w", err)
	}
	var identity struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(message.Task, &identity); err != nil {
		return fmt.Errorf("decode runner task identity: %w", err)
	}
	identity.ID = strings.TrimSpace(identity.ID)
	if identity.ID == "" {
		return errors.New("runner task ID is required")
	}

	s.mu.Lock()
	switch {
	case s.currentTask == "":
		s.currentTask = identity.ID
	case s.currentTask != identity.ID:
		s.mu.Unlock()
		return fmt.Errorf(
			"received task %q while task %q is active",
			identity.ID,
			s.currentTask,
		)
	default:
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.tasks <- message.Task:
		return nil
	}
}

func (s *Session) receiveCancellation(ctx context.Context, raw []byte) error {
	var message taskControlMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("decode task cancellation: %w", err)
	}
	s.mu.Lock()
	currentTask := s.currentTask
	s.mu.Unlock()
	if message.TaskID != currentTask {
		return fmt.Errorf(
			"received cancellation for task %q while task %q is active",
			message.TaskID,
			currentTask,
		)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.cancellations <- message.TaskID:
	default:
	}
	return nil
}

func (s *Session) receiveAcknowledgement(raw []byte) error {
	var acknowledgement ackMessage
	if err := json.Unmarshal(raw, &acknowledgement); err != nil {
		return fmt.Errorf("decode task completion acknowledgement: %w", err)
	}
	s.mu.Lock()
	completion := s.completion
	if completion == nil || acknowledgement.RequestID != completion.message.RequestID {
		s.mu.Unlock()
		return fmt.Errorf(
			"unexpected task completion acknowledgement %q",
			acknowledgement.RequestID,
		)
	}
	s.completion = nil
	if s.currentTask == completion.message.TaskID {
		s.currentTask = ""
	}
	s.mu.Unlock()
	completion.result <- nil
	return nil
}

func (s *Session) receiveProtocolError(raw []byte) error {
	var message protocolErrorMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return fmt.Errorf("decode runner protocol error: %w", err)
	}
	protocolErr := &ProtocolError{
		Code:      message.Code,
		Message:   message.Message,
		RequestID: message.RequestID,
	}
	if message.RequestID == "" {
		return protocolErr
	}

	s.mu.Lock()
	completion := s.completion
	if completion != nil && completion.message.RequestID == message.RequestID {
		s.completion = nil
	}
	s.mu.Unlock()
	if completion == nil || completion.message.RequestID != message.RequestID {
		return protocolErr
	}
	completion.result <- protocolErr
	return protocolErr
}

func (s *Session) hello() helloMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return helloMessage{
		Type:          messageTypeHello,
		RunnerID:      s.config.Registration.RunnerID,
		FleetID:       s.config.Registration.FleetID,
		Version:       s.config.Version,
		CurrentTaskID: s.currentTask,
	}
}

func (s *Session) pendingCompletion() *pendingCompletion {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.completion
}

func (s *Session) pendingShutdown() *shutdownRequestMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutdown == nil {
		return nil
	}
	message := *s.shutdown
	message.CurrentTaskID = s.currentTask
	return &message
}

func (s *Session) signalCompletionChanged() {
	select {
	case s.completionChanged <- struct{}{}:
	default:
	}
}

func (s *Session) signalShutdownChanged() {
	select {
	case s.shutdownChanged <- struct{}{}:
	default:
	}
}

func (s *Session) reconnectDelay() time.Duration {
	span := s.config.ReconnectMax - s.config.ReconnectMin
	if span <= 0 {
		return s.config.ReconnectMin
	}
	return s.config.ReconnectMin + time.Duration(rand.Int64N(int64(span)+1))
}

func connectURL(baseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	switch {
	case strings.HasPrefix(baseURL, "https://"):
		return "wss://" + strings.TrimPrefix(baseURL, "https://") + "/runner/v1/connect", nil
	case strings.HasPrefix(baseURL, "http://"):
		return "ws://" + strings.TrimPrefix(baseURL, "http://") + "/runner/v1/connect", nil
	default:
		return "", errors.New("runner API URL must start with http:// or https://")
	}
}

func queueMessage(ctx context.Context, outgoing chan<- any, message any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case outgoing <- message:
		return nil
	}
}

type ProtocolError struct {
	Code      string
	Message   string
	RequestID string
}

func (e *ProtocolError) Error() string {
	if e.RequestID == "" {
		return fmt.Sprintf("runner protocol error %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf(
		"runner protocol error %s for request %s: %s",
		e.Code,
		e.RequestID,
		e.Message,
	)
}
