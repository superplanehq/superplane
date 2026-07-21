package broker

import (
	"strings"
	"testing"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestValidateCreateTaskPayload_files(t *testing.T) {
	t.Run("command_list allows files", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeCommandList),
			Commands:      models.CommandList{{Command: `cat "$SUPERPLANE_TASK_DIR/hi.txt"`}},
			Files:         []api.TaskFile{{Path: "hi.txt", Content: "hello"}},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeCommandList),
			Commands:      models.CommandList{{Command: "echo hi"}},
			Files:         []api.TaskFile{{Path: "/etc/passwd", Content: "x"}},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects oversized content", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeArgv),
			Command:       []string{"echo", "hi"},
			Files:         []api.TaskFile{{Path: "big.txt", Content: strings.Repeat("x", api.MaxTaskFileBytes+1)}},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})
}
