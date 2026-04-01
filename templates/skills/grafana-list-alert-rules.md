# Grafana List Alert Rules

Use this component when a workflow needs an inventory of Grafana-managed alert rules.

Good fits:
- reviewing existing alert coverage in Grafana
- sending alert rule summaries to Slack, Jira, or documentation steps
- feeding downstream workflows that need to inspect available rule UIDs and titles

Expected input:
- no configuration

Expected output:
- an object containing an `alertRules` array with Grafana alert rule summaries
