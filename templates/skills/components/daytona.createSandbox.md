# Daytona Create Sandbox Skill

Use this guidance when planning or configuring `daytona.createSandbox`.

## Purpose

`daytona.createSandbox` creates an isolated Daytona sandbox used by downstream Daytona steps.

## Required Configuration

- No required fields.
- Optional fields:
  - `snapshot`
  - `target`
  - `autoStopInterval`
  - `env` (list of `{name, value}`)

## Planning Rules

When generating workflow operations that include `daytona.createSandbox`:

1. Add this node before `daytona.executeCommand` or `daytona.getPreviewUrl` if no sandbox producer exists.
2. Reuse an existing sandbox-producing node when present instead of adding duplicates.
3. Only set optional fields when the user requests specific environment, region, timeout, or env vars.
4. Keep defaults when the user has no strict requirement.

## Output Wiring

- This component produces the sandbox ID at `data.id`.
- Downstream Daytona nodes should reference that produced sandbox value.

## Mistakes To Avoid

- Adding multiple sandbox-creation nodes without a user reason.
- Setting unnecessary optional config by default.
- Leaving downstream Daytona nodes without a sandbox reference.
