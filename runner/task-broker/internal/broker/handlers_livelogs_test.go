package broker

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/superplane/runner/shared/models"
	"github.com/superplane/runner/task-broker/internal/livelogs"
	"github.com/superplane/runner/task-broker/internal/store"
)

func TestCloudWatchLiveLogsEndAfterTerminalTaskCatchesUp(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	for _, status := range []models.TaskStatus{models.StatusSucceeded, models.StatusFailed, models.StatusCanceled} {
		t.Run(string(status), func(t *testing.T) {
			calls := 0
			cw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/x-amz-json-1.1")
				switch calls {
				case 1:
					_, _ = io.WriteString(w, `{"events":[{"message":"first"}],"nextForwardToken":"page-1"}`)
				case 2:
					// An empty page with an advancing token is not caught up.
					_, _ = io.WriteString(w, `{"events":[],"nextForwardToken":"page-2"}`)
				case 3:
					_, _ = io.WriteString(w, `{"events":[{"message":"last"}],"nextForwardToken":"tail"}`)
				default:
					_, _ = io.WriteString(w, `{"events":[],"nextForwardToken":"tail"}`)
				}
			}))
			defer cw.Close()
			t.Setenv("AWS_ENDPOINT_URL", cw.URL)
			st := &liveLogTaskStore{task: &models.Task{ID: "task-1", Status: models.StatusClaimed}, terminalStatus: status}
			s := &Server{Store: st, TaskCloudWatchLogGroup: "tasks", TaskCloudWatchRegion: "us-east-1"}
			ts := httptest.NewServer(NewRouter(s, RouterOptions{AuthToken: "test-token"}))
			defer ts.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/v1/tasks/task-1/live-logs", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("stream did not end cleanly: %v", err)
			}
			if got, want := string(body), "{\"text\":\"first\",\"type\":\"line\"}\n{\"text\":\"last\",\"type\":\"line\"}\n"; got != want {
				t.Fatalf("body = %q, want %q", got, want)
			}
			if resp.StatusCode != http.StatusOK || resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Accel-Buffering") != "no" {
				t.Fatalf("status=%d headers=%v", resp.StatusCode, resp.Header)
			}
		})
	}
}

type liveLogTaskStore struct {
	store.Store
	task           *models.Task
	terminalStatus models.TaskStatus
	lookups        int
}

func (s *liveLogTaskStore) GetTask(context.Context, string) (*models.Task, error) {
	s.lookups++
	task := *s.task
	if s.lookups > 1 && s.terminalStatus != "" {
		task.Status = s.terminalStatus
	}
	return &task, nil
}

func TestLocalLiveLogsHeartbeatWhileRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		hub := livelogs.NewHub()
		s := &Server{Store: &liveLogTaskStore{task: &models.Task{ID: "task-1", Status: models.StatusClaimed}}, LiveLogs: hub}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		req := httptest.NewRequest(http.MethodGet, "/v1/tasks/task-1/live-logs", nil).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer test-token")
		w := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			NewRouter(s, RouterOptions{AuthToken: "test-token"}).ServeHTTP(w, req)
			close(done)
		}()
		time.Sleep(15 * time.Second)
		synctest.Wait()
		if w.Body.String() != "{\"type\":\"ping\"}\n" || !w.Flushed {
			t.Fatalf("quiet stream: body=%q flushed=%v", w.Body.String(), w.Flushed)
		}
		select {
		case <-done:
			t.Fatal("running stream ended")
		default:
		}
		hub.Append("task-1", []byte("working\n"))
		synctest.Wait()
		time.Sleep(14 * time.Second)
		hub.Append("task-1", []byte("still working\n"))
		synctest.Wait()
		time.Sleep(14 * time.Second)
		synctest.Wait()
		if strings.Count(w.Body.String(), `"type":"ping"`) != 1 {
			t.Fatalf("heartbeat emitted while lines were flowing: %q", w.Body.String())
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if strings.Count(w.Body.String(), `"type":"ping"`) != 2 {
			t.Fatalf("heartbeat did not resume after log activity: %q", w.Body.String())
		}
		// Subsequent log data still arrives and the hub closes with clean EOF.
		hub.Append("task-1", []byte("finished\n"))
		hub.Close("task-1")
		<-done
		if !strings.Contains(w.Body.String(), `"text":"finished"`) || strings.Contains(w.Body.String(), `"type":"error"`) {
			t.Fatalf("body = %q", w.Body.String())
		}
	})
}

func TestLocalLiveLogsHeartbeatWriteFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failure := errors.New("connection closed")
		ctx, cancel := context.WithTimeout(context.Background(), 16*time.Second)
		defer cancel()
		err := streamLocalLiveLogs(ctx, failingLiveLogWriter{failure}, nil, livelogs.NewHub(), "task-1")
		if !errors.Is(err, failure) {
			t.Fatalf("error = %v", err)
		}
	})
}

type failingLiveLogWriter struct{ err error }

func (w failingLiveLogWriter) Write([]byte) (int, error) { return 0, w.err }
