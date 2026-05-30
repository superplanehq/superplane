package railway

import "encoding/json"

type GraphQLRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

type GraphQLResponse struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations,omitempty"`
	Path       []any                  `json:"path,omitempty"`
	Extensions map[string]any         `json:"extensions,omitempty"`
}

type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type Workspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	WorkspaceID  string              `json:"workspaceId,omitempty"`
	Environments ProjectEnvironments `json:"environments,omitempty"`
	Services     ProjectServices     `json:"services,omitempty"`
}

type ProjectEnvironments struct {
	Edges []EnvironmentEdge `json:"edges"`
}

type EnvironmentEdge struct {
	Node Environment `json:"node"`
}

type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectServices struct {
	Edges []ServiceEdge `json:"edges"`
}

type ServiceEdge struct {
	Node Service `json:"node"`
}

type Service struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Deployment struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	StatusUpdatedAt   string `json:"statusUpdatedAt,omitempty"`
	ProjectID         string `json:"projectId,omitempty"`
	ServiceID         string `json:"serviceId,omitempty"`
	EnvironmentID     string `json:"environmentId,omitempty"`
	SnapshotID        string `json:"snapshotId,omitempty"`
	StaticURL         string `json:"staticUrl,omitempty"`
	URL               string `json:"url,omitempty"`
	CanRollback       bool   `json:"canRollback"`
	CanRedeploy       bool   `json:"canRedeploy"`
	DeploymentStopped bool   `json:"deploymentStopped"`
	Meta              any    `json:"meta,omitempty"`
	Diagnosis         any    `json:"diagnosis,omitempty"`
	Creator           any    `json:"creator,omitempty"`
}

type NotificationRule struct {
	ID                    string                `json:"id"`
	ProjectID             string                `json:"projectId,omitempty"`
	EventTypes            []string              `json:"eventTypes"`
	Severities            []string              `json:"severities"`
	EphemeralEnvironments bool                  `json:"ephemeralEnvironments"`
	CreatedAt             string                `json:"createdAt"`
	UpdatedAt             string                `json:"updatedAt"`
	Channels              []NotificationChannel `json:"channels"`
}

type NotificationChannel struct {
	ID        string                    `json:"id"`
	Config    NotificationChannelConfig `json:"config"`
	CreatedAt string                    `json:"createdAt"`
	UpdatedAt string                    `json:"updatedAt"`
}

type NotificationChannelConfig struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}
