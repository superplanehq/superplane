package terraform

import (
	"time"
)

type WebhookConfiguration struct {
	WorkspaceID string   `json:"workspaceId"`
	Events      []string `json:"events"`
}

type PayloadVersion1 struct {
	PayloadVersion              int            `json:"payload_version"`
	NotificationConfigurationID string         `json:"notification_configuration_id"`
	RunURL                      string         `json:"run_url"`
	RunID                       string         `json:"run_id"`
	RunMessage                  string         `json:"run_message"`
	RunCreatedAt                *time.Time     `json:"run_created_at"`
	RunCreatedBy                string         `json:"run_created_by"`
	WorkspaceID                 string         `json:"workspace_id"`
	WorkspaceName               string         `json:"workspace_name"`
	OrganizationName            string         `json:"organization_name"`
	Notifications               []Notification `json:"notifications"`
}

type PayloadVersion2 struct {
	PayloadVersion              int            `json:"payload_version"`
	NotificationConfigurationID string         `json:"notification_configuration_id"`
	WorkspaceID                 string         `json:"workspace_id"`
	WorkspaceName               string         `json:"workspace_name"`
	OrganizationName            string         `json:"organization_name"`
	Notifications               []Notification `json:"notifications"`
	TriggerScope                string         `json:"trigger_scope"`
}

type Notification struct {
	Message      string     `json:"message"`
	Trigger      string     `json:"trigger"`
	RunStatus    string     `json:"run_status"`
	RunUpdatedAt *time.Time `json:"run_updated_at"`
	RunUpdatedBy string     `json:"run_updated_by"`
}

type RunEventData struct {
	RunID            string `json:"runId"`
	RunURL           string `json:"runUrl"`
	RunMessage       string `json:"runMessage"`
	RunStatus        string `json:"runStatus"`
	WorkspaceID      string `json:"workspaceId"`
	WorkspaceName    string `json:"workspaceName"`
	OrganizationName string `json:"organizationName"`
	Action           string `json:"action"`
	RunCreatedBy     string `json:"runCreatedBy"`
}
