package api

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/superplanehq/superplane/pkg/runners/models"
)

const (
	DefaultExecutionTimeoutSeconds       = 3600
	DefaultLogChunkBytes           int64 = 64 * 1024
	DefaultLogFlushMinimumMS       int64 = 2500
	DefaultLogFlushMaximumMS       int64 = 5000
	LogUploadMaxChunkMultiplier    int64 = 2

	LogUploadActionContinue = "continue"
	LogUploadActionStop     = "stop"

	HeaderLogUploadAction     = "X-SuperPlane-Log-Action"
	HeaderLogTargetChunkBytes = "X-SuperPlane-Log-Target-Chunk-Bytes"
	HeaderLogFlushMinimumMS   = "X-SuperPlane-Log-Flush-Minimum-Ms"
	HeaderLogFlushMaximumMS   = "X-SuperPlane-Log-Flush-Maximum-Ms"
)

var environmentNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type EnvironmentVariable = models.EnvironmentVariable

type LogUploadPolicy struct {
	TargetChunkBytes      int64 `json:"target_chunk_bytes"`
	PartialFlushMinimumMS int64 `json:"partial_flush_interval_min_ms"`
	PartialFlushMaximumMS int64 `json:"partial_flush_interval_max_ms"`
}

func DefaultLogUploadPolicy() LogUploadPolicy {
	return LogUploadPolicy{
		TargetChunkBytes:      DefaultLogChunkBytes,
		PartialFlushMinimumMS: DefaultLogFlushMinimumMS,
		PartialFlushMaximumMS: DefaultLogFlushMaximumMS,
	}
}

func (p LogUploadPolicy) MaxUploadBytes() int64 {
	normalized := NormalizeLogUploadPolicy(&p)
	return normalized.TargetChunkBytes * LogUploadMaxChunkMultiplier
}

func NormalizeLogUploadPolicy(policy *LogUploadPolicy) LogUploadPolicy {
	if policy == nil {
		return DefaultLogUploadPolicy()
	}
	normalized := *policy
	if normalized.TargetChunkBytes <= 0 ||
		normalized.TargetChunkBytes > (1<<63-1)/LogUploadMaxChunkMultiplier {
		normalized.TargetChunkBytes = DefaultLogChunkBytes
	}
	if normalized.PartialFlushMinimumMS <= 0 {
		normalized.PartialFlushMinimumMS = DefaultLogFlushMinimumMS
	}
	if normalized.PartialFlushMaximumMS < normalized.PartialFlushMinimumMS {
		normalized.PartialFlushMaximumMS = normalized.PartialFlushMinimumMS
	}
	return normalized
}

// TaskPayload is the encrypted task specification delivered through the v1
// runner WebSocket.
type TaskPayload struct {
	ID                      string                `json:"id"`
	RunMode                 string                `json:"run_mode,omitempty"`
	Script                  string                `json:"script,omitempty"`
	MessageChain            json.RawMessage       `json:"message_chain,omitempty"`
	Command                 []string              `json:"command,omitempty"`
	Commands                models.CommandList    `json:"commands,omitempty"`
	SetupCommands           []string              `json:"setup_commands,omitempty"`
	Environment             []EnvironmentVariable `json:"environment,omitempty"`
	Files                   []TaskFile            `json:"files,omitempty"`
	ExecutionMode           string                `json:"execution_mode"`
	DockerImage             string                `json:"docker_image,omitempty"`
	ExecutionTimeoutSeconds *int                  `json:"execution_timeout_seconds,omitempty"`
	WebhookPayloadSizeLimit int                   `json:"webhook_payload_size_limit,omitempty"`
	LogUploadPolicy         *LogUploadPolicy      `json:"log_upload_policy,omitempty"`
}

func RunModeForTask(task *TaskPayload) models.RunMode {
	if task == nil {
		return ""
	}
	if mode := models.RunMode(
		strings.ToLower(strings.TrimSpace(task.RunMode)),
	); mode != "" {
		return mode
	}
	return models.InferRunMode(
		task.Commands,
		task.Command,
		strings.TrimSpace(task.Script),
	)
}

func NormalizeCommandLines(commands []string) []string {
	var normalized []string
	for _, command := range commands {
		if command = strings.TrimSpace(command); command != "" {
			normalized = append(normalized, command)
		}
	}
	return normalized
}

func ValidateEnvironment(environment []EnvironmentVariable) string {
	seen := make(map[string]struct{}, len(environment))
	for _, variable := range environment {
		if !environmentNamePattern.MatchString(variable.Name) {
			return "invalid environment variable name"
		}
		if _, ok := seen[variable.Name]; ok {
			return "duplicate environment variable name"
		}
		seen[variable.Name] = struct{}{}
		if strings.ContainsRune(variable.Value, '\x00') {
			return "environment variable values cannot contain NUL bytes"
		}
	}
	return ""
}
