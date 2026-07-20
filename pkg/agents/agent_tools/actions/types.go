package actions

// Input is the top-level JSON payload accepted by the superplane_app tool.
type Input struct {
	Action              string            `json:"action"`
	CanvasID            string            `json:"canvas_id,omitempty"`
	VersionID           string            `json:"version_id,omitempty"`
	DraftVersionID      string            `json:"draft_version_id,omitempty"`
	IncludeConsole      bool              `json:"include_console,omitempty"`
	IncludeIntegrations bool              `json:"include_integrations,omitempty"`
	IncludeCanvasYAML   bool              `json:"include_canvas_yaml,omitempty"`
	ConsoleYAML         string            `json:"console_yaml,omitempty"`
	PatchOperations     []PatchOperation  `json:"patch_operations,omitempty"`
	AutoLayout          *AutoLayoutInput  `json:"auto_layout,omitempty"`
	IntegrationID       string            `json:"integration_id,omitempty"`
	ResourceType        string            `json:"resource_type,omitempty"`
	Parameters          map[string]string `json:"parameters,omitempty"`
	Resource            string            `json:"resource,omitempty"`
	Namespace           string            `json:"namespace,omitempty"`
	NodeID              string            `json:"node_id,omitempty"`
	EventID             string            `json:"event_id,omitempty"`
	ExecutionID         string            `json:"execution_id,omitempty"`
	RunID               string            `json:"run_id,omitempty"`
	Limit               uint32            `json:"limit,omitempty"`
	Before              string            `json:"before,omitempty"`
	States              []string          `json:"states,omitempty"`
	Results             []string          `json:"results,omitempty"`
	Path                string            `json:"path,omitempty"`
	Paths               []string          `json:"paths,omitempty"`
	Content             string            `json:"content,omitempty"`
	Query               string            `json:"query,omitempty"`
}

// PatchOperation describes one small graph edit for patch_staging.
type PatchOperation struct {
	Op       string         `json:"op"`
	NodeID   string         `json:"node_id,omitempty"`
	Node     *PatchNode     `json:"node,omitempty"`
	Position *PatchPosition `json:"position,omitempty"`
	Edge     *PatchEdge     `json:"edge,omitempty"`
}

type PatchNode struct {
	ID            string         `json:"id,omitempty"`
	Name          string         `json:"name,omitempty"`
	Component     string         `json:"component,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
	IntegrationID string         `json:"integration_id,omitempty"`
	Position      *PatchPosition `json:"position,omitempty"`
	IsCollapsed   *bool          `json:"is_collapsed,omitempty"`
}

type PatchEdge struct {
	SourceID string `json:"source_id,omitempty"`
	TargetID string `json:"target_id,omitempty"`
	Channel  string `json:"channel,omitempty"`
}

type PatchPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// AutoLayoutInput configures optional backend auto-layout for draft updates.
type AutoLayoutInput struct {
	Enabled *bool    `json:"enabled,omitempty"`
	Scope   string   `json:"scope,omitempty"`
	NodeIDs []string `json:"node_ids,omitempty"`
}

type readResult struct {
	Action            string              `json:"action"`
	CanvasID          string              `json:"canvas_id"`
	Source            string              `json:"source"`
	VersionID         string              `json:"version_id,omitempty"`
	Draft             *draftResult        `json:"draft,omitempty"`
	Summary           summary             `json:"summary"`
	CanvasYAML        string              `json:"canvas_yaml,omitempty"`
	CanvasYAMLBytes   int                 `json:"canvas_yaml_bytes,omitempty"`
	CanvasYAMLOmitted bool                `json:"canvas_yaml_omitted,omitempty"`
	ConsoleYAML       string              `json:"console_yaml,omitempty"`
	Integrations      []integrationResult `json:"integrations,omitempty"`
}

type updateResult struct {
	Action     string      `json:"action"`
	CanvasID   string      `json:"canvas_id"`
	VersionID  string      `json:"version_id"`
	Draft      draftResult `json:"draft"`
	Summary    summary     `json:"summary"`
	NodeIssues []nodeIssue `json:"node_issues,omitempty"`
}

type integrationsResult struct {
	Action       string              `json:"action"`
	CanvasID     string              `json:"canvas_id"`
	Integrations []integrationResult `json:"integrations"`
}

type resourcesResult struct {
	Action        string                      `json:"action"`
	CanvasID      string                      `json:"canvas_id"`
	IntegrationID string                      `json:"integration_id"`
	ResourceType  string                      `json:"resource_type"`
	Count         int                         `json:"count"`
	Truncated     bool                        `json:"truncated,omitempty"`
	Resources     []integrationResourceResult `json:"resources"`
}

type integrationResourceResult struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type runtimeReadResult struct {
	Action   string `json:"action"`
	CanvasID string `json:"canvas_id"`
	Resource string `json:"resource"`
	Payload  any    `json:"payload"`
}

type fileListResult struct {
	Action       string   `json:"action"`
	CanvasID     string   `json:"canvas_id"`
	Files        []string `json:"files"`
	ContextFiles []string `json:"context_files,omitempty"`
}

type fileReadResult struct {
	Action   string          `json:"action"`
	CanvasID string          `json:"canvas_id"`
	Files    []fileReadEntry `json:"files"`
	Errors   []fileReadError `json:"errors,omitempty"`
}

type fileReadEntry struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Source    string `json:"source"`
	VersionID string `json:"version_id,omitempty"`
}

type fileReadError struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

type fileStageResult struct {
	Action         string         `json:"action"`
	CanvasID       string         `json:"canvas_id"`
	VersionID      string         `json:"version_id"`
	Path           string         `json:"path"`
	Deleted        bool           `json:"deleted,omitempty"`
	StagingSummary stagingSummary `json:"staging_summary"`
}

type stagingSummary struct {
	HasStaging  bool     `json:"has_staging"`
	StagedPaths []string `json:"staged_paths,omitempty"`
}

type accessResult struct {
	Action         string             `json:"action"`
	CanvasID       string             `json:"canvas_id"`
	OrganizationID string             `json:"organization_id"`
	UserID         string             `json:"user_id"`
	TokenScopes    []string           `json:"token_scopes"`
	ToolActions    []toolAccessResult `json:"tool_actions"`
	Accessible     []apiAccessResult  `json:"accessible"`
	Unavailable    []apiAccessResult  `json:"unavailable,omitempty"`
	Notes          []string           `json:"notes,omitempty"`
}

type toolAccessResult struct {
	Action   string   `json:"action"`
	Allowed  bool     `json:"allowed"`
	Requires []string `json:"requires,omitempty"`
	Reason   string   `json:"reason,omitempty"`
}

type apiAccessResult struct {
	Method    string   `json:"method"`
	Service   string   `json:"service,omitempty"`
	RPC       string   `json:"rpc,omitempty"`
	Resource  string   `json:"resource"`
	Operation string   `json:"operation"`
	Resources []string `json:"resources,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

type draftResult struct {
	VersionID   string `json:"version_id"`
	DisplayName string `json:"display_name,omitempty"`
	BranchName  string `json:"branch_name,omitempty"`
}

type summary struct {
	CanvasName string        `json:"canvas_name,omitempty"`
	NodeCount  int           `json:"node_count"`
	EdgeCount  int           `json:"edge_count"`
	Nodes      []nodeSummary `json:"nodes,omitempty"`
}

type nodeSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	Component string `json:"component,omitempty"`
	Issue     string `json:"issue,omitempty"`
}

type nodeIssue struct {
	NodeID   string `json:"node_id"`
	NodeName string `json:"node_name,omitempty"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type integrationResult struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Vendor string `json:"vendor"`
	State  string `json:"state"`
}
