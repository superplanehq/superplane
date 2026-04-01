# Grafana Delete Alert Rule

Use this component when a workflow needs to delete an existing Grafana-managed alert rule.

Good fits:
- removing temporary alert rules after an incident or rollout
- cleaning up obsolete rules during service retirement
- pairing deletions with approvals or audit notifications

Expected input:
- a Grafana alert rule UID

Expected output:
- a confirmation object containing the deleted rule UID, title, and `deleted: true`
