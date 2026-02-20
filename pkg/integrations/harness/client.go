package harness

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
)

const (
	DefaultBaseURL         = "https://app.harness.io/gateway"
	ResourceTypeOrg        = "org"
	ResourceTypeProject    = "project"
	ResourceTypePipeline   = "pipeline"
	DefaultExecutionsLimit = 50
)

type Client struct {
	APIToken                 string
	AccountID                string
	OrgID                    string
	ProjectID                string
	BaseURL                  string
	disableCurrentUserLookup bool
	http                     core.HTTPContext
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with %d: %s", e.StatusCode, e.Body)
}

type RunPipelineRequest struct {
	PipelineIdentifier string
	Ref                string
	InputSetRefs       []string
	InputYAML          string
}

type RunPipelineResponse struct {
	ExecutionID string
}

type ExecutionSummary struct {
	ExecutionID        string
	PipelineIdentifier string
	Status             string
	PlanExecutionURL   string
	StartedAt          string
	EndedAt            string
}

type Pipeline struct {
	Identifier string
	Name       string
}

type Organization struct {
	Identifier string
	Name       string
}

type Project struct {
	Identifier string
	Name       string
}

type UpsertPipelineNotificationRuleRequest struct {
	PipelineIdentifier string
	RuleIdentifier     string
	RuleName           string
	EventTypes         []string
	WebhookURL         string
	Headers            map[string]string
}
