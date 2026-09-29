package runnerapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/jwt"
	"github.com/superplanehq/superplane/pkg/models"
	runnercontrol "github.com/superplanehq/superplane/pkg/runners/control"
)

type Server struct {
	router    *mux.Router
	signer    *jwt.Signer
	encryptor crypto.Encryptor

	mu          sync.Mutex
	httpServer  *http.Server
	connections map[uuid.UUID]*runnerConnection
}

func NewServer(signer *jwt.Signer, encryptor crypto.Encryptor) (*Server, error) {
	if signer == nil {
		return nil, fmt.Errorf("JWT signer is required")
	}
	if encryptor == nil {
		return nil, fmt.Errorf("encryptor is required")
	}

	server := &Server{
		router:      mux.NewRouter(),
		signer:      signer,
		encryptor:   encryptor,
		connections: map[uuid.UUID]*runnerConnection{},
	}
	server.registerRoutes()
	return server, nil
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) registerRoutes() {
	s.router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)
	s.router.HandleFunc("/runner/v1/register", s.registerRunner).Methods(http.MethodPost)
	s.router.Handle(
		"/runner/v1/connect",
		s.authenticateRunner(http.HandlerFunc(s.connectRunner)),
	).Methods(http.MethodGet)
	s.router.Handle(
		"/runner/v1/tasks/{task_id}/logs/chunks/{sequence}",
		s.authenticateRunner(http.HandlerFunc(s.uploadTaskLogChunk)),
	).Methods(http.MethodPut)
}

func (s *Server) Serve(address string) error {
	server := &http.Server{
		Addr:              address,
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       75 * time.Second,
	}
	s.mu.Lock()
	s.httpServer = server
	s.mu.Unlock()
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	server := s.httpServer
	s.mu.Unlock()
	var shutdownErr error
	if server != nil {
		listenerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		shutdownErr = server.Shutdown(listenerCtx)
		cancel()
	}

	s.mu.Lock()
	connections := make([]*runnerConnection, 0, len(s.connections))
	for _, connection := range s.connections {
		connections = append(connections, connection)
	}
	s.mu.Unlock()

	for _, connection := range connections {
		_ = connection.write(typedMessage{Type: messageTypeReconnect})
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		remaining := len(s.connections)
		s.mu.Unlock()
		if remaining == 0 {
			return shutdownErr
		}
		select {
		case <-ctx.Done():
			for _, connection := range connections {
				_ = connection.socket.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseServiceRestart, "service restart"),
					time.Now().Add(runnerWriteTimeout),
				)
				_ = connection.socket.Close()
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) ConnectionCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.connections)
}

func (s *Server) StartControlNotifications(ctx context.Context, gatewayID string) {
	go runnercontrol.ConsumeWithReconnect(ctx, gatewayID, s.wakeConnections)
}

func (s *Server) wakeConnections(notification runnercontrol.Notification) {
	s.mu.Lock()
	connections := make(map[uuid.UUID]*runnerConnection, len(s.connections))
	for id, connection := range s.connections {
		connections[id] = connection
	}
	s.mu.Unlock()

	for runnerID, connection := range connections {
		if notification.RunnerID != "" && notification.RunnerID != runnerID.String() {
			continue
		}
		runner, err := models.FindRunner(database.Conn(), runnerID)
		if err != nil {
			continue
		}
		if notification.FleetID != "" && runner.FleetID.String() != notification.FleetID {
			continue
		}
		if runner.State == models.RunnerStateTerminated {
			_ = connection.write(typedMessage{Type: messageTypeShutdown})
			continue
		}
		task, taskErr := runner.FindActiveTask(database.Conn())
		if taskErr == nil && task.CancelRequestedAt != nil {
			_ = connection.write(taskControlMessage{
				Type:   messageTypeCancel,
				TaskID: task.ID.String(),
			})
			continue
		}
		_ = connection.write(typedMessage{Type: messageTypePing})
	}
}

func (s *Server) trackConnection(runnerID uuid.UUID, connection *runnerConnection) {
	s.mu.Lock()
	previous := s.connections[runnerID]
	s.connections[runnerID] = connection
	s.mu.Unlock()
	if previous != nil && previous != connection {
		_ = previous.socket.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseServiceRestart, "connection replaced"),
			time.Now().Add(runnerWriteTimeout),
		)
		_ = previous.socket.Close()
	}
}

func (s *Server) untrackConnection(runnerID uuid.UUID, connection *runnerConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.connections[runnerID] == connection {
		delete(s.connections, runnerID)
	}
}
