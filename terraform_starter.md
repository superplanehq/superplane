# Getting started with Terraform Integration

This is an unofficial starter guide to integrating Terraform into SuperPlane.

## Overview

SuperPlane's Terraform integration leverages Hashicorp's official `go-tfe` SDK for making authenticated API calls, and provides a robust Webhook Handler for securely parsing, validating, and subscribing to workspace events.

### Features

The integration comes out of the box with the following nodes:

- **Triggers**:
  - `Terraform Run Event` - Fires when a run's status changes (applied, planned, errored, etc)
  - `Needs Attention` - Dedicated trigger for runs requiring manual approval or policy overrides.
- **Actions/Components**:
  - `Queue Run` - Start a new Terraform run in a workspace.
  - `Apply Run` - Apply a planned run.
  - `Discard Run` - Cancel or discard a run.
  - `Override Policy` - Override a failed Sentinel policy block.
  - `Read Run Details` - Extract detailed information about a specific run.

## Prerequisites

1. **Terraform API Token**: You will need a User, Team, or Organization API token from Terraform Cloud/Enterprise.
   - **User Token**: Go to **User Settings > Tokens** and click **Create an API token**.
   - **Team Token**: Go to **Organization Settings > Teams**, select your team, and generate a token under **Team API Token**.
   - **Organization Token**: Go to **Organization Settings > API Tokens** and click **Create an organization token**.
2. **Workspace ID**: Required for webhook setup and running operations.
   - Go to your specific workspace in HCP Terraform.
   - Navigate to **Settings > General**.
   - You will find the **Workspace ID** near the top of the settings page (it typically starts with `ws-`).

### Configuring SuperPlane and Automated Webhooks

You do **not** need to manually create webhooks in Terraform Cloud. SuperPlane handles this automatically!

Once you have your token and workspace ID:

1. Navigate to the **Integrations** page in SuperPlane.
2. Find the **Terraform** integration and click on **Connect** or **Configure**.
3. In the configuration modal:
   - Paste your **API Token** into the "API Token" field.
   - Enter your **Workspace ID** into the corresponding field.
   - (Optional) Provide a Webhook Secret to secure incoming events to SuperPlane, or one will be automatically generated.
4. Save the configuration.
5. Next, create a new Workflow in SuperPlane and drag a **Terraform Trigger** (like *Terraform Run Event*) onto your canvas.
6. When you activate your workflow, SuperPlane will securely communicate with the Terraform API and **automatically provision the webhook** in your specified workspace.

## Under the Hood

Each time the integration receives an event, it will:

1. Verify the payload using **HMAC SHA-512** matching your configured Webhook Secret.
2. Normalize the payload into a `RunEventData` structure.
3. Emit events via the SuperPlane Event Bus (`ctx.Events.Emit()`).

The backend handles the initial webhook creation protocol automatically when `Setup()` is invoked for any trigger pointing to a workspace.
