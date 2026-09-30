export const DATADOG_WEBHOOKS_HELP =
  "Recent calls from Datadog. The list shows the event type, service, result, and task ID. It does not show the payload or tokens.";
export const DATADOG_WEBHOOKS_EMPTY = "No Datadog webhooks have arrived.";
export const DATADOG_WEBHOOKS_PAGE_EMPTY = "This page has no Datadog webhooks.";
export const DATADOG_WEBHOOKS_LOAD_ERROR = "SuperPlane could not load Datadog webhooks.";
export const DATADOG_WEBHOOK_PAGE_SIZE = 50;

export type DatadogWebhookReceipt = {
  id: string;
  received_at: string;
  integration_id: string;
  organization_id: string;
  event_type: string;
  alert_transition: string;
  alert_id: string;
  service: string;
  issue_id: string;
  http_status: number;
  outcome: string;
  subscription_count: number;
  task_ids?: string[];
};

export type DatadogWebhooksResponse = {
  items: DatadogWebhookReceipt[];
  total: number;
  page: number;
  limit: number;
};

export function datadogWebhookOutcomeLabel(outcome: string) {
  switch (outcome) {
    case "accepted":
      return "Accepted";
    case "no_subscription":
      return "No subscription";
    case "ignored":
      return "Ignored";
    case "rejected":
      return "Rejected";
    case "failed":
      return "Failed";
    case "pending":
      return "Pending";
    default:
      return outcome || "Unknown";
  }
}
