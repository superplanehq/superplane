# GCP Cloud Build • Run Trigger Skill

Use this guidance when planning or configuring `gcp.cloudbuild.runTrigger`.

## Purpose

`gcp.cloudbuild.runTrigger` runs an existing Cloud Build trigger and waits for the resulting build to reach a terminal status. Use this when the build steps are already defined in a trigger, and you want to fire it from a workflow (optionally overriding the branch, tag, or commit).

## Required Configuration

- `trigger` (required): The Cloud Build trigger to run. Must be the trigger ID from the GCP project.

## Optional Configuration

- `branchName`: Override the branch to build from. Mutually exclusive with `tagName` and `commitSha`.
- `tagName`: Override the tag to build from. Mutually exclusive with `branchName` and `commitSha`.
- `commitSha`: Override the commit SHA to build from. Mutually exclusive with `branchName` and `tagName`.
- `projectId`: Override the GCP project from the integration.

## Planning Rules

When generating workflow operations that include `gcp.cloudbuild.runTrigger`:

1. Always set `configuration.trigger` to a valid Cloud Build trigger ID.
2. Set at most one of `branchName`, `tagName`, or `commitSha` — they are mutually exclusive.
3. Omit all revision fields to use the trigger's configured default.
4. `gcp.cloudbuild.runTrigger` emits on `passed` or `failed` channels (not `default`).
5. When connecting downstream nodes, set `source.handleId` explicitly:
   - `passed` for the success path
   - `failed` for failure/recovery paths
6. Never connect from this component with `source.handleId: "default"`.
7. The output `data` contains the full Cloud Build resource (same schema as `createBuild`).

## Output Fields

- `data.id`: The Cloud Build build ID.
- `data.status`: Terminal build status (`SUCCESS`, `FAILURE`, `INTERNAL_ERROR`, `TIMEOUT`, `CANCELLED`, `EXPIRED`).
- `data.buildTriggerId`: The trigger ID that produced this build.
- `data.logUrl`: URL to the build logs in the GCP console.
- `data.createTime`, `data.finishTime`: Build timestamps.
- `data.projectId`: The GCP project the build ran in.

## Accessing Output in Downstream Nodes

- Build ID: `{{ $["Run Trigger"].data.id }}`
- Build status: `{{ $["Run Trigger"].data.status }}`
- Log URL: `{{ $["Run Trigger"].data.logUrl }}`

## Mistakes To Avoid

- Setting more than one of `branchName`, `tagName`, `commitSha` simultaneously.
- Connecting from this component with `source.handleId: "default"` — use `passed` or `failed`.
- Using `createBuild` when the steps are already defined in a Cloud Build trigger — use `runTrigger` instead.
