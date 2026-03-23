# Sentry On Issue Event

Use `sentry.onIssue` when a workflow should start from Sentry issue activity.

Configuration guidance:
- Use `project` to scope the trigger to a single Sentry project when the workflow should not react to every issue.
- Use `actions` to filter the webhook actions. Supported values are `created`, `assigned`, `resolved`, `archived`, and `unresolved`.

Payload guidance:
- The emitted payload includes the Sentry webhook `action`, `resource`, `installation`, optional `actor`, and the raw `data` object.
- The issue object is usually available at `data.issue`.
- Common expressions include `data.issue.id`, `data.issue.shortId`, `data.issue.title`, `data.issue.status`, and `data.issue.project.slug`.

Recommended patterns:
- Feed `data.issue.id` into `sentry.updateIssue` for follow-up automation.
- Use the project filter when separate workflows own different teams or services.
