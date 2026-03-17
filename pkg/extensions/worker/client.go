package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

type JobHandler func(context.Context, protocol.JobAssignMessage) (json.RawMessage, error)

type ClientConfig struct {
	HubURL            string
	WorkerID          string
	RegistrationToken string
	ReconnectDelay    time.Duration
}

type Client struct {
	config      ClientConfig
	wsDialer    *websocket.Dialer
	handleJob   JobHandler
	sendMu      sync.Mutex
	currentConn messageConn
}

type messageConn interface {
	WriteJSON(v any) error
	WriteMessage(messageType int, data []byte) error
	Close() error
}

func NewClient(config ClientConfig, handleJob JobHandler) *Client {
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = time.Second
	}

	if handleJob == nil {
		handleJob = func(_ context.Context, _ protocol.JobAssignMessage) (json.RawMessage, error) {
			return nil, fmt.Errorf("job execution is not implemented")
		}
	}

	return &Client{
		config:    config,
		wsDialer:  websocket.DefaultDialer,
		handleJob: handleJob,
	}
}

func (c *Client) Run(ctx context.Context) error {
	registration := protocol.Registration{
		WorkerID: c.config.WorkerID,
	}
	if registration.WorkerID == "" {
		return fmt.Errorf("worker ID is required")
	}
	if c.config.RegistrationToken == "" {
		return fmt.Errorf("registration token is required")
	}

	for {
		err := c.runOnce(ctx, registration)
		if ctx.Err() != nil {
			return nil
		}
		if err == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(c.config.ReconnectDelay):
		}
	}
}

func (c *Client) runOnce(ctx context.Context, registration protocol.Registration) error {
	wsURL, err := joinWebsocketURL(c.config.HubURL, "/api/v1/register")
	if err != nil {
		return err
	}

	wsURL, err = addQuery(wsURL, map[string]string{
		protocol.QueryWorkerID: registration.WorkerID,
		protocol.QueryToken:    c.config.RegistrationToken,
	})

	if err != nil {
		return err
	}

	conn, _, err := c.wsDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	c.sendMu.Lock()
	c.currentConn = conn
	c.sendMu.Unlock()
	defer func() {
		c.sendMu.Lock()
		c.currentConn = nil
		c.sendMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"))
			return nil
		default:
		}

		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		if err := c.handleMessage(ctx, payload); err != nil {
			return err
		}
	}
}

func (c *Client) handleMessage(ctx context.Context, payload []byte) error {
	var envelope protocol.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode hub message: %w", err)
	}

	switch envelope.Type {
	case protocol.MessageTypePing:
		return c.writeMessage(protocol.NewPong())

	case protocol.MessageTypeJobAssign:
		var message protocol.JobAssignMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			return err
		}

		go c.processJob(ctx, message)
		return nil

	case protocol.MessageTypeJobCancel:
		return nil

	default:
		return nil
	}
}

func (c *Client) processJob(ctx context.Context, message protocol.JobAssignMessage) {
	output, err := c.handleJob(ctx, message)
	if err != nil {
		_ = c.writeMessage(protocol.NewJobFailure(message.JobID, "job_execution_failed", err.Error()))
		return
	}

	_ = c.writeMessage(protocol.NewJobSuccess(message.JobID, output))
}

func (c *Client) writeMessage(message any) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	if c.currentConn == nil {
		return fmt.Errorf("worker is not connected")
	}

	return c.currentConn.WriteJSON(message)
}

func joinWebsocketURL(base string, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	}

	if strings.HasPrefix(path, "/") {
		parsed.Path = path
	} else {
		parsed.Path = "/" + path
	}

	return parsed.String(), nil
}

func addQuery(rawURL string, values map[string]string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	query := parsed.Query()
	for key, value := range values {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}
