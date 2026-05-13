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

func transportWebSocket(c Config) bool {
	return strings.EqualFold(strings.TrimSpace(c.Transport), "websocket")
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

		exit, out, runErr, userCanceled := a.execute(ctx, a.fleetBase(), task, pushCh)

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
			Output:   truncateString(out, a.Config.MaxOutputBytes),
			Error:    errMsg,
			Canceled: userCanceled,
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
