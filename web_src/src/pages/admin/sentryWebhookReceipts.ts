export const SENTRY_WEBHOOKS_TITLE = "Sentry Webhooks";
export const SENTRY_WEBHOOKS_HELP =
  "Recent calls from Sentry. The list shows the event type, project, result, and task ID. It does not show the payload or tokens.";
export const SENTRY_WEBHOOKS_EMPTY = "No Sentry webhooks have arrived.";
export const SENTRY_WEBHOOKS_PAGE_EMPTY = "This page has no Sentry webhooks.";
export const SENTRY_WEBHOOKS_LOAD_ERROR = "SuperPlane could not load Sentry webhooks.";
export const SENTRY_WEBHOOK_PAGE_SIZE = 50;

export type SentryWebhookReceipt = {
  id: string;
  received_at: string;
  hook_resource: string;
  action: string;
  installation_uuid: string;
  organization_slug: string;
  project_slug: string;
  issue_id: string;
  issue_short_id: string;
  http_status: number;
  outcome: string;
  integration_count: number;
  task_ids?: string[];
};

export type SentryWebhooksResponse = {
  items: SentryWebhookReceipt[];
  total: number;
  page: number;
  limit: number;
};

export function sentryWebhookOutcomeLabel(outcome: string) {
  switch (outcome) {
    case "accepted":
      return "Accepted";
    case "no_connection":
      return "No connection";
    case "rejected":
      return "Rejected";
    case "failed":
      return "Failed";
    default:
      return outcome || "Unknown";
  }
}

export function sentryWebhookIssueLabel(receipt: SentryWebhookReceipt) {
  return receipt.issue_short_id || receipt.issue_id || "";
}
