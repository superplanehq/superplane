# Sentry Update Issue

Use `sentry.updateIssue` to change the status of an existing Sentry issue.

Configuration guidance:
- `Issue` is selected from the connected Sentry issues list.
- `status` is required.
- Valid `status` values are `unresolved`, `resolved`, `resolvedInNextRelease`, and `ignored`.

Input guidance:
- This component emits the updated Sentry issue object on the default output channel.

Recommended patterns:
- Resolve or reopen issues after a remediation or deployment workflow.
