package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/superplanehq/superplane/pkg/runners/api"
	"github.com/superplanehq/superplane/pkg/runners/diagnostics"
)

const diagnosticCaptureBytes = 16 * 1024

// failureDiagnostics observes the existing live command records. Retaining the
// first failed step prevents later artifact or cleanup output from hiding it.
// PTY execution combines stdout and stderr, so its excerpt contains both.
type failureDiagnostics struct {
	mu            sync.Mutex
	live          io.Writer
	pending       []byte
	oversizedLine bool
	step          liveLogCommandStartRecord
	output        string
	failed        *failedStep
}
type failedStep struct {
	step   liveLogCommandStartRecord
	output string
}

func (d *failureDiagnostics) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, b := range p {
		if b == '\n' {
			if d.oversizedLine {
				d.appendOutput("[Output line omitted: exceeds diagnostic limit]\n")
			} else {
				d.line(d.pending)
			}
			d.pending = nil
			d.oversizedLine = false
			continue
		}
		if len(d.pending) < diagnosticCaptureBytes && !d.oversizedLine {
			d.pending = append(d.pending, b)
			continue
		}
		d.pending = nil
		d.oversizedLine = true
	}
	if d.live != nil {
		return d.live.Write(p)
	}
	return len(p), nil
}
func (d *failureDiagnostics) line(line []byte) {
	var record struct {
		Type   string `json:"type"`
		Status string `json:"status"`
		Index  int    `json:"index"`
	}
	if json.Unmarshal(bytes.TrimSpace(line), &record) == nil {
		if record.Type == "cmd_start" {
			var start liveLogCommandStartRecord
			if json.Unmarshal(line, &start) == nil {
				d.step = start
				d.output = ""
			}
			return
		}
		if record.Type == "cmd_end" {
			if record.Status == "failed" && d.failed == nil && record.Index == d.step.Index {
				d.failed = &failedStep{d.step, d.output}
			}
			return
		}
	}
	d.appendOutput(string(line) + "\n")
}
func (d *failureDiagnostics) appendOutput(line string) {
	d.output += line
	// Discard complete lines so a cut cannot expose part of a redacted value.
	for len(d.output) > diagnosticCaptureBytes {
		_, tail, found := strings.Cut(d.output, "\n")
		if !found {
			d.output = ""
			return
		}
		d.output = tail
	}
}

var sensitiveEnvironmentName = regexp.MustCompile(`(?i)api[_-]?key|token|secret|password|credential`)

func diagnosticEnvironmentValues(task *api.TaskPayload) []string {
	var values []string
	for _, pair := range os.Environ() {
		name, value, _ := strings.Cut(pair, "=")
		if sensitiveEnvironmentName.MatchString(name) {
			values = append(values, strings.Split(value, "\n")...)
		}
	}
	for _, variable := range task.Environment {
		values = append(values, strings.Split(variable.Value, "\n")...)
	}
	return values
}

func (d *failureDiagnostics) failure(task *api.TaskPayload, code int, runErr error, result json.RawMessage) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.oversizedLine {
		d.line(d.pending)
	}
	step, output := d.step, d.output
	if d.failed != nil {
		step, output = d.failed.step, d.failed.output
	}
	values := diagnosticEnvironmentValues(task)
	cause := "Process failed without an error message."
	if runErr != nil {
		cause = runErr.Error()
	}
	// Only a prompt result is a provider error; ordinary script results can use
	// these same JSON keys for unrelated application data.
	if step.Kind == "prompt" {
		var provider struct {
			IsError bool     `json:"is_error"`
			Error   string   `json:"error"`
			Result  string   `json:"result"`
			Errors  []string `json:"errors"`
		}
		if json.Unmarshal(result, &provider) == nil && provider.IsError {
			providerMessage := strings.TrimSpace(provider.Error)
			if providerMessage == "" {
				providerMessage = strings.TrimSpace(provider.Result)
			}
			if providerMessage == "" {
				providerMessage = strings.Join(provider.Errors, "\n")
			}
			if providerMessage != "" {
				cause = providerMessage
			}
		}
	}
	name := diagnostics.Sanitize(step.Text, values, 256)
	if name == "" {
		name = "Step unavailable"
	}
	excerpt := diagnostics.Sanitize(output, values, 0)
	if len(excerpt) > 2048 {
		excerpt = excerpt[len(excerpt)-2048:]
		for !utf8.ValidString(excerpt) {
			excerpt = excerpt[1:]
		}
		excerpt = "[truncated]\n" + excerpt
	}
	if excerpt == "" {
		excerpt = "No diagnostic output was captured."
	}
	identity := name
	if step.Index >= 0 {
		identity = fmt.Sprintf("%s (step %d)", name, step.Index+1)
	}
	message := fmt.Sprintf("%s (exit code %d): %s\nOutput excerpt:\n%s", identity, code, diagnostics.Sanitize(cause, values, 768), excerpt)
	return fmt.Errorf("%s", diagnostics.WithLogReference(message, task.ID))
}
