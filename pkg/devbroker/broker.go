// Package devbroker implements a task broker for local development.
//
// Runner components (Run Bash, Run Commands, Run Claude Code, …) hand their work
// to an external task broker, which SuperPlane Cloud provides. Self-hosted
// installations have none, so those components fail before doing anything. This
// package speaks the same HTTP contract and runs the commands on the machine it
// is deployed to, which is enough to exercise Runner components locally.
//
// It is a development tool, not a runner. Commands arrive as shell source and
// are executed as such, unsandboxed, in the process's own container — anyone who
// can reach this server can run arbitrary code as the server's user. Bind it to
// the development network only, and never deploy it anywhere reachable from the
// internet. Tasks live in memory and are lost on restart.
package devbroker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	statusQueued    = "queued"
	statusRunning   = "running"
	statusSucceeded = "succeeded"
	statusFailed    = "failed"
	statusCanceled  = "canceled"

	defaultTimeout = time.Hour

	// A runner is expected to provide these three paths. Scripts read upstream
	// canvas data from the payload file and write their structured result to the
	// result file, which the runner reports back as the task's result.
	taskDirEnv     = "SUPERPLANE_TASK_DIR"
	payloadFileEnv = "SUPERPLANE_PAYLOAD_FILE"
	resultFileEnv  = "SUPERPLANE_RESULT_FILE"

	payloadFileName = "payload.json"
	resultFileName  = "result.json"
)

// Options configures a Server.
type Options struct {
	// AuthToken must match TASK_BROKER_AUTH_TOKEN on the SuperPlane side.
	AuthToken string
	// WorkDir is where task files are materialized before commands run.
	WorkDir string
}

type command struct {
	Command string `json:"command"`
}

type file struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode,omitempty"`
}

type environmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type createTaskRequest struct {
	FleetID                 string                `json:"fleet_id"`
	Commands                []command             `json:"commands"`
	SetupCommands           []string              `json:"setup_commands"`
	Script                  string                `json:"script"`
	Files                   []file                `json:"files"`
	Environment             []environmentVariable `json:"environment"`
	WebhookURL              string                `json:"webhook_url"`
	ExecutionMode           string                `json:"execution_mode"`
	ExecutionTimeoutSeconds *int                  `json:"execution_timeout_seconds"`
}

// task is both the in-memory record and the status/webhook payload. SuperPlane
// reads the ID from "task_id" on webhooks and from "id" on status responses, so
// both are always populated.
type task struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	Status     string     `json:"status"`
	FleetID    string     `json:"fleet_id"`
	ExitCode   *int       `json:"exit_code,omitempty"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ClaimedAt  *time.Time `json:"claimed_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	// Result carries whatever the script wrote to SUPERPLANE_RESULT_FILE.
	Result json.RawMessage `json:"result,omitempty"`

	ExecutionMode   string `json:"execution_mode,omitempty"`
	CancelRequested bool   `json:"cancel_requested,omitempty"`

	webhookURL string
	cancel     chan struct{}
}

// Server keeps tasks in memory and executes them on the local machine.
type Server struct {
	options Options

	mu    sync.Mutex
	tasks map[string]*task
	next  int

	// running tracks in-flight executions so tests and shutdown can wait.
	running sync.WaitGroup
}

// New builds a Server. It does not listen; call Handler to mount it.
func New(options Options) *Server {
	return &Server{
		options: options,
		tasks:   map[string]*task{},
	}
}

// Handler routes the broker endpoints SuperPlane calls.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tasks", s.authorized(s.handleTasks))
	mux.HandleFunc("/v1/tasks/", s.authorized(s.handleTaskByID))
	return mux
}

// Wait blocks until every accepted task has finished. Intended for tests.
func (s *Server) Wait() {
	s.running.Wait()
}

func (s *Server) authorized(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.options.AuthToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.createTask(w, r)
	case http.MethodGet:
		s.listTasks(w)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTaskByID serves /v1/tasks/{id} and /v1/tasks/{id}/cancel.
func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	id, action, _ := strings.Cut(rest, "/")

	s.mu.Lock()
	t, ok := s.tasks[id]
	s.mu.Unlock()

	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	if action == "cancel" {
		s.cancelTask(t)
		w.WriteHeader(http.StatusOK)
		return
	}

	s.respondJSON(w, t)
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("decode request: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.next++
	id := fmt.Sprintf("task-%d", s.next)
	t := &task{
		ID:            id,
		TaskID:        id,
		Status:        statusQueued,
		FleetID:       req.FleetID,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: req.ExecutionMode,
		webhookURL:    req.WebhookURL,
		cancel:        make(chan struct{}),
	}
	s.tasks[id] = t
	s.mu.Unlock()

	s.running.Add(1)
	go func() {
		defer s.running.Done()
		s.run(t, req)
	}()

	// SuperPlane's broker client accepts 201 and nothing else here.
	s.respondJSONStatus(w, http.StatusCreated, map[string]string{"id": id})
}

func (s *Server) listTasks(w http.ResponseWriter) {
	s.mu.Lock()
	tasks := make([]*task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	s.mu.Unlock()

	s.respondJSON(w, map[string]any{"tasks": tasks})
}

func (s *Server) cancelTask(t *task) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.Status == statusQueued || t.Status == statusRunning {
		t.CancelRequested = true
		close(t.cancel)
	}
}

// run materializes the task's files, executes its commands in order, and reports
// the outcome back through the webhook.
func (s *Server) run(t *task, req createTaskRequest) {
	dir, err := os.MkdirTemp(s.options.WorkDir, "task-")
	if err != nil {
		s.finish(t, statusFailed, nil, "", fmt.Sprintf("create task dir: %v", err))
		return
	}
	defer os.RemoveAll(dir)

	if err := writeFiles(dir, req.Files); err != nil {
		s.finish(t, statusFailed, nil, "", err.Error())
		return
	}

	// Scripts read the payload unconditionally, so it must exist even when
	// SuperPlane sent no upstream data.
	if err := ensurePayloadFile(dir); err != nil {
		s.finish(t, statusFailed, nil, "", err.Error())
		return
	}

	s.mu.Lock()
	now := time.Now().UTC()
	t.Status = statusRunning
	t.ClaimedAt = &now
	s.mu.Unlock()

	// Names only — values are credentials. Which variables SuperPlane passed is
	// the first thing you need when a task fails to authenticate.
	names := make([]string, 0, len(req.Environment))
	for _, variable := range req.Environment {
		names = append(names, variable.Name)
	}
	log.Printf("task %s: %d command(s), %d file(s), environment: %s",
		t.ID, len(req.Commands), len(req.Files), strings.Join(names, ", "))

	output, exitCode, err := s.execute(t, dir, req)
	result := readResultFile(dir)

	if t.CancelRequested {
		s.finish(t, statusCanceled, &exitCode, output, "")
		return
	}

	s.mu.Lock()
	t.Result = result
	s.mu.Unlock()

	status := statusSucceeded
	message := ""
	if err != nil {
		message = err.Error()
	}
	if exitCode != 0 || err != nil {
		status = statusFailed
	}

	s.finish(t, status, &exitCode, output, message)
}

// execute runs every command in one shell session per command, stopping at the
// first failure the way a CI step would.
func (s *Server) execute(t *task, dir string, req createTaskRequest) (string, int, error) {
	timeout := defaultTimeout
	if req.ExecutionTimeoutSeconds != nil && *req.ExecutionTimeoutSeconds > 0 {
		timeout = time.Duration(*req.ExecutionTimeoutSeconds) * time.Second
	}
	deadline := time.After(timeout)

	environment := append(os.Environ(),
		taskDirEnv+"="+dir,
		payloadFileEnv+"="+filepath.Join(dir, payloadFileName),
		resultFileEnv+"="+filepath.Join(dir, resultFileName),
	)
	for _, variable := range req.Environment {
		environment = append(environment, variable.Name+"="+variable.Value)
	}

	commands := make([]string, 0, len(req.SetupCommands)+len(req.Commands)+1)
	commands = append(commands, req.SetupCommands...)
	for _, c := range req.Commands {
		commands = append(commands, c.Command)
	}
	if script := strings.TrimSpace(req.Script); script != "" {
		commands = append(commands, script)
	}

	var output strings.Builder
	for _, c := range commands {
		if strings.TrimSpace(c) == "" {
			continue
		}

		cmd := exec.Command("bash", "-lc", c)
		cmd.Dir = dir
		cmd.Env = environment
		cmd.Stdout = &output
		cmd.Stderr = &output

		if err := cmd.Start(); err != nil {
			return output.String(), 1, fmt.Errorf("start command: %w", err)
		}

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case err := <-done:
			if code := cmd.ProcessState.ExitCode(); code != 0 {
				return output.String(), code, nil
			}
			if err != nil {
				return output.String(), 1, err
			}
		case <-t.cancel:
			_ = cmd.Process.Kill()
			return output.String(), 1, nil
		case <-deadline:
			_ = cmd.Process.Kill()
			return output.String(), 1, fmt.Errorf("execution timed out after %s", timeout)
		}
	}

	return output.String(), 0, nil
}

func (s *Server) finish(t *task, status string, exitCode *int, output, message string) {
	s.mu.Lock()
	now := time.Now().UTC()
	t.Status = status
	t.ExitCode = exitCode
	t.Output = output
	t.Error = message
	t.FinishedAt = &now
	payload := *t
	s.mu.Unlock()

	if t.webhookURL == "" {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	res, err := http.Post(t.webhookURL, "application/json", strings.NewReader(string(body)))
	if err != nil {
		return
	}
	res.Body.Close()
}

// ensurePayloadFile creates an empty payload when SuperPlane didn't send one,
// so scripts that read it unconditionally don't fail on a missing file.
func ensurePayloadFile(dir string) error {
	path := filepath.Join(dir, payloadFileName)
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		return fmt.Errorf("write payload file: %w", err)
	}

	return nil
}

// readResultFile returns the script's structured result, or nil when it wrote
// nothing valid — the result is optional and must never fail the task.
func readResultFile(dir string) json.RawMessage {
	content, err := os.ReadFile(filepath.Join(dir, resultFileName))
	if err != nil || len(content) == 0 {
		return nil
	}

	if !json.Valid(content) {
		return nil
	}

	return json.RawMessage(content)
}

func writeFiles(dir string, files []file) error {
	for _, f := range files {
		path := filepath.Join(dir, filepath.Clean("/"+f.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", f.Path, err)
		}

		mode := os.FileMode(0o644)
		if f.Mode != "" {
			var parsed uint32
			if _, err := fmt.Sscanf(f.Mode, "%o", &parsed); err == nil {
				mode = os.FileMode(parsed)
			}
		}

		if err := os.WriteFile(path, []byte(f.Content), mode); err != nil {
			return fmt.Errorf("write %s: %w", f.Path, err)
		}
	}

	return nil
}

func (s *Server) respondJSON(w http.ResponseWriter, payload any) {
	s.respondJSONStatus(w, http.StatusOK, payload)
}

func (s *Server) respondJSONStatus(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "encode response", http.StatusInternalServerError)
	}
}
