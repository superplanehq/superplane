# Grafana Create Alert Rule

Use this component when a workflow needs to create a Grafana-managed alert rule from structured alert settings.

Good fits:
- creating baseline monitoring during service onboarding
- provisioning environment-specific alert coverage
- creating temporary alert rules during an incident or rollout

Expected input:
- structured alert rule fields such as `title`, `folderUID`, `ruleGroup`, `dataSourceUid`, `query`, and evaluation settings

Expected output:
- the created Grafana alert rule object with identifiers like `uid`
