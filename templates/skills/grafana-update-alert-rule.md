# Grafana Update Alert Rule

Use this component when a workflow needs to update an existing Grafana-managed alert rule.

Good fits:
- adjusting thresholds after an incident review
- updating labels and annotations used for routing
- tuning alert behavior during rollouts or environment changes

Expected input:
- a Grafana alert rule UID
- one or more structured alert rule fields to change, such as `title`, `folderUID`, `ruleGroup`, `dataSourceUid`, `query`, or labels

Expected output:
- the updated Grafana alert rule object returned by the provisioning API
