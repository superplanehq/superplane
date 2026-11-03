export const SENTRY_WEBHOOKS_HELP =
  "Recent calls from Sentry. The list shows the event type, project, result, and task ID. HTTP status is the answer that Sentry received. Result is the processing state. The list does not show the payload or tokens.";
export const SENTRY_WEBHOOKS_EMPTY = "No Sentry webhooks have arrived.";
export const SENTRY_WEBHOOKS_PROJECT_EMPTY = "No Sentry webhooks match this project.";
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
  processing_error?: string;
  integration_count: number;
  task_ids?: string[];
};

export type SentryWebhooksResponse = {
  items: SentryWebhookReceipt[];
  total: number;
  page: number;
  limit: number;
};

// Accepted, No connection, Rejected, and Failed come from the synchronous
// delivery path. The other outcomes come from the durable inbox.
const SENTRY_WEBHOOK_OUTCOME_LABELS = new Map<string, string>([
  ["accepted", "Accepted"],
  ["no_connection", "No connection"],
  ["rejected", "Rejected"],
  ["failed", "Failed"],
  ["queued", "Queued"],
  ["processing", "Processing"],
  ["retrying", "Retrying"],
  ["blocked", "Blocked"],
  ["processed", "Processed"],
  ["filtered", "Filtered"],
]);

export function sentryWebhookOutcomeLabel(outcome: string) {
  return SENTRY_WEBHOOK_OUTCOME_LABELS.get(outcome) ?? (outcome || "Unknown");
}

export function sentryWebhookIssueLabel(receipt: SentryWebhookReceipt) {
  return receipt.issue_short_id || receipt.issue_id || "";
}
