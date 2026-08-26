# First-Class PR Feedback Handler

## Goal

Add PR feedback handling to an onboarding factory.

The factory will create a first-class PR feedback handler during onboarding.
The handler will own an editable factory canvas, as a factory intake does.
The canvas will address pull request feedback after the Verify step opens a pull
request.

The first version will:

- Require an exact `@superplaneagent` mention.
- Ignore comments from bots.
- Handle PR conversation comments.
- Handle one submitted review as one activation.
- Handle later replies in review threads.
- Run only one feedback agent at a time for each pull request.
- Scan all applicable feedback in each agent run.
- Keep the work order in Verify.
- Show queued, active, and completed runs in the factory UI.

## Confirmed GitHub behavior

GitHub uses three relevant webhook event types:

- `issue_comment.created` for a PR conversation comment.
- `pull_request_review.submitted` for a submitted review.
- `pull_request_review_comment.created` for each inline review comment or reply.

A submitted review with five inline comments produces:

- One `pull_request_review.submitted` delivery.
- Five `pull_request_review_comment.created` deliveries.

The submitted-review payload contains `review.body`. It does not contain the
inline review comments.

Fetch the inline comments for one review with:

```text
GET /repos/{owner}/{repo}/pulls/{pull_number}/reviews/{review_id}/comments
```

Use `review.id` and `pull_request.number` from the submitted-review payload.
Follow pagination because a review can contain more than 100 comments.

## Product behavior

### PR conversation comment

1. A human writes a PR conversation comment with `@superplaneagent`.
2. `github.onPRComment` emits one canvas event.
3. SuperPlane creates one canvas run.
4. The run finds the work order through the PR artifact URL.
5. The feedback runner waits if another runner is active for the same PR.
6. The runner scans current PR feedback, applies valid changes, tests, and
   pushes.

A second mentioned conversation comment creates a second canvas run. Its runner
waits for the first runner.

### Single inline review comment

GitHub normally sends one submitted-review delivery and one inline-comment
delivery.

1. `github.onPRReview` receives the submitted review.
2. The trigger fetches the review's inline comments.
3. The trigger finds the mention in the inline comment.
4. The trigger emits one canvas event.
5. `github.onPRReviewComment` ignores the top-level inline-comment delivery.
6. SuperPlane creates one canvas run.

### Submitted review with multiple comments

1. `github.onPRReview` receives the one submitted-review delivery.
2. The trigger fetches all comments for that review.
3. The trigger checks the review summary and all fetched comment bodies.
4. If any body contains the mention, the trigger emits one canvas event.
5. `github.onPRReviewComment` ignores the top-level comment deliveries.
6. SuperPlane creates one canvas run for the review.
7. The run scans all current applicable PR feedback.

The number of mentions in the review does not change the run count.

### Later reply in a review thread

1. A human adds a reply with `@superplaneagent`.
2. GitHub sends `pull_request_review_comment.created`.
3. The comment has `in_reply_to_id`.
4. `github.onPRReviewComment` emits one canvas event.
5. SuperPlane creates one canvas run.

### No mention

The trigger does not emit a canvas event. SuperPlane does not create a run.

## Domain model

Use `FactoryPRFeedbackHandler` as the backend type and **PR feedback** as the UI
name.

Follow the `FactoryIntake` ownership pattern:

- A database row owns the first-class identity.
- One factory canvas implements the handler.
- The live canvas graph owns executable behavior and settings.
- Ordinary canvas runs remain the execution records.
- The API projects domain-specific run data from canvas runs and node outputs.

Do not store a second copy of canvas configuration in the handler table.

## Database structure

Create the migration with:

```bash
make db.migration.create NAME=add-factory-pr-feedback-handlers
```

Do not create or edit the migration file manually.

The generated up migration must create this table:

```sql
CREATE TABLE factory_pr_feedback_handlers (
  id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  organization_id UUID NOT NULL,
  factory_id      UUID NOT NULL REFERENCES factories(id) ON DELETE RESTRICT,
  canvas_id       UUID NOT NULL REFERENCES workflows(id) ON DELETE RESTRICT,
  source          VARCHAR(64) NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_factory_pr_feedback_handlers_canvas_id
  ON factory_pr_feedback_handlers (canvas_id);

CREATE INDEX idx_factory_pr_feedback_handlers_factory_id
  ON factory_pr_feedback_handlers (factory_id);
```

Use `github-pull-requests` as the first `source` value.

Do not add a unique index on `(factory_id, source)`. A factory can support more
than one repository in a later version.

Do not add a `factory_pr_feedback_runs` table. Use these existing tables:

- `workflow_runs` for canvas runs.
- `workflow_events` for root events and node outputs.
- `workflow_node_executions` for queued, running, and finished nodes.
- `factory_work_order_artifacts` for PR-to-work-order lookup.

### Model API

Add `pkg/models/factory_pr_feedback_handler.go` with:

```go
type FactoryPRFeedbackHandler struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    FactoryID      uuid.UUID
    CanvasID       uuid.UUID
    Source         string
    CreatedAt      time.Time
    UpdatedAt      time.Time

    Canvas *Canvas `gorm:"foreignKey:CanvasID"`
}
```

Add these model operations:

- `(*Factory).CreatePRFeedbackHandler(tx, canvasID, source)`
- `(*Factory).FindPRFeedbackHandler(tx, handlerID)`
- `(*Factory).ListPRFeedbackHandlers(tx)`
- `(*FactoryPRFeedbackHandler).Delete(tx)`
- `DeleteFactoryPRFeedbackHandlersByCanvas(tx, canvasID)`

Scope all lookups by organization and factory.
Exclude rows whose canvases are soft-deleted.
Map the unique canvas constraint to a domain error.

Update:

- `pkg/models/factory_resource_cleaner.go`
- `pkg/workers/canvas_cleanup_worker.go`

Delete the ownership row before hard-deleting its canvas.

## Protobuf and REST API

Add the following RPCs to `protos/factories.proto`:

```proto
rpc ListFactoryPRFeedbackHandlers(
  ListFactoryPRFeedbackHandlersRequest
) returns (ListFactoryPRFeedbackHandlersResponse) {
  option (google.api.http) = {
    get: "/api/v1/factories/{factory_id}/pr-feedback-handlers"
  };
}

rpc CreateFactoryPRFeedbackHandler(
  CreateFactoryPRFeedbackHandlerRequest
) returns (CreateFactoryPRFeedbackHandlerResponse) {
  option (google.api.http) = {
    post: "/api/v1/factories/{factory_id}/pr-feedback-handlers"
    body: "*"
  };
}

rpc UpdateFactoryPRFeedbackHandler(
  UpdateFactoryPRFeedbackHandlerRequest
) returns (UpdateFactoryPRFeedbackHandlerResponse) {
  option (google.api.http) = {
    patch: "/api/v1/factories/{factory_id}/pr-feedback-handlers/{handler_id}"
    body: "*"
  };
}

rpc DeleteFactoryPRFeedbackHandler(
  DeleteFactoryPRFeedbackHandlerRequest
) returns (DeleteFactoryPRFeedbackHandlerResponse) {
  option (google.api.http) = {
    delete: "/api/v1/factories/{factory_id}/pr-feedback-handlers/{handler_id}"
  };
}

rpc ListFactoryPRFeedbackHandlerRuns(
  ListFactoryPRFeedbackHandlerRunsRequest
) returns (ListFactoryPRFeedbackHandlerRunsResponse) {
  option (google.api.http) = {
    get: "/api/v1/factories/{factory_id}/pr-feedback-handlers/{handler_id}/runs"
  };
}
```

Add these messages. Keep field numbers contiguous.

```proto
message FactoryPRFeedbackHandler {
  enum Source {
    SOURCE_UNSPECIFIED = 0;
    SOURCE_GITHUB_PULL_REQUESTS = 1;
  }

  message Settings {
    string repository = 1;
    string mention = 2;
    bool ignore_bots = 3;
  }

  string id = 1;
  string factory_id = 2;
  string canvas_id = 3;
  string name = 4;
  string description = 5;
  Source source = 6;
  Settings settings = 7;
  bool healthy = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message FactoryPRFeedbackHandlerRun {
  enum Trigger {
    TRIGGER_UNSPECIFIED = 0;
    TRIGGER_PR_COMMENT = 1;
    TRIGGER_PR_REVIEW = 2;
    TRIGGER_PR_REVIEW_REPLY = 3;
  }

  enum Status {
    STATUS_UNSPECIFIED = 0;
    STATUS_QUEUED = 1;
    STATUS_RUNNING = 2;
    STATUS_PASSED = 3;
    STATUS_FAILED = 4;
    STATUS_CANCELLED = 5;
  }

  string id = 1;
  string title = 2;
  string work_order_id = 3;
  string repository = 4;
  int64 pull_request_number = 5;
  string pull_request_url = 6;
  Trigger trigger = 7;
  string trigger_author = 8;
  string trigger_url = 9;
  Status status = 10;
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp started_at = 12;
  google.protobuf.Timestamp finished_at = 13;
}

message ListFactoryPRFeedbackHandlersRequest {
  string factory_id = 1;
}

message ListFactoryPRFeedbackHandlersResponse {
  repeated FactoryPRFeedbackHandler handlers = 1;
}

message CreateFactoryPRFeedbackHandlerRequest {
  string factory_id = 1;
  string name = 2;
  string repository = 3;
}

message CreateFactoryPRFeedbackHandlerResponse {
  FactoryPRFeedbackHandler handler = 1;
}

message UpdateFactoryPRFeedbackHandlerRequest {
  string factory_id = 1;
  string handler_id = 2;
  optional string name = 3;
  optional FactoryPRFeedbackHandler.Settings settings = 4;
}

message UpdateFactoryPRFeedbackHandlerResponse {
  FactoryPRFeedbackHandler handler = 1;
}

message DeleteFactoryPRFeedbackHandlerRequest {
  string factory_id = 1;
  string handler_id = 2;
}

message DeleteFactoryPRFeedbackHandlerResponse {}

message ListFactoryPRFeedbackHandlerRunsRequest {
  string factory_id = 1;
  string handler_id = 2;
  int32 limit = 3;
  google.protobuf.Timestamp before = 4;
}

message ListFactoryPRFeedbackHandlerRunsResponse {
  repeated FactoryPRFeedbackHandlerRun runs = 1;
  google.protobuf.Timestamp last_timestamp = 2;
}
```

Creation rules:

- Default `name` to `Address PR feedback`.
- Default `repository` to the factory onboarding app repository.
- Return `InvalidArgument` when no repository is available.
- Set `mention` to `@superplaneagent`.
- Set `ignore_bots` to `true`.
- Resolve the GitHub integration and selected agent from factory onboarding.
- Create and publish the canvas before inserting the ownership row.
- Soft-delete the canvas if ownership-row creation fails.
- Do not seed historical comments or reviews.

Update rules:

- Rename the canvas when `name` changes.
- Update all GitHub trigger nodes when `repository`, `mention`, or
  `ignore_bots` changes.
- Publish the updated graph transactionally.
- Update the handler row's `updated_at` after a successful graph update.
- Reject settings that leave the repository or mention empty.

Delete rules:

- Delete the ownership row and soft-delete the canvas in one transaction.
- Keep the down migration empty, as required by repository policy.

Authorization:

- Use `factories:read` for list and list-runs.
- Use `factories:update` for create, update, and delete.
- Keep all endpoints behind the `factories` feature.
- Register all routes in `pkg/authorization/gateway_auth_rules.go`.

After proto changes, run:

```bash
make pb.gen
make check.proto.field.numbers
```

Do not edit generated files by hand.

## GitHub trigger changes

### Add `github.onPRReview`

Add:

- `pkg/integrations/github/components/pulls/on_pr_review.go`
- `pkg/integrations/github/components/pulls/on_pr_review_test.go`

Add this paginated operation to `pkg/integrations/github/common/client.go`:

```go
func (c *Client) ListPullRequestReviewComments(
    ctx context.Context,
    repository string,
    pullNumber int,
    reviewID int64,
) ([]*github.PullRequestComment, error)
```

Create the client in the webhook handler with `ctx.Integration` and `ctx.HTTP`.
Use the existing GitHub client transport so tests can control HTTP responses.

The trigger must:

- Subscribe only to `pull_request_review`.
- Accept only action `submitted`.
- Verify the webhook signature before processing.
- Read `review.body`, `review.id`, and `pull_request.number`.
- Fetch all review comments with the REST endpoint for that review.
- Follow all result pages.
- Check the summary and comment bodies for the configured content filter.
- Ignore the event when the review author is a bot and `ignoreBots` is true.
- Return a retriable server error when GitHub comment loading fails.
- Emit one `github.prReview` event after all checks pass.
- Add the fetched comments under `review_comments` in the emitted payload.

Do not emit one event per fetched comment.

Register the component in the GitHub component registry, capability mapper, UI
mapper, docs, and fixtures.

### Restrict `github.onPRReviewComment` for this handler

The current component listens to both `pull_request_review` and
`pull_request_review_comment`.

Add backward-compatible configuration:

```text
includeReviewSubmissions: boolean, default true
commentScope: all | replies, default all
ignoreBots: boolean, default false
```

The generated PR feedback handler must set:

```text
includeReviewSubmissions: false
commentScope: replies
ignoreBots: true
```

In `replies` scope, accept only comments with a non-null `in_reply_to_id`.
This suppresses top-level comment deliveries already represented by
`github.onPRReview`.

Existing canvases keep the current default behavior.

### Update `github.onPRComment`

Add `ignoreBots` as a backward-compatible common configuration field.
Default it to `false`.
The generated handler sets it to `true`.

Use one exact mention matcher for all three triggers.
Do not use a raw substring check.
For example, `@superplaneagent-old` must not match `@superplaneagent`.

## Generated canvas

Generate and publish the canvas in the backend.
Follow the intake template pattern in:

- `pkg/grpc/actions/factories/intake_template.go`
- `pkg/grpc/actions/factories/intake_graph.go`
- `pkg/grpc/actions/factories/intake_settings.go`

Use stable node IDs:

```text
on-pr-comment
on-pr-review
on-pr-review-reply
find-work-order
address-pr-feedback
```

Use this logical graph:

```text
github.onPRComment -----------\
github.onPRReview -------------+--> findWorkOrder --> address feedback
github.onPRReviewComment -----/          |                  |
                                         | found            |
                                         v                  v
                                      runner              finish

findWorkOrder.notFound --> finish
```

The `findWorkOrder` node must use the PR URL artifact key.
Use the existing paths:

- PR conversation: `root().data.issue.pull_request.html_url`
- Review or reply: `root().data.pull_request.html_url`

Keep PR Closure on the same artifact-key convention.
Do not add a PR artifact key migration in this change.

### Runner concurrency

Set concurrency only on the `address-pr-feedback` runner:

```yaml
concurrency:
  key: 'github-{{ repository_id }}-pr-{{ pull_request_number }}'
  max: 1
```

Resolve `repository_id` and `pull_request_number` from the root event.

Put all mutating agent work in this one runner node:

- Load the current PR and branch.
- Clone the repository.
- Fetch current PR conversation comments.
- Fetch submitted review summaries.
- Fetch review threads and resolution state through GraphQL.
- Validate each requested change.
- Apply valid changes.
- Run relevant tests.
- Commit and push.

Do not split clone, edit, test, commit, or push across separate nodes.
Concurrency applies per node, not per canvas run.

When run B starts while run A is active:

- Run B can execute `findWorkOrder`.
- Run B waits at `address-pr-feedback`.
- The UI reports run B as queued.
- Different PR keys can run in parallel.

### Agent feedback policy

The prompt must use `/babysit` behavior:

- Read unresolved review threads.
- Read mentioned PR conversation comments.
- Check whether each request is valid before changing code.
- Apply valid requests.
- Explain disagreements instead of making unsafe changes.
- Ignore bot comments.
- Ignore SuperPlane's own replies.
- Stop after current feedback is addressed.

Do not automatically resolve GitHub review threads.
The reviewer controls thread resolution.

Do not report a work-order check.
Do not add a work-order comment or result event.
The ordinary canvas run is the only feedback execution result.

## Graph settings and health

Add:

- `pr_feedback_template.go`
- `pr_feedback_graph.go`
- `pr_feedback_settings.go`

Resolve graph roles by stable node ID first.
Use component type as a fallback where it is unambiguous.

Settings are a facade over the live graph:

- `repository` reads and writes all three trigger nodes.
- `mention` reads and writes all three content filters.
- `ignore_bots` reads and writes all three trigger nodes.

The handler is healthy only when:

- All three trigger roles exist.
- Every trigger can reach `findWorkOrder`.
- The found branch can reach the feedback runner.
- The repository binding is present on all triggers.
- The runner has a supported agent component.

Keep an unhealthy handler in list responses.
The UI must let the user open and repair its canvas.

## Run projection

Implement `list_factory_pr_feedback_handler_runs.go`.

Use limits of 50 by default and 200 maximum.
Page backward with `before`, as intake runs do.

For each page, batch-load:

- Root events by run ID.
- Node executions by run ID.
- Work orders by PR artifact key.

Do not issue one database query per run.

Derive fields as follows:

- `id`: canvas run ID.
- `title`: `Address feedback on PR #<number>`.
- `repository`: root webhook repository full name.
- `pull_request_number`: root webhook PR number.
- `pull_request_url`: root webhook PR URL.
- `trigger`: root event type and payload shape.
- `trigger_author`: comment or review author.
- `trigger_url`: comment URL or review URL.
- `work_order_id`: PR artifact-key lookup result.
- Times: run and runner execution timestamps.

Derive status in this order:

1. `CANCELLED` when the canvas run was cancelled.
2. `FAILED` when the canvas run or runner failed.
3. `PASSED` when the canvas run completed successfully.
4. `RUNNING` when the runner execution is started.
5. `QUEUED` when the runner is pending or not yet created.

The run endpoint must tolerate an edited or unhealthy graph.
Return the available metadata instead of dropping the run.

## Onboarding

Add an idempotent `provisionPRFeedbackHandler` beside
`provisionGithubIntake`.

Run it after onboarding saves:

- The application repository.
- The GitHub integration.
- The selected agent.

Idempotency rule:

1. List existing GitHub PR feedback handlers.
2. Compare each handler's repository setting.
3. Reuse a handler for the selected repository.
4. Create one only when no match exists.

Keep PR Closure as a separate event app.
Do not add the feedback handler as a line step.
Do not seed historical PR comments.

Update the Verify automation copy to tell users:

```text
Mention `@superplaneagent` in a pull request comment or review to request changes.
```

## Frontend

Add a PR feedback data hook based on `useFactoryIntakeData.ts`.
It must support:

- List handlers.
- Update and delete a handler.
- List handler runs.
- Poll active runs every 10 seconds.
- Invalidate handler and factory-app queries after mutations.

Add **PR feedback** to the factory sidebar.
Reuse the intake settings shell:

- **General**: name, repository, mention, bot policy, and health.
- **Runs**: queued, running, passed, failed, and cancelled runs.
- **Automation**: graph preview and **Edit automation**.

Each run row must link to:

- The existing canvas run inspector.
- The associated work order.
- The pull request or triggering comment.

### Verify visibility

Match handler runs to work orders through `work_order_id`.

Show this status on the Verify card:

```text
Addressing PR feedback
```

Show it while any matching run is queued or running.
Link it to the oldest active run.

When no run is active, keep:

```text
Listening for user review
```

Do not change the persisted work-order state.
Do not create a new line execution.
Do not use status notes as the durable run relationship.

## Backend file plan

Add:

- `pkg/models/factory_pr_feedback_handler.go`
- `pkg/models/factory_pr_feedback_handler_test.go`
- `pkg/grpc/actions/factories/create_factory_pr_feedback_handler.go`
- `pkg/grpc/actions/factories/list_factory_pr_feedback_handlers.go`
- `pkg/grpc/actions/factories/update_factory_pr_feedback_handler.go`
- `pkg/grpc/actions/factories/delete_factory_pr_feedback_handler.go`
- `pkg/grpc/actions/factories/list_factory_pr_feedback_handler_runs.go`
- `pkg/grpc/actions/factories/pr_feedback_template.go`
- `pkg/grpc/actions/factories/pr_feedback_graph.go`
- `pkg/grpc/actions/factories/pr_feedback_settings.go`
- Tests beside each action and graph concern.
- `pkg/integrations/github/components/pulls/on_pr_review.go`
- `pkg/integrations/github/components/pulls/on_pr_review_test.go`

Modify:

- `protos/factories.proto`
- `pkg/grpc/factory_service.go`
- `pkg/grpc/actions/factories/serialization.go`
- `pkg/grpc/actions/factories/errors.go`
- `pkg/authorization/gateway_auth_rules.go`
- `pkg/models/factory_resource_cleaner.go`
- `pkg/workers/canvas_cleanup_worker.go`
- `pkg/integrations/github/github.go`
- `pkg/integrations/github/capability_mapper.go`
- `pkg/integrations/github/common/client.go`
- `pkg/integrations/github/components/pulls/on_pr_comment.go`
- `pkg/integrations/github/components/pulls/on_pr_review_comment.go`
- GitHub component docs and examples.

## Frontend file plan

Add:

- `web_src/src/hooks/useFactoryPRFeedbackData.ts`
- PR feedback drawer and settings components under
  `web_src/src/pages/factories/pages/`.
- Models and tests beside those components.
- `web_src/src/pages/app/mappers/github/on_pr_review.ts`

Modify:

- `web_src/src/pages/factories/pages/onboarding/onboardingProvision.ts`
- `web_src/src/pages/factories/pages/onboarding/useFinishOnboarding.ts`
- `web_src/src/pages/factories/layout/FactoriesSidebarNav.tsx`
- Verify-card model and rendering files.
- `web_src/src/pages/home/factories/line-apps/pr.canvas.yaml`
- GitHub mapper registration.

Do not edit these generated directories manually:

- `pkg/protos/`
- `pkg/openapi_client/`
- `web_src/src/api-client/`
- `api/`

## Test plan

### GitHub trigger tests

- Accept a submitted review with a mention in its summary.
- Accept a submitted review with a mention in one inline comment.
- Fetch all pages of review comments.
- Emit one event for a review with five comments.
- Ignore the five top-level comment deliveries in handler configuration.
- Accept a later mentioned reply.
- Ignore a reply without the mention.
- Ignore bot reviews and comments.
- Reject near matches such as `@superplaneagent-old`.
- Return a retriable error when GitHub comment loading fails.
- Preserve current `github.onPRReviewComment` defaults for existing canvases.

### Backend domain tests

- Create a live factory-owned canvas and ownership row.
- Clean up the canvas when row creation fails.
- Prevent one canvas from implementing two handlers.
- Scope list and find by organization and factory.
- Hide handlers whose canvas is soft-deleted.
- Delete the handler and soft-delete its canvas.
- Remove ownership during hard canvas cleanup.
- Serialize settings from the live graph.
- Update all trigger settings together.
- Detect each unhealthy graph condition.
- Keep unhealthy handlers visible.
- Enforce API authorization.

### Run tests

- Project PR comment, review, and reply runs.
- Show a second same-PR runner as queued.
- Run different PR partitions in parallel.
- Derive all run statuses.
- Derive work-order association from the PR artifact.
- Tolerate missing nodes after a canvas edit.
- Page runs without per-run database queries.

### Frontend tests

- Provision one handler during onboarding.
- Reuse the handler when onboarding retries.
- Show healthy and unhealthy states.
- Update repository and mention settings.
- Open the automation editor.
- Show active and completed runs.
- Link runs to the inspector, work order, and GitHub.
- Show `Addressing PR feedback` on the Verify card.
- Restore `Listening for user review` after completion.
- Show failure and cancellation states.

## Verification commands

Run these commands during implementation:

```bash
make db.migrate DB_NAME=superplane_dev
make pb.gen
make check.proto.field.numbers
make format.go
make format.js
make lint
make check.build.app
make test
make check.build.ui
```

Run targeted package and UI tests before the full test commands.

## Implementation order

Implement the change in this order:

1. Add and test `github.onPRReview`.
2. Add the backward-compatible comment-scope and bot-filter options.
3. Add the ownership migration and model.
4. Add the generated canvas, settings facade, and health checks.
5. Add CRUD and run-projection APIs.
6. Regenerate clients and add onboarding provisioning.
7. Add PR feedback settings and run UI.
8. Add the Verify-card active-run state.
9. Run targeted tests, then all required checks.

## Deferred work

Do not include these changes in the first implementation:

- Durable deduplication by `X-GitHub-Delivery`.
- A quiet-period dispatcher for multiple independent mentions.
- Automatic review-thread resolution.
- Automatic reaction to unmentioned comments.
- Bot allowlists.
- Historical comment replay.
- A migration from PR URL artifact keys to repository-ID keys.

GitHub retries can create a duplicate serialized run in the first version.
That run can pass without making additional changes.
