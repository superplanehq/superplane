package superplane

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/superplane/runner/fleet-manager/internal/store"
	"github.com/superplane/runner/shared/models"
)

// RunSyncLoop pulls jobs from SuperPlane and enqueues them in the local fleet-manager store.
func RunSyncLoop(ctx context.Context, log *slog.Logger, client *Client, st store.Store, notify func()) {
	if client == nil || st == nil {
		return
	}
	interval := 2 * time.Second
	if log == nil {
		log = slog.Default()
	}

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		syncCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		resp, err := client.Sync(syncCtx)
		cancel()
		if err != nil {
			log.Warn("superplane sync", slog.Any("err", err))
			sleep(ctx, interval)
			continue
		}
		if resp == nil || !resp.Continue || resp.Job == nil {
			sleep(ctx, interval)
			continue
		}

		if err := enqueueBridgeJob(ctx, st, resp.Job); err != nil {
			log.Warn("superplane enqueue local task", slog.String("task_id", resp.Job.ID), slog.Any("err", err))
			sleep(ctx, interval)
			continue
		}
		if notify != nil {
			notify()
		}
		log.Info("superplane job enqueued locally", slog.String("task_id", resp.Job.ID))
	}
}

func enqueueBridgeJob(ctx context.Context, st store.Store, job *BridgeJob) error {
	if job == nil || strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("bridge job missing id")
	}
	spec := job.Spec

	var normalizedCmds []string
	for _, c := range spec.Commands {
		c = strings.TrimSpace(c)
		if c != "" {
			normalizedCmds = append(normalizedCmds, c)
		}
	}
	hasShell := len(normalizedCmds) > 0
	hasArgv := len(spec.Command) > 0
	if !hasShell && !hasArgv {
		return fmt.Errorf("bridge job %s has no command or commands", job.ID)
	}

	mode := models.ExecutionHost
	switch strings.ToLower(strings.TrimSpace(spec.ExecutionMode)) {
	case "", string(models.ExecutionHost):
		mode = models.ExecutionHost
	case string(models.ExecutionDocker):
		mode = models.ExecutionDocker
	default:
		mode = models.ExecutionHost
	}

	env := make([]models.EnvironmentVariable, 0, len(spec.Environment))
	for _, e := range spec.Environment {
		env = append(env, models.EnvironmentVariable{Name: e.Name, Value: e.Value})
	}

	task := &models.Task{
		ID:            strings.TrimSpace(job.ID),
		WebhookURL:    BridgeWebhookURL,
		Status:        models.StatusQueued,
		CreatedAt:     time.Now().UTC(),
		ExecutionMode: mode,
		DockerImage:   spec.DockerImage,
		Environment:   env,
	}
	if hasShell {
		task.Commands = normalizedCmds
	} else {
		task.Command = spec.Command
	}
	if spec.ExecutionTimeoutSeconds != nil {
		v := *spec.ExecutionTimeoutSeconds
		task.ExecutionTimeoutSeconds = &v
	}

	return st.CreateTask(ctx, task)
}

func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
