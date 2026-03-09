# Daytona Execute Command Skill

Use this guidance when planning or configuring `daytona.executeCommand`.

## Purpose

`daytona.executeCommand` runs shell commands in an existing Daytona sandbox.

## Required Configuration

- `sandbox` (required): sandbox identifier from an upstream sandbox node.
- `command` (required): shell command to run.
- `cwd` (optional): working directory.
- `timeout` (optional): execution timeout in seconds.

## Planning Rules

When generating workflow operations that include `daytona.executeCommand`:

1. Always set `configuration.sandbox` from the sandbox-producing node:
   - `daytona.createSandbox` output: `{{ $["Create Sandbox"].data.id }}`
   - `daytona.createRepositorySandbox` output: `{{ $["Clone and Run App"].data.sandboxId }}`
2. Always set `configuration.command` to a concrete command string.
3. Only set `cwd` and `timeout` when the user asks for them or they are clearly needed.
4. If this node is used to "setup and run dev server", ensure the command starts the server in a way that allows the step to finish (for example background process), then follow with `daytona.getPreviewUrl`.
5. Use a deterministic preview port when the next step generates a preview URL.
6. Never leave `configuration.sandbox` empty.
7. Do not use `sandboxId` as a configuration key; the correct key is `sandbox`.
8. If there is no upstream sandbox-producing node, add one first (typically `daytona.createSandbox`) and wire `configuration.sandbox` from it.
9. `daytona.executeCommand` emits on `success` or `failed` channels (not `default`).
10. When connecting downstream nodes from `daytona.executeCommand`, set `source.handleId` explicitly:
    - `success` for the happy path (for example to `daytona.getPreviewUrl`)
    - `failed` for failure handling/recovery paths
11. Never emit `connect_nodes` from this component with `source.handleId: "default"` because this block does not expose a `default` output.

## GitHub PR-Comment Trigger Notes

When the upstream trigger is `github.onPRComment`:

- Use these trigger fields in command expressions:
  - PR number: `root().data.issue.number`
  - repo owner: `root().data.repository.owner.login`
  - repo name: `root().data.repository.name`
- Use PR checkout by PR number (for example `git fetch origin pull/<number>/head && git checkout FETCH_HEAD`).

## Common Dev-Server Pattern

1. `daytona.createSandbox`
2. `daytona.executeCommand` with `configuration.sandbox` set from step 1 and install + start command (server bound to host/port)
3. `connect_nodes` from step 2 to step 4 with `source.handleId: "success"`
4. `daytona.getPreviewUrl` with matching `sandbox` and `port`

## Configuration Example (Important)

- `sandbox: "{{ $[\"Create Sandbox\"].data.id }}"`
- `command: "npm install && npm run dev -- --host 0.0.0.0 --port 3000"`
- If upstream is `daytona.createRepositorySandbox`, use:
  - `sandbox: "{{ $[\"Clone and Run App\"].data.sandboxId }}"`

If using an existing sandbox node, reference that node's output ID in `sandbox`.

## Mistakes To Avoid

- Missing `sandbox`.
- Missing `command`.
- Running a long-lived foreground dev server command that never returns and blocks the flow.
- Using a preview URL step without ensuring a server is started on the requested port.
- Connecting from `daytona.executeCommand` with `source.handleId: "default"` instead of `success` or `failed`.
- Referencing `{{ $["Clone and Run App"].data.id }}` instead of `{{ $["Clone and Run App"].data.sandboxId }}`.
