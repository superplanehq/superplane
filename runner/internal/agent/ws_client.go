package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/superplane/runner/shared/wsrunner"
)

const wsReconnectDelay = 2 * time.Second

const wsClientWriteWait = 10 * time.Second

const wsClientReadIdle = 90 * time.Second

func fleetStreamURL(base string) (string, error) {
	b := strings.TrimSpace(base)
	b = strings.TrimRight(b, "/")
	if b == "" {
		return "", errors.New("empty fleet manager base URL")
	}
	switch {
	case strings.HasPrefix(b, "https://"):
		return "wss://" + strings.TrimPrefix(b, "https://") + "/v1/runners/stream", nil
	case strings.HasPrefix(b, "http://"):
		return "ws://" + strings.TrimPrefix(b, "http://") + "/v1/runners/stream", nil
	default:
		return "", fmt.Errorf("FLEET_MANAGER_URL must start with http:// or https://")
	}
}

// transportWebSocket reports whether to use fleet-manager WebSocket (GET /v1/runners/stream).
// Default is WebSocket. Set Transport to "http", "polling", or "legacy" (case-insensitive) for HTTP claim/complete.
func transportWebSocket(c Config) bool {
	switch strings.ToLower(strings.TrimSpace(c.Transport)) {
	case "http", "polling", "legacy":
		return false
	default:
		return true
	}
}

// RunWebSocket runs the agent against fleet-manager using GET /v1/runners/stream.
// It reconnects with a fixed delay after session errors. Returns nil after one task when ExitAfterEachTask is set.
func RunWebSocket(ctx context.Context, a *Agent) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := runWebSocketSession(ctx, a)
		if err == nil {
			return nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if a.Config.Log != nil {
			a.Config.Log.Warn("fleet_manager_ws", slog.String("op", "session"), slog.Any("err", err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wsReconnectDelay):
		}
	}
}

func runWebSocketSession(ctx context.Context, a *Agent) error {
	wsURL, err := fleetStreamURL(a.Config.BaseURL)
	if err != nil {
		return err
	}
	dialer := websocket.Dialer{}
	hdr := http.Header{}
	if t := strings.TrimSpace(a.Config.Token); t != "" {
		hdr.Set("Authorization", "Bearer "+strings.TrimSpace(t))
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, hdr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if a.Config.Log != nil {
		a.Config.Log.Info("fleet_manager_ws", slog.String("op", "connect"), slog.String("url", wsURL))
	}

	var writeMu sync.Mutex

	// Reset read deadline on every server ping so idle runners stay connected
	// beyond the initial wsClientReadIdle window (fleet-manager pings every 30s).
	conn.SetPingHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(wsClientReadIdle))
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsClientWriteWait))
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})
	// Set an initial read deadline so the connection is not open-ended before the first task.
	_ = conn.SetReadDeadline(time.Now().Add(wsClientReadIdle))

	hello := wsrunner.Hello{
		Type:         wsrunner.TypeHello,
		RunnerID:     a.Config.RunnerID,
		LeaseSeconds: int((10 * time.Minute).Seconds()),
	}
	writeMu.Lock()
	helloErr := conn.WriteJSON(hello)
	writeMu.Unlock()
	if helloErr != nil {
		return fmt.Errorf("hello: %w", helloErr)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var taskMsg wsrunner.Task
		if err := conn.ReadJSON(&taskMsg); err != nil {
			return fmt.Errorf("read task: %w", err)
		}
		if taskMsg.Type != wsrunner.TypeTask || taskMsg.Task == nil {
			return fmt.Errorf("unexpected message type %q", taskMsg.Type)
		}
		task := taskMsg.Task

		pushCh := make(chan struct{}, 1)
		readCtx, readStop := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		go wsCtrlReadDuringExecute(readCtx, conn, task.ID, pushCh, &wg, &writeMu)

		exit, _, runErr, userCanceled, result := a.execute(ctx, a.fleetBase(), task, pushCh)

		readStop()
		_ = conn.SetReadDeadline(time.Now().Add(time.Millisecond))
		wg.Wait()
		_ = conn.SetReadDeadline(time.Now().Add(wsClientReadIdle))

		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}
		comp := wsrunner.Complete{
			Type:     wsrunner.TypeComplete,
			TaskID:   task.ID,
			RunnerID: a.Config.RunnerID,
			ExitCode: exit,
			Error:    errMsg,
			Canceled: userCanceled,
			Result:   result,
		}
		writeMu.Lock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsClientWriteWait))
		werr := conn.WriteJSON(comp)
		writeMu.Unlock()
		if werr != nil {
			return fmt.Errorf("write complete: %w", werr)
		}

		_, raw, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read complete reply: %w", err)
		}
		var disc struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &disc); err != nil {
			return fmt.Errorf("decode reply: %w", err)
		}
		switch disc.Type {
		case wsrunner.TypeError:
			var wse wsrunner.Error
			if err := json.Unmarshal(raw, &wse); err != nil {
				return err
			}
			return fmt.Errorf("complete failed: %d %s", wse.Code, wse.Message)
		case wsrunner.TypeAck:
			if a.Config.Log != nil {
				a.Config.Log.Info("fleet_manager_ws",
					slog.String("op", "complete_task"),
					slog.String("runner_id", a.Config.RunnerID),
					slog.String("task_id", task.ID),
				)
			}
		default:
			return fmt.Errorf("unexpected reply type %q", disc.Type)
		}

		if a.Config.ExitAfterEachTask {
			return nil
		}
	}
}

// wsCtrlReadDuringExecute reads control frames and server push cancel while execute runs.
// Exactly one goroutine may call ReadMessage on conn until this returns.
func wsCtrlReadDuringExecute(readCtx context.Context, conn *websocket.Conn, taskID string, pushCh chan<- struct{}, wg *sync.WaitGroup, writeMu *sync.Mutex) {
	defer wg.Done()
	for {
		select {
		case <-readCtx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(wsClientReadIdle))
		mt, r, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch mt {
		case websocket.PingMessage:
			writeMu.Lock()
			_ = conn.SetWriteDeadline(time.Now().Add(wsClientWriteWait))
			_ = conn.WriteMessage(websocket.PongMessage, nil)
			writeMu.Unlock()
		case websocket.TextMessage:
			var d struct {
				Type   string `json:"type"`
				TaskID string `json:"task_id"`
			}
			if json.Unmarshal(r, &d) != nil {
				continue
			}
			if strings.EqualFold(d.Type, wsrunner.TypeCancel) && strings.TrimSpace(d.TaskID) == taskID {
				select {
				case pushCh <- struct{}{}:
				default:
				}
			}
		}
	}
}
