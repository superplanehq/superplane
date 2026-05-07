package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/superplane/runner/shared/api"
	"github.com/superplane/runner/shared/models"
)

// Config controls runner behavior.
type Config struct {
	BaseURL   string
	RunnerID  string
	Token     string
	PollEmpty time.Duration
	// MaxOutputBytes caps combined stdout+stderr stored and sent back.
	MaxOutputBytes int
	// ExitAfterEachTask stops the runner process after one successful CompleteTask once fleet-manager accepts the result.
	// Fleet-manager terminates the EC2 instance when runner_id is the instance id (see cloud-init user-data). Local env: RUNNER_TERMINATE_AFTER_EACH_TASK.
	ExitAfterEachTask bool
}

// DefaultConfig returns safe defaults.
func DefaultConfig() Config {
	return Config{
		PollEmpty:      time.Second,
		MaxOutputBytes: 512 * 1024,
	}
}

// Agent polls fleet-manager, executes tasks, and reports results.
type Agent struct {
	HTTP   *http.Client
	Config Config
}

// Run blocks until ctx is cancelled, processing tasks in a loop.
func (a *Agent) Run(ctx context.Context) error {
	if a.HTTP == nil {
		a.HTTP = http.DefaultClient
	}
	if a.Config.MaxOutputBytes <= 0 {
		a.Config.MaxOutputBytes = 512 * 1024
	}
	if a.Config.PollEmpty <= 0 {
		a.Config.PollEmpty = time.Second
	}
	base := strings.TrimRight(a.Config.BaseURL, "/")
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		task, err := a.claim(ctx, base)
		if err != nil {
			return err
		}
		if task == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(a.Config.PollEmpty):
			}
			continue
		}
		exit, out, runErr := a.execute(ctx, task)
		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}
		if err := a.complete(ctx, base, task.ID, exit, out, errMsg); err != nil {
			return err
		}
		if a.Config.ExitAfterEachTask {
			return nil
		}
	}
}

func (a *Agent) claim(ctx context.Context, base string) (*api.TaskPayload, error) {
	body, err := json.Marshal(api.ClaimTaskRequest{
		RunnerID:     a.Config.RunnerID,
		LeaseSeconds: int((10 * time.Minute).Seconds()),
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/tasks/claim", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	a.auth(req)

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("claim: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out api.ClaimTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Task, nil
}

func (a *Agent) complete(ctx context.Context, base, id string, exit int, output, errMsg string) error {
	payload := api.CompleteTaskRequest{
		RunnerID: a.Config.RunnerID,
		ExitCode: exit,
		Output:   output,
		Error:    errMsg,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/tasks/"+id+"/complete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	a.auth(req)

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("complete: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

func (a *Agent) auth(req *http.Request) {
	if t := strings.TrimSpace(a.Config.Token); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
}

func (a *Agent) execute(ctx context.Context, task *api.TaskPayload) (int, string, error) {
	mode := models.ExecutionMode(strings.ToLower(strings.TrimSpace(task.ExecutionMode)))
	if mode == "" {
		mode = models.ExecutionHost
	}

	execCtx, cancel := context.WithTimeout(ctx, 9*time.Minute)
	defer cancel()

	switch mode {
	case models.ExecutionDocker:
		return a.runDocker(execCtx, task)
	case models.ExecutionHost:
		return a.runHost(execCtx, task)
	default:
		return 1, "", fmt.Errorf("unknown execution_mode %q", task.ExecutionMode)
	}
}

func (a *Agent) runHost(ctx context.Context, task *api.TaskPayload) (int, string, error) {
	if len(task.Commands) > 0 {
		return a.runHostShellScripts(ctx, task.Commands)
	}
	if len(task.Command) == 0 {
		return 1, "", errors.New("empty command")
	}
	cmd := exec.CommandContext(ctx, task.Command[0], task.Command[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := truncateString(buf.String(), a.Config.MaxOutputBytes)
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		return exit, out, err
	}
	return exit, out, nil
}

func (a *Agent) runHostShellScripts(ctx context.Context, scripts []string) (int, string, error) {
	return runHostShellDirectives(ctx, a.Config.MaxOutputBytes, scripts)
}

func (a *Agent) runDocker(ctx context.Context, task *api.TaskPayload) (int, string, error) {
	if strings.TrimSpace(task.DockerImage) == "" {
		return 1, "", errors.New("docker_image required")
	}
	if len(task.Commands) > 0 {
		return a.runDockerShellScripts(ctx, task.DockerImage, task.Commands)
	}
	args := []string{"run", "--rm", task.DockerImage}
	args = append(args, task.Command...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := truncateString(buf.String(), a.Config.MaxOutputBytes)
	exit := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exit = ee.ExitCode()
		} else {
			exit = 1
		}
		return exit, out, err
	}
	return exit, out, nil
}

func (a *Agent) runDockerShellScripts(ctx context.Context, image string, scripts []string) (int, string, error) {
	return runDockerShellDirectives(ctx, a.Config.MaxOutputBytes, image, scripts)
}

func truncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n…(truncated)"
}
