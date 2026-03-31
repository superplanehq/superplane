# Grafana Create Alert Rule

Use this component when a workflow needs to create a Grafana-managed alert rule from a full provisioning API definition.

Good fits:
- creating baseline monitoring during service onboarding
- provisioning environment-specific alert coverage
- creating temporary alert rules during an incident or rollout

Expected input:
- a complete Grafana alert rule object, including fields such as `title`, `folderUID`, `ruleGroup`, `condition`, and `data`

Expected output:
- the created Grafana alert rule object with identifiers like `uid`
