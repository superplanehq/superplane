# Grafana Get Alert Rule

Use this component when a workflow needs the current source of truth for a Grafana-managed alert rule.

Good fits:
- reviewing a rule before updating it
- enriching Slack, Jira, or PagerDuty steps with alert rule context
- comparing the current rule definition with an expected configuration

Expected input:
- a Grafana alert rule UID

Expected output:
- the full Grafana alert rule object returned by the provisioning API
