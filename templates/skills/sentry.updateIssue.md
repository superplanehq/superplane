# Sentry Update Issue

Use `sentry.updateIssue` to change the state or assignee of an existing Sentry issue.

Configuration guidance:
- `issueId` should usually come from an upstream expression such as `$['On Issue Event'].data.issue.id`.
- At least one of `status` or `assignedTo` must be provided.
- Valid `status` values are `unresolved`, `resolved`, `resolvedInNextRelease`, and `ignored`.

Input guidance:
- `assignedTo` should be a Sentry user or team identifier accepted by the Sentry issue update API.
- This component emits the updated Sentry issue object on the default output channel.

Recommended patterns:
- Resolve issues after a successful remediation or deployment workflow.
- Reassign issues based on project, team, or severity from upstream trigger data.
