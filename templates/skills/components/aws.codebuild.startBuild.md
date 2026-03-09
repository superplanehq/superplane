# AWS CodeBuild • Start Build Skill

Use this guidance when planning or configuring `aws.codebuild.startBuild`.

## Purpose

`aws.codebuild.startBuild` starts an AWS CodeBuild build for a specified project and waits for it to reach a terminal status (SUCCEEDED, FAILED, STOPPED, TIMED_OUT, or FAULT). Use this when you want to trigger a build from a workflow and route downstream actions based on the result.

## Required Configuration

- `region` (required): The AWS region where the CodeBuild project exists. Default: `us-east-1`.
- `project` (required): The CodeBuild project name to build. Selected from integration resources.

## Optional Configuration

- `environmentVariables`: List of environment variable overrides for the build. Each entry has a `name` and `value`. Togglable — hidden by default.

## Planning Rules

When generating workflow operations that include `aws.codebuild.startBuild`:

1. Always set `configuration.region` and `configuration.project`.
2. `aws.codebuild.startBuild` emits on `passed` or `failed` channels (not `default`).
3. When connecting downstream nodes, set `source.handleId` explicitly:
   - `passed` for the success path
   - `failed` for failure/recovery paths
4. Never connect from this component with `source.handleId: "default"`.
5. Environment variable overrides are optional — omit if the project's default configuration is sufficient.

## Output Fields

- `data.build.project`: The CodeBuild project name.
- `data.build.id`: The full build ID (format: `project-name:build-uuid`).
- `data.build.status`: Terminal build status (`SUCCEEDED`, `FAILED`, `STOPPED`, `TIMED_OUT`, `FAULT`).
- `data.detail`: Raw EventBridge event detail with `project-name`, `build-id`, `build-status`, `current-phase`.

## Accessing Output in Downstream Nodes

- Build ID: `{{ $["Start Build"].build.id }}`
- Build status: `{{ $["Start Build"].build.status }}`
- Project name: `{{ $["Start Build"].build.project }}`

## Mistakes To Avoid

- Connecting from this component with `source.handleId: "default"` — use `passed` or `failed`.
- Forgetting to set `region` — it is required for the project resource picker to work.
- Using `getBuildStatus` when you want to start a new build — use `startBuild` instead.
