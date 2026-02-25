# Daytona Execute Command Skill

Use this guidance when planning or configuring `daytona.executeCommand`.

## Purpose

`daytona.executeCommand` runs shell commands in an existing Daytona sandbox.

## Required Configuration

- `sandboxId` (required): sandbox identifier from an upstream sandbox node.
- `command` (required): shell command to run.
- `cwd` (optional): working directory.
- `timeout` (optional): execution timeout in seconds.

## Planning Rules

When generating workflow operations that include `daytona.executeCommand`:

1. Always set `configuration.sandboxId` from the sandbox-producing node (for example `daytona.createSandbox` output).
2. Always set `configuration.command` to a concrete command string.
3. Only set `cwd` and `timeout` when the user asks for them or they are clearly needed.
4. If this node is used to "setup and run dev server", ensure the command starts the server in a way that allows the step to finish (for example background process), then follow with `daytona.getPreviewUrl`.
5. Use a deterministic preview port when the next step generates a preview URL.

## GitHub PR-Comment Trigger Notes

When the upstream trigger is `github.onPRComment`:

- Use these trigger fields in command expressions:
  - PR number: `root().data.issue.number`
  - repo owner: `root().data.repository.owner.login`
  - repo name: `root().data.repository.name`
- Use PR checkout by PR number (for example `git fetch origin pull/<number>/head && git checkout FETCH_HEAD`).

## Common Dev-Server Pattern

1. `daytona.createSandbox`
2. `daytona.executeCommand` with install + start command (server bound to host/port)
3. `daytona.getPreviewUrl` with matching `sandbox` and `port`

## Mistakes To Avoid

- Missing `sandboxId`.
- Missing `command`.
- Running a long-lived foreground dev server command that never returns and blocks the flow.
- Using a preview URL step without ensuring a server is started on the requested port.
