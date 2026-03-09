# AWS CodeBuild • On Build Completed Skill

Use this guidance when planning or configuring `aws.codebuild.onBuild`.

## Purpose

`aws.codebuild.onBuild` is a trigger that starts a workflow execution when AWS CodeBuild emits a build state change event via EventBridge. Use this to react to build completions, failures, or other state changes.

## Required Configuration

- `region` (required): The AWS region where build events are observed. Default: `us-east-1`.

## Optional Configuration

- `projects`: Optional project name filters (supports equals, not-equals, and regex matches). When empty, all projects in the region trigger the workflow.
- `states`: Optional list of build states to match (e.g., `SUCCEEDED`, `FAILED`, `STOPPED`, `TIMED_OUT`). When empty, all states trigger the workflow.

## Planning Rules

When generating workflow operations that include `aws.codebuild.onBuild`:

1. Always set `configuration.region`.
2. This is a trigger component — it emits on the `default` channel.
3. Use `projects` filters to scope events to specific CodeBuild projects.
4. Use `states` filters to scope events to specific build outcomes.

## Output Fields

The trigger emits the full EventBridge event payload:

- `data.detail.project-name`: CodeBuild project name.
- `data.detail.build-id`: The full build ID.
- `data.detail.build-status`: Build status (IN_PROGRESS, SUCCEEDED, FAILED, STOPPED, TIMED_OUT, FAULT).
- `data.detail.current-phase`: Current build phase.
- `data.region`: AWS region.
- `data.account`: AWS account ID.

## Accessing Output in Downstream Nodes

- Project name: `{{ $["On Build Completed"].detail["project-name"] }}`
- Build status: `{{ $["On Build Completed"].detail["build-status"] }}`
- Build ID: `{{ $["On Build Completed"].detail["build-id"] }}`

## Mistakes To Avoid

- Not setting `region` — it is required for EventBridge rule provisioning.
- Setting too broad filters — without project or state filters, all build events will trigger the workflow.
