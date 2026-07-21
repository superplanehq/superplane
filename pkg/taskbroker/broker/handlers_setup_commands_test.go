package broker

import (
	"testing"

	"github.com/superplanehq/superplane/pkg/taskbroker/shared/api"
	"github.com/superplanehq/superplane/pkg/taskbroker/shared/models"
)

func TestValidateCreateTaskPayload_setupCommands(t *testing.T) {
	t.Run("javascript_script allows setup_commands", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeJavaScript),
			Script:        "function main() { return { ok: true }; }",
			SetupCommands: []string{"npm ci"},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("python_script allows setup_commands", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModePython),
			Script:        "def main(payload):\n    return {'ok': True}",
			SetupCommands: []string{"pip install requests"},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("bash_script allows setup_commands", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeBash),
			Script:        "printf '{\"ok\":true}\\n' > \"$SUPERPLANE_RESULT_FILE\"",
			SetupCommands: []string{"npm ci"},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got != "" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("command_list rejects setup_commands", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeCommandList),
			Commands:      models.CommandList{{Command: "echo hi"}},
			SetupCommands: []string{"npm ci"},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})

	t.Run("argv rejects setup_commands", func(t *testing.T) {
		req := &api.CreateTaskRequest{
			RunMode:       string(models.RunModeArgv),
			Command:       []string{"echo", "hi"},
			SetupCommands: []string{"npm ci"},
			WebhookURL:    "https://x/h",
			ExecutionMode: "host",
		}
		if got := validateCreateTaskPayload(req); got == "" {
			t.Fatal("expected error")
		}
	})
}
