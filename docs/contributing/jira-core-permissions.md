# Jira core permissions

The Jira integration supports issue actions, workflow reads, transitions, and issue and comment webhooks.
It does not support Jira Service Management approvals, incidents, alerts, or heartbeats.

SuperPlane preserves stored connections and workflow data.
Replace removed Jira Service Management components before you run or publish an affected workflow.
The stored `enableOpsFeatures` setting has no effect.

## OAuth app setup

Add these scopes under **Jira API** in the Atlassian Developer Console:

- `read:jira-work`
- `write:jira-work`
- `manage:jira-webhook`
- `read:jira-user`
- `read:issue-details:jira`

SuperPlane also requests `offline_access` for token refresh.
The Permissions tab does not list this scope.
User access supports identity checks and assignee pickers.

## Deployment follow-up

After deployment, remove unused Service Management scopes from your OAuth app in the Atlassian Developer Console.
Reauthorize existing connections to narrow their grants.
This code change does not alter Atlassian app settings or revoke existing grants.

Remove outgoing alert webhook integrations in Jira Service Management when you no longer need them.
SuperPlane no longer manages these remote registrations.

Site-restricted grants require a separate Atlassian configuration change.
Removing Service Management support does not remove the cross-site consent warning.
