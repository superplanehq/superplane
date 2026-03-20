package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/superplanehq/superplane/pkg/extensions/hub/protocol"
)

type JobHandler func(context.Context, protocol.JobAssignMessage) (json.RawMessage, error)

type ClientConfig struct {
	HubURL               string
	RegistrationToken    string
	ReconnectDelay       time.Duration
	ReconnectMaxAttempts int
}

type Runner struct {
	hubURL            *url.URL
	registrationToken string

	//
	// Related to Websocket management.
	//
	wsDialer             *websocket.Dialer
	sendMu               sync.Mutex
	reconnectDelay       time.Duration
	reconnectAttempts    int
	reconnectMaxAttempts int

	//
	// Related to job handling.
	//
	handleJob JobHandler
}

func New(config ClientConfig, handleJob JobHandler) (*Runner, error) {
	url, err := url.Parse(config.HubURL)
	if err != nil {
		return nil, fmt.Errorf("invalid HUB_URL: %w", err)
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		return nil, fmt.Errorf("invalid HUB_URL: scheme must be http or https")
	}

	if config.RegistrationToken == "" {
		return nil, fmt.Errorf("REGISTRATION_TOKEN is required")
	}

	reconnectDelay := config.ReconnectDelay
	if reconnectDelay <= 0 {
		reconnectDelay = time.Second
	}

	reconnectMaxAttempts := config.ReconnectMaxAttempts
	if reconnectMaxAttempts <= 0 {
		reconnectMaxAttempts = 60
	}

	if handleJob == nil {
		return nil, fmt.Errorf("handleJob is required")
	}

	return &Runner{
		hubURL:               url,
		registrationToken:    config.RegistrationToken,
		handleJob:            handleJob,
		reconnectDelay:       reconnectDelay,
		reconnectMaxAttempts: reconnectMaxAttempts,
		wsDialer: &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 10 * time.Second,
		},
	}, nil
}

func (c *Runner) Run(ctx context.Context) error {
	for {
		log.Printf("Registering with hub...")

		conn, err := c.register(ctx)
		if err != nil {
			return err
		}

		log.Printf("Registered with hub, processing messages")
		err = c.processMessages(ctx, conn)

		//
		// If no error is returned, it means the ctx was interrupted.
		// Just exit the loop.
		//
		if err == nil {
			return nil
		}

		//
		// Otherwise, connection was lost, so we handle reconnection logic.
		//
		if c.reconnectAttempts >= c.reconnectMaxAttempts {
			return fmt.Errorf("max reconnect attempts (%d) reached", c.reconnectMaxAttempts)
		}

		c.reconnectAttempts++
		log.Printf("Reconnecting to hub in %s - %d/%d", c.reconnectDelay, c.reconnectAttempts, c.reconnectMaxAttempts)
		time.Sleep(c.reconnectDelay)
	}
}

func (c *Runner) registrationURL() string {
	wsURL := *c.hubURL
	wsURL.Scheme = c.wsScheme()
	wsURL.Path = "/api/v1/register"

	query := wsURL.Query()
	query.Set(protocol.QueryToken, c.registrationToken)
	wsURL.RawQuery = query.Encode()

	return wsURL.String()
}

func (c *Runner) register(ctx context.Context) (*websocket.Conn, error) {
	wsURL := c.registrationURL()
	conn, _, err := c.wsDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial websocket: %w", err)
	}

	return conn, nil
}

func (c *Runner) processMessages(ctx context.Context, conn *websocket.Conn) error {
	for {
		select {

		//
		// If the agent is interrupted,
		// close the connection and return.
		//
		case <-ctx.Done():
			log.Printf("Interrupted, closing connection")
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "shutdown"))
			return nil

		default:
			_, payload, err := conn.ReadMessage()
			if err != nil {
				return err
			}

			if err := c.handleMessage(ctx, conn, payload); err != nil {
				return err
			}
		}
	}
}

func (c *Runner) handleMessage(ctx context.Context, conn *websocket.Conn, payload []byte) error {
	var envelope protocol.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return fmt.Errorf("decode hub message: %w", err)
	}

	switch envelope.Type {
	case protocol.MessageTypePing:
		log.Printf("Received ping message")
		return c.writeMessage(conn, protocol.NewPong())

	case protocol.MessageTypeJobAssign:
		log.Printf("Received job assign message")
		var message protocol.JobAssignMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			return err
		}

		go c.processJob(ctx, conn, message)
		return nil

	case protocol.MessageTypeJobCancel:
		log.Printf("Received job cancel message")
		return nil

	default:
		log.Printf("Received unknown message: %s", envelope.Type)
		return nil
	}
}

func (c *Runner) processJob(ctx context.Context, conn *websocket.Conn, message protocol.JobAssignMessage) {
	output, err := c.handleJob(ctx, message)
	if err != nil {
		log.Printf("Error handling job: %v", err)
		_ = c.writeMessage(conn,
			protocol.NewFailedJobOutput(message.JobID, message.JobType, &protocol.JobError{
				Code:    "error",
				Message: err.Error(),
			}),
		)
		return
	}

	log.Printf("Job %s completed successfully", message.JobID)
	_ = c.writeMessage(conn, protocol.NewSuccessfulJobOutput(message.JobID, message.JobType, output))
}

func (c *Runner) writeMessage(conn *websocket.Conn, message any) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	return conn.WriteJSON(message)
}

func (c *Runner) wsScheme() string {
	if c.hubURL.Scheme == "http" {
		return "ws"
	}

	return "wss"
}
