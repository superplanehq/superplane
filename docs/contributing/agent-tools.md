# Agent Tools

Agent tools are backend-managed custom tools exposed to managed-agent providers.
They let the agent call SuperPlane backend code directly instead of shelling out
to the CLI for common app operations.

## Choosing Tool vs Action

Prefer adding an action to `superplane_app` when the capability is app-related:

- reading app/canvas or Console YAML (served from the effective staged content the UI editor reads: the user's pending staged edits when present, otherwise the live version)
- listing, reading, staging, deleting, or committing normal app repository files such as `README.md`, `AGENTS.md`, or scripts used by file-backed components
- updating the app (`patch_draft` saves graph, Console, and layout edits as pending staged changes against the live canvas, exactly like edits made in the UI editor, so the user reviews and commits them)
- inspecting agent token permissions (`access`) and runtime state (`read_runtime`)
- listing connected integrations for the current app context
- validating or preparing app-specific backend state
- any operation naturally scoped to the current `AgentSessionContext.CanvasID`

Create a new top-level agent tool only when the capability is not an app action:

- it has a different domain than the current app
- it needs a distinct provider-facing schema or description
- it should be discoverable as a separate capability in Anthropic tool config
- grouping it under `superplane_app.action` would make the input schema broad or
  confusing

Current top-level tools:

- `superplane_app`: app/canvas/Console operations, dispatched through actions
- `superplane_component_schema`: component, trigger, and widget schema lookup

## Adding an App Action

App actions live in `pkg/agents/agent_tools/actions`.

1. Add a file such as `archive.go` or `validate.go`.
2. Implement the `Action` interface:

   ```go
   type myAction struct {
     deps Dependencies
   }

   func newMyAction(deps Dependencies) myAction {
     return myAction{deps: deps}
   }

   func (myAction) Name() string {
     return "my_action"
   }

   func (a myAction) Execute(ctx context.Context, session agents.AgentSessionContext, input Input) (any, error) {
     // Use session.OrganizationID, session.UserID, and session.CanvasID.
     // Return a compact JSON-marshalable result.
   }
   ```

3. Register it in `NewDefaultRegistry` in `actions/registry.go`.
4. Add any action-specific input fields to `actions.Input`.
5. Update `AppAgentTool.InputSchema()` in `app_agent_tool.go` so the provider
   knows when and how to call the action.
6. Add focused tests in `actions/`.

Action rules:

- Always stay scoped to the current `AgentSessionContext`.
- Reject or ignore attempts to operate on another canvas.
- Never commit from an agent action without an explicit user-driven commit step. `patch_draft` saves graph, Console, and layout edits as pending staged changes (exactly like edits made in the UI editor) and never commits or goes live; the user reviews and commits the staged changes with a message.
- Treat `canvas.yaml` and `console.yaml` as spec files. Agents should update them through `patch_draft`; normal repository file actions are for additional app files.
- When exposing repository file reads, preserve the same staging semantics as `read`: serve effective staged content for the current user when present.
- Return concise JSON payloads; avoid dumping large unrelated data.
- Prefer backend APIs and model methods over invoking the CLI.

## App Repository Context Files

The app repository can contain additional files beyond `canvas.yaml` and
`console.yaml`. Examples include `README.md`, scripts referenced by file-backed
components, and AI context files.

Managed agents treat these repository file names as context candidates:

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `*.agents.md`

When an app task involves repository files, the agent should use
`superplane_app` action `list_files`, read any returned context files with
`read_file`, and then stage normal file edits with `write_file` or
`delete_file`. `commit_files` commits staged repository file changes to git; it does not
commit canvas staging or go live.

## Adding a Top-Level Agent Tool

Top-level tools live in `pkg/agents/agent_tools`.

1. Define a typed input struct for the tool.
2. Implement `AgentTool[T]`:

   ```go
   type ExampleAgentTool struct {
     registry *registry.Registry
   }

   type exampleInput struct {
     Query string `json:"query,omitempty"`
   }

   var _ AgentTool[exampleInput] = (*ExampleAgentTool)(nil)

   func NewExampleAgentTool(registry *registry.Registry) *ExampleAgentTool {
     return &ExampleAgentTool{registry: registry}
   }

   func (t *ExampleAgentTool) Name() string {
     return "superplane_example"
   }

   func (t *ExampleAgentTool) Description() string {
     return "Short provider-facing description of when to use this tool."
   }

   func (t *ExampleAgentTool) InputSchema() agents.CustomToolInputSchema {
     return agents.CustomToolInputSchema{
       Type: "object",
       Properties: map[string]agents.CustomToolInputSchema{
         "query": {Type: "string"},
       },
     }
   }

   func (t *ExampleAgentTool) Call(ctx context.Context, session agents.AgentSessionContext, input exampleInput) (Result, error) {
     return Result{Payload: map[string]any{"ok": true}}, nil
   }
   ```

3. Register the tool in `init()`:

   ```go
   func init() {
     Register[exampleInput]("superplane_example", func(deps Dependencies) AgentTool[exampleInput] {
       return NewExampleAgentTool(deps.ComponentRegistry)
     })
   }
   ```

4. Add tests for definition metadata and `Call`.
5. Update `pkg/agents/anthropic/agent_prompt.md` and any preamble guidance if
   the agent should prefer the new tool.

The `agent_tools.Registry` adapts provider custom tool calls to typed tool
inputs. Concrete tools should not parse `agents.CustomToolUse` directly.
