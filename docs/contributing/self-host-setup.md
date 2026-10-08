# Self-host setup

This runbook is for a new SuperPlane installation. Cloud already holds the
public GitHub App. Do not set `SUPERPLANE_GITHUB_APP_*` in Helm for
self-host. Workspace Connect creates a private GitHub App once.

## 1. Create the owner account

Open SuperPlane and complete owner setup. Choose Community or add an
Enterprise license. You can add a license later.

## 2. Configure email (optional)

SMTP sends invites and magic codes. The owner password login works
without SMTP. Skip this step if you do not need email yet. You can add
SMTP later in installation settings.

## 3. Configure Fleet Manager (optional)

SuperPlane does not call Fleet Manager. Owner setup creates the
`e1-large-amd64` fleet and shows YAML. Copy the YAML into the Fleet
Manager configuration, then apply it.

Replace the placeholder values with the values from your Terraform
ConfigMap. Skip this step if you will add runners later.

## 4. Create the workspace and the GitHub App

Owner setup then opens workspace setup.

1. Create the workspace. SuperPlane does not block this step when the
   GitHub App is missing.
2. On Connect, choose **Create GitHub App** if this installation has no
   GitHub App.
3. GitHub creates a private app and returns to Connect.
4. Choose **Connect GitHub**. Install the app on a GitHub organization.
5. Choose the source repository. SuperPlane does not ask for a second
   repository on an installation wizard.

GitHub login OAuth, Linear, Jira, and Sentry hosted apps are not part of
this flow.

## 5. Confirm the installation

- The workspace wizard continues to tickets and the agent key.
- GitHub webhooks reach `{WEBHOOKS_BASE_URL}/api/v1/github/app/webhook`.
- Fleet Manager polls SuperPlane with `superplaneUrl` and the token from
  the YAML.
