# CodeBuild-Backed Bash Component

## Overview

This PRD defines a new core SuperPlane action component that runs user-provided Bash commands in an isolated, ephemeral build environment.

The component should feel like "run this job in a temporary VM" to workflow builders, while using AWS CodeBuild as the backend execution substrate. SuperPlane owns the orchestration, lifecycle management, repository checkout, log retrieval, cancellation, and result normalization so users do not need to model CodeBuild concepts directly in their workflows.

## Problem Statement

Workflow builders often need an ephemeral build/provisioning environment inside a workflow:

- Clone a repository and run project scripts.
- Build and push Docker images.
- Run Terraform plan/apply/destroy commands.
- Execute release, deployment, or migration scripts.
- Run diagnostic or remediation commands with standard CLI tools.

Today, users can run commands through SSH when they already have a reachable host, but that requires them to manage long-lived infrastructure, credentials, networking, and cleanup. For many workflow steps, users need a clean, temporary machine with Git, Docker, Terraform, Bash, and controlled access to secrets.

Without a managed build execution component, users either overuse external CI systems, maintain custom runners, or encode build/provisioning automation outside SuperPlane, which reduces observability and makes workflows harder to operate.

## Goals

1. Add a core action component that executes Bash commands in an isolated, ephemeral build environment.
2. Use AWS CodeBuild as the initial backend for provisioning and running those environments.
3. Support repository checkout as a first-class part of the execution flow.
4. Support common build/provisioning workloads, including Docker image builds and Terraform commands.
5. Provide first-class SuperPlane observability for command status, logs, duration, exit code, artifacts, and emitted payload.
6. Allow workflow payload values and organization secrets to be passed into commands safely.
7. Support deterministic routing for successful and failed command outcomes.
8. Keep the user-facing component simple and avoid exposing unnecessary CodeBuild implementation details.

## Non-Goals

- Providing a general-purpose interactive terminal.
- Supporting long-running daemon workloads.
- Replacing dedicated CI/CD systems for full pipeline orchestration.
- Replacing dedicated integration components for common APIs.
- Providing SSH access into the execution environment.
- Supporting arbitrary cloud providers in v1.
- Exposing raw CodeBuild project management as a user-facing integration feature.
- Guaranteeing workload portability beyond the documented default runtime image and configuration.
- Managing Terraform state backends on behalf of users.

## Primary Users

- **Workflow Builders**: Users who need to add custom build, release, or provisioning steps to workflows.
- **Platform Engineers**: Users who want controlled, auditable infrastructure and remediation scripts.
- **Support/Ops Teams**: Users who need repeatable diagnostics or one-off checks triggered by events.

## User Stories

1. As a workflow builder, I want to clone a repository and run project scripts so workflow automation can operate on real source code.
2. As a developer, I want to build and push Docker images so SuperPlane workflows can produce deployable artifacts.
3. As a platform engineer, I want to run Terraform commands so infrastructure changes can be triggered, reviewed, and observed from workflows.
4. As a platform engineer, I want commands to run in an isolated environment so scripts do not share filesystem or process state across runs.
5. As a user, I want to pass values from upstream payloads into the command so scripts can act on the event that started the workflow.
6. As an operator, I want to see command output, exit code, duration, artifacts, and logs so I can debug failed runs quickly.
7. As a security-conscious admin, I want secrets to be passed securely and redacted from visible output where possible.
8. As a workflow author, I want different output channels for successful and failed commands so downstream routing is straightforward.

## Functional Requirements

### Component Identity

- The component must be a core action component.
- Proposed component name: `runner`.
- Proposed label: `Runner`.
- Proposed icon: `terminal` from Lucide.
- Proposed color: `blue`, matching existing command-oriented core components.

### Configuration

The component should expose the following configuration fields:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `source` | `object` | No | None | Optional repository to clone before running commands. |
| `commands` | `text` | Yes | None | Bash commands or script body to execute. Supports expressions. |
| `environment` | `list` | No | Empty | Name/value pairs exposed as environment variables. Values may use expressions. |
| `secrets` | `list` | No | Empty | Environment variables sourced from organization secrets. |
| `timeout` | `number` | No | `600` | Maximum execution time in seconds. |
| `runtimeImage` | `select` | No | SuperPlane default image | Supported execution image/runtime. |
| `computeSize` | `select` | No | Small | Backend compute size tier. |
| `docker` | `object` | No | Disabled | Options for Docker builds, including privileged mode and registry credentials. |
| `artifacts` | `object` | No | Disabled | Optional paths to collect after the command completes. |

Configuration guidelines:

- `commands` should use a multiline editor.
- `source` should support Git repository URL, ref/branch/tag/SHA, and optional checkout credentials.
- Environment variable names must match standard shell variable naming rules: `^[A-Za-z_][A-Za-z0-9_]*$`.
- Secret values must be selected using SuperPlane secret references, not entered as plain text.
- Docker options, artifacts, runtime image, compute size, and timeout should be grouped or visually de-emphasized so the common path remains simple.
- The default configuration should be close to runnable after the user enters commands.

### Repository Source

- The component should support cloning a Git repository before executing commands.
- Users must be able to specify:
  - Repository URL.
  - Ref, branch, tag, or commit SHA.
  - Optional checkout depth.
  - Optional credentials through organization secrets.
- The checkout path should be predictable and documented.
- Checkout metadata should be included in execution metadata and emitted payload:
  - Repository URL or sanitized repository identifier.
  - Requested ref.
  - Resolved commit SHA, when available.
- Authentication failures during checkout should be `error` because the component could not execute the requested command.

### Docker Builds

- The default runtime image should include Docker tooling or clearly document the supported Docker build path.
- Docker builds that require Docker daemon access should run only when Docker support is explicitly enabled for the component execution.
- If CodeBuild privileged mode is required, the component and backend must make that security boundary explicit.
- Users should be able to provide registry credentials through organization secrets.
- Common commands such as `docker build`, `docker login`, and `docker push` should be supported in the documented happy path.
- Docker build failures caused by the user's build or push command should emit on the `failed` channel.
- Backend failures that prevent Docker from being available should use the SuperPlane `error` state.

### Terraform Runs

- The default runtime image should include Terraform or provide a documented supported image that includes Terraform.
- Users should be able to inject cloud provider credentials, Terraform variables, and backend credentials through environment variables and secrets.
- The component should not manage Terraform remote state backends in v1.
- Terraform command failures, including plan/apply validation failures, should emit on the `failed` channel when Terraform runs and exits non-zero.
- Backend or credential injection failures that prevent Terraform from starting should use the SuperPlane `error` state.

### Backend Execution

- SuperPlane must submit each component execution to AWS CodeBuild.
- Each execution must run in an isolated, ephemeral CodeBuild build environment.
- The backend must inject the command script, source checkout configuration, environment variables, secrets, timeout, artifact configuration, and metadata needed to correlate the CodeBuild run with the SuperPlane run item.
- The backend must collect:
  - Build ID.
  - Build ARN.
  - Build status.
  - Start time.
  - Finish time.
  - Exit code, when available.
  - Standard output.
  - Standard error, when available.
  - Artifact metadata, when configured.
  - CloudWatch log group and stream references, when available.
- SuperPlane must normalize CodeBuild statuses into SuperPlane execution states and output channels.
- SuperPlane must not require users to understand or configure CodeBuild projects in the component UI.

### CodeBuild Resource Model

The v1 implementation should prefer a small number of managed reusable CodeBuild projects over creating a new project for every run.

Requirements:

- Projects must be scoped to the SuperPlane deployment, environment, runtime image, compute-size, and Docker support combination.
- Per-run data must be passed through build overrides.
- Build names, tags, or environment variables must include SuperPlane correlation IDs for traceability.
- The backend must cleanly handle missing projects by creating or reconciling them through infrastructure setup, not during arbitrary workflow execution unless explicitly designed.
- Projects that enable Docker privileged mode must be separate from non-privileged projects.

### Inputs

- Commands, source refs, and environment values may reference upstream workflow payloads through existing expression support.
- The component should receive the full message chain through normal SuperPlane execution context.
- Large payloads should not be automatically written to files in v1 unless explicitly configured in a future iteration.
- Source checkout credentials, registry credentials, Terraform credentials, and other sensitive inputs must use organization secrets.

### Outputs

The component must emit on one of two output channels:

| Channel | When emitted |
|---------|--------------|
| `success` | Command completes and exits with code `0`. |
| `failed` | Command runs to completion and exits with a non-zero code. |

Unexpected failures, such as failure to submit to CodeBuild, missing backend configuration, auth errors, or log retrieval errors that prevent determining the result, should use the SuperPlane `error` state rather than the `failed` channel.

The emitted payload should be flat and expression-friendly:

```json
{
  "command": {
    "exitCode": 0,
    "status": "SUCCEEDED",
    "durationSeconds": 42,
    "stdout": "hello\n",
    "stderr": "",
    "source": {
      "repository": "github.com/example/app",
      "ref": "main",
      "commitSha": "abc123"
    },
    "artifacts": [
      {
        "name": "image-digest.txt",
        "path": "dist/image-digest.txt"
      }
    ],
    "buildId": "superplane-runner:example",
    "buildArn": "arn:aws:codebuild:...",
    "logUrl": "https://console.aws.amazon.com/..."
  }
}
```

Payload requirements:

- Include `exitCode` when known.
- Include `stdout` and `stderr` with documented size limits.
- Include source checkout metadata when a repository is configured.
- Include artifact metadata when artifacts are configured.
- Include stable backend identifiers for support/debugging.
- Include a log URL when available.
- Do not include secret values.
- Truncate oversized output deterministically and expose truncation metadata when truncation occurs.

### Execution State and UI

- The node footer and run history must show running, success, failed, cancelled, and error states.
- A running execution should show elapsed time.
- A completed execution subtitle may show `Exit 0`, `Exit 1`, or similar short context plus timestamp.
- The node metadata should show at most:
  - Repository name, if configured.
  - Runtime image or short runtime label.
  - Timeout, only if customized.
  - Docker enabled or compute size, only if customized.
- The details tab must show:
  - Started at.
  - Finished at, when available.
  - Duration.
  - Exit code, when available.
  - Repository and commit SHA, when available.
  - Artifact count or artifact link, when available.
  - CodeBuild build ID or ARN.
  - Link to backend logs, when available.
  - Error message last, when applicable.
- The payload tab remains the place for raw emitted output.

### Cancellation

- If a user cancels a running SuperPlane execution, SuperPlane must attempt to stop the associated CodeBuild build.
- The run item should move to `cancelled` when cancellation succeeds or when CodeBuild reports a stopped state.
- If cancellation cannot be completed, the run item should show an actionable error.

### Logs and Output Limits

- SuperPlane should capture enough output for common debugging without allowing unbounded storage growth.
- v1 should define explicit limits for:
  - Maximum captured stdout bytes.
  - Maximum captured stderr bytes.
  - Maximum rendered log lines in the details view.
- Full logs should remain available through the backend log link when possible.
- Secret-looking values should be redacted in SuperPlane-rendered output where feasible, but users should still be warned not to echo secrets.

### Artifacts

- Users should be able to configure optional artifact paths to collect after successful or failed runs.
- Artifact paths must be relative to the checkout root and must not allow reading outside the execution workspace.
- Artifacts should be stored in backend-managed storage with access controlled by SuperPlane.
- The emitted payload should include artifact metadata, not raw artifact contents.
- Artifacts are intended for build outputs such as image digest files, Terraform plans, reports, and logs that are too large for stdout.

### Security Requirements

- Commands must run in isolated ephemeral environments with no shared writable state across runs.
- Secrets must be injected only for the duration of the execution.
- Secrets must not be persisted in component configuration, metadata, emitted payloads, or logs intentionally stored by SuperPlane.
- The default runtime environment should have least-privilege AWS permissions required for command execution and logging.
- User commands should not receive AWS credentials unless explicitly configured through organization secrets or a future scoped credentials feature.
- Docker privileged mode must be opt-in and isolated from non-Docker executions.
- Repository, registry, Terraform, and cloud provider credentials must be handled as secrets.
- The backend must prevent command execution from accessing SuperPlane internal services beyond documented network access.
- All command executions should be auditable by organization, canvas, workflow node, run item, actor or trigger, and timestamp.

### Backend IAM Requirements

The CodeBuild backend requires AWS permissions for SuperPlane-managed infrastructure, including:

- Starting builds.
- Stopping builds.
- Reading build status.
- Reading CloudWatch logs for the build.
- Passing the CodeBuild service role to CodeBuild.
- Accessing backend-managed artifact storage when artifacts are enabled.

The CodeBuild service role should have only the permissions needed to run the build, write logs, access configured artifact storage, and access explicitly configured runtime resources.

### Timeouts and Quotas

- Component-level timeout must be enforced by both SuperPlane and CodeBuild where possible.
- SuperPlane should surface timeout as a failed command outcome if CodeBuild started the build and the script exceeded its allowed runtime.
- Submission failures caused by CodeBuild throttling, quota exhaustion, or backend unavailability should be `error`.
- v1 should document deployment-level concurrency and timeout limits.

## UX Requirements

- The configuration form should make the main path obvious: choose an optional repository, enter commands, optionally add env vars/secrets, run.
- The component description should clearly state that commands run in a managed ephemeral build environment.
- The UI must avoid showing raw CodeBuild terminology unless it helps users debug a specific run.
- Error messages should distinguish:
  - Invalid configuration.
  - Repository checkout failed.
  - Command exited non-zero.
  - Backend could not start the run.
  - Backend run timed out.
  - Backend logs could not be retrieved.
- Details view should make it easy to copy output and backend identifiers.

## Implementation Scope (v1)

- Core `Runner` action component.
- CodeBuild-backed asynchronous execution.
- Two output channels: `success` and `failed`.
- Cancellation via CodeBuild stop build.
- Git repository checkout with secret-backed credentials.
- Docker builds through an explicitly enabled Docker-capable runtime path.
- Terraform execution through the default or a documented supported runtime image.
- Basic runtime selection from a supported list.
- Environment variables and secret-backed environment variables.
- Captured stdout/stderr with size limits.
- Optional artifact metadata for configured artifact paths.
- Frontend mapper for node metadata, states, subtitles, and details.
- Unit tests for configuration validation, state routing, payload generation, and error handling.

## Out of Scope (v1)

- Interactive shell sessions.
- Streaming logs into the UI in real time.
- User-provided Docker images.
- Customer-owned AWS account execution.
- Persistent workspaces or caches.
- SuperPlane-managed Terraform state backends.
- Rich artifact browser UI beyond links/metadata.
- Scheduled cleanup UI for backend resources.
- Per-step approvals inside the script.

## Acceptance Criteria

1. A user can add the `Runner` core component to a workflow and configure a multiline Bash script.
2. A user can configure repository checkout and run commands against the checked-out source.
3. A user can build a Docker image in the supported Docker runtime path.
4. A user can run Terraform commands with secret-backed provider/backend credentials.
5. A successful script exits through the `success` output channel with exit code, duration, output, source metadata, artifact metadata, and backend identifiers in the emitted payload.
6. A non-zero script exits through the `failed` output channel with exit code and captured output.
7. Backend submission, checkout, infrastructure, or credential injection errors show as SuperPlane `error` states and do not emit `success` or `failed`.
8. Running executions show elapsed time in the UI.
9. Completed executions show useful details, including timestamps, duration, exit code, source revision, artifact links, and backend log link when available.
10. Cancelling a running execution attempts to stop the CodeBuild build and reflects the cancelled state in SuperPlane.
11. Secret-backed environment variables can be configured without exposing secret values in configuration, metadata, or emitted payloads.
12. Output capture respects documented size limits and records when truncation occurred.
13. The component has backend unit tests and frontend mapper tests covering success, failed, running, cancelled, and error states.

## Success Metrics

- Adoption: number of workflows using `Runner`.
- Workload coverage: share of executions that use repository checkout, Docker builds, Terraform commands, and artifacts.
- Reliability: percentage of executions that reach a terminal SuperPlane state without backend reconciliation.
- Debuggability: reduction in support cases where users cannot determine why a command failed.
- Performance: median time from SuperPlane execution start to CodeBuild start.
- Safety: zero confirmed incidents of secrets being persisted in emitted payloads or node metadata.

## Risks and Mitigations

- **Risk:** Users treat the component as a general-purpose compute platform.  
  **Mitigation:** Document timeout, concurrency, and workload limits clearly; reject unsupported long-running patterns.

- **Risk:** CodeBuild startup latency makes workflows feel slow.  
  **Mitigation:** Set expectations in documentation, track queue/start latency, and evaluate warm pool or alternative runner backends later.

- **Risk:** Command output leaks secrets.  
  **Mitigation:** Use secret references, redact known secret values in SuperPlane-rendered output where feasible, and warn users not to echo secrets.

- **Risk:** Backend quotas throttle workflow execution.  
  **Mitigation:** Define concurrency controls, expose actionable quota errors, and add deployment-level monitoring.

- **Risk:** Docker privileged mode increases the impact of untrusted commands.  
  **Mitigation:** Make Docker support opt-in, run it on separate CodeBuild projects, document the trust boundary, and apply strict IAM/network limits.

- **Risk:** Terraform runs mutate production infrastructure from workflow context.  
  **Mitigation:** Require explicit credentials, recommend plan-before-apply workflows, and keep audit records for command, actor, run item, and source revision.

- **Risk:** Repository checkout creates confusing failures before user commands start.  
  **Mitigation:** Represent checkout failures as `error`, show checkout diagnostics in details, and include resolved source metadata when checkout succeeds.

- **Risk:** CodeBuild implementation details leak into the user experience.  
  **Mitigation:** Keep CodeBuild identifiers in details/support contexts while presenting the component as managed Bash execution.

- **Risk:** Cancellation and status polling drift from actual backend state.  
  **Mitigation:** Store backend build IDs, reconcile terminal states, and make cancellation idempotent.

## Rollout Plan

1. Build backend proof of concept that checks out a repository, submits a command to CodeBuild, polls status, retrieves logs, and normalizes the result.
2. Implement the core component behind a feature flag.
3. Validate Docker build and Terraform command examples in the supported runtime path.
4. Add frontend mapper and validate node/run details against component design guidelines.
5. Enable internally for SuperPlane workflows and collect latency, reliability, output-size, checkout, Docker, and Terraform data.
6. Document security expectations, limits, and example use cases.
7. Roll out to selected users with concurrency limits.
8. Promote to general availability after reliability and safety metrics meet product thresholds.

## Open Questions

1. Should CodeBuild run in a SuperPlane-owned AWS account, a deployment-owned AWS account, or customer-owned AWS accounts in future versions?
2. What are the v1 limits for maximum runtime, output capture size, and concurrent executions per organization?
3. Which runtime images should be supported at launch, and who owns their patching cadence?
4. Should v1 expose a `computeSize` selector, or should compute size remain an internal default until demand is clear?
5. Should command timeout be considered `failed` when the build starts successfully, or should it be represented as a distinct output channel later?
6. How much log redaction can be guaranteed before we need to state that users are responsible for not printing secrets?
7. Which Git providers and authentication methods should repository checkout support at launch?
8. Should Docker builds support only push-by-script, or should the component have first-class image name/tag/digest fields?
9. Should Terraform plan files be treated as artifacts with special display behavior?
10. What audit events should be emitted for checkout, command execution, Docker privileged mode, artifact upload, cancellation, and secret injection?
