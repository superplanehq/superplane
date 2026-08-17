# Factory Notifications

## Overview

Work order activity in a workspace (factory) is only visible in the work order
timeline. Owners and creators miss comments, ownership changes, and status
changes unless they open the work order page.

This document describes per-user email notifications for factory work order
activity. Each user configures their own notification settings from a page in
workspace settings. SuperPlane sends the emails through the existing platform
email service (Resend in cloud deployments, SMTP in owner-setup deployments).

## Goals

1. Notify users by email when relevant work order activity occurs.
2. Let each user control which notification types they receive.
3. Let each user scope notifications to all workspaces or to a selected list
   of workspaces in the organization.
4. Reuse the existing email delivery stack (`EmailService`, RabbitMQ consumer
   pattern, email templates).

## Non-Goals

- **@mentions**: Comments do not support mentions today. The
  `work_order_mention` notification type is reserved for a future release.
- **In-app inbox**: No bell icon or unread list. Email is the only channel in
  this release.
- **Digests**: Each event sends one immediate email. Batching and daily
  digests are future work.
- **Per-work-order watch/unwatch**: Subscription is implicit through
  ownership and creation.
- **Slack or other channels**: Email only.

## Settings Model

Settings are stored per user, per organization. A user edits them from
`/:organizationId/workspaces/:factoryKey/settings/notifications`. The page
edits the user's organization-wide settings; the workspace in the URL only
provides the navigation context.

- **Master switch**: Notifications are off until the user turns them on.
  Without a settings row, SuperPlane sends no notification emails.
- **Workspace scope**: "All workspaces" or "Selected workspaces" (a list of
  factory IDs).
- **Notification types** (each one is a toggle; all are on by default when
  the user enables notifications):

| Type | Trigger | Recipients |
| --- | --- | --- |
| `work_order_assigned` | A user becomes an owner of a work order | The newly assigned users |
| `work_order_comment_owned` | New comment on a work order | The work order owners |
| `work_order_comment_created` | New comment on a work order | The work order creator |
| `work_order_status_owned` | A work order changes state or result | Owners and creator |
| `work_order_artifact_owned` | An artifact is attached to a work order | Owners |
| `work_order_mention` | Reserved, not shipped | — |

## Delivery Rules

- One email per event, sent immediately. No batching.
- The actor never receives an email about their own action.
- Automation actors (canvas runs) have no user to exclude, so all matching
  recipients get the email.
- The initial status transition into `draft` (work order creation) does not
  send a status email.
- Recipients resolve at delivery time from the work order's owners
  (assignees) and creator, then filter through each candidate's settings:
  master switch, type toggle, and workspace scope.

## Architecture

```
gRPC action / canvas automation records FactoryWorkOrderEvent
  → publishes FactoryWorkOrderNotificationMessage (RabbitMQ, after commit)
  → FactoryNotificationConsumer (START_CONSUMERS=yes)
      → load work order, owners, creator, factory
      → resolve candidate recipients per notification type
      → drop the actor
      → filter by user_notification_settings
      → EmailService.SendWorkOrderNotificationEmail per recipient
  → templates/email/work_order_notification.{html,txt}
```

The message carries the notification payload (event type, actor, assigned
user IDs, comment body, state transition) so the consumer does not race with
later timeline writes. Publishers emit the message after the database
transaction commits, next to the existing websocket fan-out message.

### Data Model

Table `user_notification_settings`:

| Column | Type | Notes |
| --- | --- | --- |
| `id` | uuid | Primary key |
| `organization_id` | uuid | Unique together with `user_id` |
| `user_id` | uuid | |
| `enabled` | boolean | Master switch |
| `workspace_scope` | varchar | `all` or `selected` |
| `factory_ids` | jsonb | Factory IDs when scope is `selected` |
| `types` | jsonb | Map of type name to boolean; a missing key means on |

### API

Two RPCs on the `Factories` service, self-scoped to the calling user:

- `DescribeNotificationSettings` — `GET /api/v1/factory-notification-settings`
- `UpdateNotificationSettings` — `PUT /api/v1/factory-notification-settings`

The path sits outside `/api/v1/factories/...` so the REST gateway does not
match it against the `{id}` wildcard routes.

Both require organization membership (`factories` / `read`) and the
`FEATURE_FACTORIES` experimental feature. No extra permission is needed
because a user can only read and write their own settings.

### Email Content

One data-driven template pair (`work_order_notification.html` / `.txt`):

- Subject: `[<KEY>-<number>] <event summary>` (for example
  `[SP-42] New comment from Ana Souza`).
- Body: work order key and title, actor, event summary, comment excerpt when
  present, and a link to
  `/:organizationId/workspaces/:factoryKey/work-order/:number`.

## Future Work

- @mentions in comment bodies and the `work_order_mention` type.
- In-app notification inbox.
- Digest and batching options.
- Per-work-order watch/unwatch.
- Additional channels (Slack).
