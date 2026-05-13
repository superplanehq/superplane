package fleetmanager

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/shared/wsrunner"
)

// runnerStreamUpgrader allows local httptest and same-origin upgrades. Tighten CheckOrigin for public deployments.
var runnerStreamUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

const (
	runnerStreamReadIdle   = 90 * time.Second
	runnerStreamWriteWait  = 10 * time.Second
	runnerStreamPingPeriod = 30 * time.Second
)

// runnerStream handles GET /v1/runners/stream (WebSocket). Requires TaskNotify to be set.
func (s *Server) runnerStream(w http.ResponseWriter, r *http.Request) {
	if s.TaskNotify == nil {
		writeError(w, http.StatusServiceUnavailable, "websocket runner stream unavailable")
		return
	}

	conn, err := runnerStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		if s.Log != nil {
			s.Log.Warn("runner stream upgrade", slog.Any("err", err))
		}
		return
	}
	defer func() { _ = conn.Close() }()

	writeMu := new(sync.Mutex)

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(runnerStreamReadIdle))
	})
	_ = conn.SetReadDeadline(time.Now().Add(runnerStreamReadIdle))

	_, raw, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &disc); err != nil {
		_ = writeWSError(writeMu, conn, http.StatusBadRequest, "invalid json")
		return
	}
	if disc.Type != wsrunner.TypeHello {
		_ = writeWSError(writeMu, conn, http.StatusBadRequest, "expected hello")
		return
	}
	var hello wsrunner.Hello
	if err := json.Unmarshal(raw, &hello); err != nil {
		_ = writeWSError(writeMu, conn, http.StatusBadRequest, "invalid hello")
		return
	}
	runnerID := strings.TrimSpace(hello.RunnerID)
	if runnerID == "" {
		_ = writeWSError(writeMu, conn, http.StatusBadRequest, "runner_id required")
		return
	}
	lease := time.Duration(hello.LeaseSeconds) * time.Second
	if lease <= 0 {
		lease = 5 * time.Minute
	}

	notifyCh := s.TaskNotify.Register()
	defer s.TaskNotify.Unregister(notifyCh)

	pingTicker := time.NewTicker(runnerStreamPingPeriod)
	defer pingTicker.Stop()

	ctx := r.Context()
	for {
		task, err := s.Store.ClaimTask(ctx, runnerID, lease)
		if err != nil {
			_ = writeWSError(writeMu, conn, http.StatusInternalServerError, "could not claim task")
			return
		}
		if task != nil {
			if err := s.runnerStreamOneTask(conn, ctx, task, runnerID, writeMu); err != nil {
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(runnerStreamReadIdle))
			continue
		}

		select {
		case <-notifyCh:
		case <-pingTicker.C:
			writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(runnerStreamWriteWait))
			err := conn.WriteMessage(websocket.PingMessage, nil)
			writeMu.Unlock()
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *Server) runnerStreamOneTask(conn *websocket.Conn, ctx context.Context, task *models.Task, runnerID string, writeMu *sync.Mutex) error {
	rid := strings.TrimSpace(runnerID)
	if s.RunnerCancel != nil && rid != "" {
		unreg := s.RunnerCancel.Register(rid, task.ID, conn, writeMu)
		defer unreg()
	}

	payload := api.TaskPayloadFrom(task)
	if err := wsWriteJSON(writeMu, conn, wsrunner.Task{Type: wsrunner.TypeTask, Task: payload}); err != nil {
		return err
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(runnerStreamReadIdle))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var disc struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &disc); err != nil {
			_ = writeWSError(writeMu, conn, http.StatusBadRequest, "invalid json")
			continue
		}
		if disc.Type != wsrunner.TypeComplete {
			_ = writeWSError(writeMu, conn, http.StatusBadRequest, "expected complete")
			continue
		}
		var comp wsrunner.Complete
		if err := json.Unmarshal(raw, &comp); err != nil {
			_ = writeWSError(writeMu, conn, http.StatusBadRequest, "invalid complete")
			continue
		}
		if strings.TrimSpace(comp.TaskID) != task.ID {
			_ = writeWSError(writeMu, conn, http.StatusBadRequest, "task_id mismatch")
			continue
		}
		compRunnerID := strings.TrimSpace(comp.RunnerID)
		if compRunnerID == "" {
			_ = writeWSError(writeMu, conn, http.StatusBadRequest, "runner_id required")
			continue
		}

		req := api.CompleteTaskRequest{
			RunnerID: compRunnerID,
			ExitCode: comp.ExitCode,
			Output:   comp.Output,
			Error:    comp.Error,
			Canceled: comp.Canceled,
		}
		_, cerr := s.completeTaskCore(ctx, task.ID, compRunnerID, req)
		if cerr != nil {
			code := http.StatusInternalServerError
			msg := "could not complete task"
			if strings.Contains(cerr.Error(), "not found") || strings.Contains(cerr.Error(), "wrong runner") {
				code = http.StatusConflict
				msg = "cannot complete task"
			}
			_ = writeWSError(writeMu, conn, code, msg)
			continue
		}

		if err := wsWriteJSON(writeMu, conn, wsrunner.Ack{Type: wsrunner.TypeAck, OK: true}); err != nil {
			return err
		}
		return nil
	}
}

func wsWriteJSON(writeMu *sync.Mutex, conn *websocket.Conn, v any) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(runnerStreamWriteWait))
	return conn.WriteJSON(v)
}

func writeWSError(writeMu *sync.Mutex, conn *websocket.Conn, code int, message string) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(runnerStreamWriteWait))
	return conn.WriteJSON(wsrunner.Error{
		Type:    wsrunner.TypeError,
		Code:    code,
		Message: message,
	})
}
