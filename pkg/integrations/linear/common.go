package linear

// Team represents a Linear team.
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// Label represents a Linear issue label.
type Label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Issue represents a Linear issue (minimal for create response and webhook payload).
type Issue struct {
	ID          string  `json:"id"`
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	URL         string  `json:"url,omitempty"`
	CreatedAt   string  `json:"createdAt,omitempty"`
	Team        *IDRef  `json:"team,omitempty"`
	State       *IDRef  `json:"state,omitempty"`
	Assignee    *IDRef  `json:"assignee,omitempty"`
}

// IDRef is a minimal reference with just an ID, used for nested GraphQL objects.
type IDRef struct {
	ID string `json:"id"`
}

// pageInfo holds Relay-style cursor pagination info.
type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// WorkflowState represents a Linear workflow state (e.g., Todo, In Progress, Done).
type WorkflowState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // backlog, unstarted, started, completed, cancelled
}

// Member represents a Linear team member.
type Member struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName,omitempty"`
	Email       string `json:"email,omitempty"`
}

// NodeMetadata stores metadata on trigger/component nodes.
type NodeMetadata struct {
	Team *Team `json:"team,omitempty"`
}
