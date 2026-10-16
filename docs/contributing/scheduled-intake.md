# Scheduled intake

Enable `factory_scheduled_intake` on an organization to add Scheduled intake from
the Backlog column menu. The setup form asks for a daily time or a cron expression and an agent prompt.
It detects the browser timezone and stores its current offset, as the Schedule
component does. Fixed offsets do not adjust for daylight saving time. Cron
expressions accept five fields, or six fields with seconds first.

The read-only schedule preview returns the next run from the same scheduler
that activates the trigger. Invalid expressions prevent creation. Preview does
not create an automation or start a run.

The backend creates the bundled `scheduled-intake` template through
`CreateFactoryAutomation`. It resolves the repository, default branch, Git
integration, and hosted or BYOK agent from workspace settings. Checkout uses the
shared default-branch clone helper and integration setup credentials.

The graph connects Schedule to Agent. The agent's `passed` channel leads to a
proposal filter and Create Task. The `failed` channel leads to Add Run Error.

The prompt writes `/tmp/proposal.json`. A final output step validates the file and
writes structured JSON to `SUPERPLANE_RESULT_FILE`. A non-empty proposal must
include string fields `title` and `description`. An explicit `{}` creates no task.
A missing file, invalid JSON, or incomplete proposal fails the run. The prompt's
final text is not used as structured output.

Creating a scheduled intake does not start a manual run. The Schedule trigger
activates at its configured time. The existing manual schedule hook is available
for smoke tests:

`POST /api/v1/canvases/{id}/triggers/schedule/hooks/run`

The template proposes one task per run. It does not implement proposals or
provide cross-run deduplication.
