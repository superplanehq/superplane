package semaphore

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetJobLogs struct{}

type GetJobLogsSpec struct {
	JobID string `json:"jobId"`
	Limit int    `json:"limit,omitempty"`
}

type GetJobLogsMetadata struct {
	JobID  string `json:"jobId"`
	Limit  int    `json:"limit,omitempty"`
	Status string `json:"status,omitempty"`
}

const (
	GetJobLogsOutputChannel = "logs"
	PayloadTypeJobLogs      = "semaphore.job.logs"
	DefaultLogLimit         = 1000
	MaxLogLimit             = 10000
)

func (g *GetJobLogs) Name() string {
	return "semaphore.getJobLogs"
}

func (g *GetJobLogs) Label() string {
	return "Get Job Logs"
}

func (g *GetJobLogs) Description() string {
	return "Fetches the log output for a Semaphore job by job ID"
}

func (g *GetJobLogs) Documentation() string {
	return `The Get Job Logs component fetches log output from a Semaphore CI/CD job.

## Use Cases

- **Debugging failures**: When a pipeline fails, fetch logs to analyze the error
- **Notifications**: Attach job logs to Slack/PagerDuty notifications
- **Log analysis**: Parse job logs to extract test summaries, error lines, or metrics
- **Audit trails**: Keep records of job outputs for compliance

## How It Works

1. Accepts a Semaphore Job ID (from pipeline/block event data, or from Run Workflow output)
2. Calls the Semaphore API to fetch job logs
3. Optionally limits the number of log lines returned (default: 1000, max: 10000)
4. Emits the log content as output

## Configuration

- **Job ID** (required): The Semaphore job ID. Can be an expression that resolves to a job ID from:
  - \`data.blocks[].jobs[].id\` from On Pipeline Done events
  - Run Workflow component output
  - Previous Get Pipeline or similar actions

- **Limit** (optional): Maximum number of log lines to return. 
  - Default: 1000 lines
  - Maximum: 10000 lines
  - If job has fewer lines, returns all available

## Output Channel

- **Logs**: Emits job log content as raw text
  - Payload includes: log content, job metadata (name, result, status)
  - If job has no logs, emits empty string

## Notes

- Job must exist and be accessible with the configured API token
- Log content is returned as plain text
- For large logs, use the limit parameter to truncate
- Combine with text parsing nodes to extract specific information`
}

func (g *GetJobLogs) Icon() string {
	return "file-text"
}

func (g *GetJobLogs) Color() string {
	return "blue"
}

func (g *GetJobLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GetJobLogsOutputChannel,
			Label: "Logs",
		},
	}
}

func (g *GetJobLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobId",
			Label:       "Job ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "Semaphore job ID (e.g., from pipeline data: data.blocks[].jobs[].id)",
			Placeholder: "e.g., ${data.blocks[0].jobs[0].id}",
		},
		{
			Name:        "limit",
			Label:       "Limit",
			Type:        configuration.FieldTypeNumber,
			Description: "Maximum number of log lines to return (default: 1000, max: 10000)",
			Default:     DefaultLogLimit,
		},
	}
}

func (g *GetJobLogs) Setup(ctx core.SetupContext) error {
	return nil
}

func (g *GetJobLogs) Execute(ctx core.ExecutionContext) error {
	spec := GetJobLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.JobID == "" {
		return fmt.Errorf("jobId is required")
	}

	// Validate and apply limits
	limit := spec.Limit
	if limit <= 0 {
		limit = DefaultLogLimit
	}
	if limit > MaxLogLimit {
		limit = MaxLogLimit
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	ctx.Logger.Infof("Fetching logs for job %s (limit: %d)", spec.JobID, limit)

	// Fetch job details first to get metadata
	job, err := client.GetJob(spec.JobID)
	if err != nil {
		return fmt.Errorf("error fetching job %s: %v", spec.JobID, err)
	}

	// Fetch job logs
	logs, err := client.GetJobLogs(spec.JobID, limit)
	if err != nil {
		return fmt.Errorf("error fetching logs for job %s: %v", spec.JobID, err)
	}

	// Store metadata
	metadata := GetJobLogsMetadata{
		JobID:  spec.JobID,
		Limit:  limit,
		Status: job.Status,
	}
	ctx.Metadata.Set(metadata)

	// Prepare payload
	payload := map[string]any{
		"job": map[string]any{
			"id":     job.ID,
			"name":   job.Name,
			"status": job.Status,
			"result": job.Result,
		},
		"logs": logs,
		"metadata": map[string]any{
			"linesReturned": len(logs),
			"limit":         limit,
		},
	}

	// Emit logs output
	return ctx.ExecutionState.Emit(GetJobLogsOutputChannel, PayloadTypeJobLogs, []any{payload})
}

func (g *GetJobLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}

// Job represents a Semaphore job
type Job struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Result string `json:"result"`
}

// JobResponse represents the API response for job details
type JobResponse struct {
	Job *Job `json:"job"`
}

// GetJob fetches job details from Semaphore API
func (c *Client) GetJob(jobID string) (*Job, error) {
	URL := fmt.Sprintf("%s/api/v1alpha/jobs/%s", c.OrgURL, jobID)
	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	var jobResponse JobResponse
	err = json.Unmarshal(responseBody, &jobResponse)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling job response: %v", err)
	}

	return jobResponse.Job, nil
}

// JobLogsResponse represents the API response for job logs
type JobLogsResponse struct {
	Logs []LogLine `json:"logs"`
}

// LogLine represents a single log line
type LogLine struct {
	Number  int    `json:"number"`
	Content string `json:"content"`
}

// GetJobLogs fetches job logs from Semaphore API
func (c *Client) GetJobLogs(jobID string, limit int) (string, error) {
	// Semaphore API uses offset-based pagination for logs
	// We'll fetch all logs and truncate if needed
	URL := fmt.Sprintf("%s/api/v1alpha/jobs/%s/logs", c.OrgURL, jobID)
	
	// Add limit parameter if specified
	if limit > 0 && limit < MaxLogLimit {
		URL = fmt.Sprintf("%s?limit=%s", URL, strconv.Itoa(limit))
	}

	responseBody, err := c.execRequest(http.MethodGet, URL, nil)
	if err != nil {
		// Handle 404 - job exists but no logs yet
		if httpErr, ok := err.(interface{ StatusCode() int }); ok && httpErr.StatusCode() == http.StatusNotFound {
			return "", nil // No logs available yet
		}
		return "", err
	}

	var logsResponse JobLogsResponse
	err = json.Unmarshal(responseBody, &logsResponse)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling logs response: %v", err)
	}

	// Concatenate log lines
	var logs string
	for _, line := range logsResponse.Logs {
		logs += line.Content + "\n"
	}

	return logs, nil
}
