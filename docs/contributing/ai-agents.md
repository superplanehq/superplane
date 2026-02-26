# Using AI Agents for Development

SuperPlane development is AI native. The repository is designed to work seamlessly with AI coding assistants like
Cursor, Codex, or Claude.

This isn't just about using AI to write code faster. It's about building differently. When you pair a great AI
assistant with a codebase that's designed for it, something interesting happens: you can build features that would
have taken days in hours, and ideas that seemed too ambitious suddenly become feasible. The best developers we know
aren't just writing code, they're orchestrating AI agents to build entire systems. That's what we're optimizing
for here. Don't just use AI to do the same work faster. Use it to do work you couldn't do before.

## AGENTS.md Files

The repository includes `AGENTS.md` files that contain guidelines, coding standards, and project structure
information for AI agents:

- **[AGENTS.md](../../AGENTS.md)** - Repository guidelines for backend/GoLang work
- **[web_src/AGENTS.md](../../web_src/AGENTS.md)** - Frontend-specific guidelines for TypeScript/React work

These files are automatically loaded in Cursor as workspace rules. For other tools, you can provide them as
context to your AI agent.

We encourage you to use these files with your AI agents and contribute improvements to them. The goal is to
encode good taste into these files, the patterns, principles, and preferences that make code and product feel right.

## Where AI Canvas Behavior Is Defined

When you need to document or update AI behavior for the canvas builder, these are the source-of-truth locations:

- **Runtime component skill loading (AI chat grounding):**
  - `pkg/grpc/actions/canvases/send_ai_message.go`
  - `templates/skills/*.md`
  - Key functions: `buildCanvasSkillPromptContext`, `loadComponentSkillContent`, `candidateSkillPaths`
- **Canvas auto-layout API contract (backend + CLI + UI):**
  - `protos/canvases.proto` (`CanvasAutoLayout` and `UpdateCanvasRequest.auto_layout`)
  - Generated types:
    - `pkg/protos/canvases/canvases.pb.go`
    - `pkg/openapi_client/model_canvases_canvas_auto_layout.go`
    - `web_src/src/api-client/types.gen.ts`
- **Backend auto-layout behavior:**
  - `pkg/grpc/actions/canvases/auto_layout.go`
  - `pkg/grpc/actions/canvases/update_canvas.go`
  - Includes scope resolution (`FULL_CANVAS`, `CONNECTED_COMPONENT`, `EXACT_SET`) and anchor-relative positioning (selected subgraph keeps its top-left anchor)
- **CLI surface used by AI agents and scripts:**
  - `pkg/cli/commands/canvases/root.go` (flags)
  - `pkg/cli/commands/canvases/update.go` (parsing + request body)
  - Main flags: `--auto-layout`, `--auto-layout-scope`, `--auto-layout-node`

If behavior is component-specific, update `templates/skills/` first. If behavior is operation-level (like canvas layout),
update the protobuf/API contract and the CLI/backend implementation docs together.

### CLI Examples

```bash
# Layout full canvas
superplane canvases update <canvas-id> --auto-layout horizontal

# Layout only the connected subgraph around one seed node
superplane canvases update <canvas-id> \
  --auto-layout horizontal \
  --auto-layout-scope connected-component \
  --auto-layout-node <node-id>

# Layout only an explicit set of nodes
superplane canvases update <canvas-id> \
  --auto-layout horizontal \
  --auto-layout-scope exact-set \
  --auto-layout-node <node-a> \
  --auto-layout-node <node-b>
```

## Component State Coverage

When developing a new component, ensure every statemap includes an explicit error state and a clear state resolution
path. This applies to both implementation and Storybook coverage.

## Rule of Thumb

If you wrote something manually, think about how we could extend and improve the project to automate this
knowledge for the next time.
