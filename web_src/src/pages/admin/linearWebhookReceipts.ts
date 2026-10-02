export const LINEAR_WEBHOOKS_HELP =
  "Recent calls from Linear. The list shows the event type, issue, result, and task ID. It does not show the payload or tokens.";
export const LINEAR_WEBHOOKS_EMPTY = "No Linear webhooks have arrived.";
export const LINEAR_WEBHOOKS_PAGE_EMPTY = "This page has no Linear webhooks.";
export const LINEAR_WEBHOOKS_LOAD_ERROR = "SuperPlane could not load Linear webhooks.";
export const LINEAR_WEBHOOK_PAGE_SIZE = 50;

export type LinearWebhookReceipt = {
  id: string;
  received_at: string;
  integration_id: string;
  organization_id: string;
  webhook_id: string;
  event_type: string;
  action: string;
  issue_identifier: string;
  issue_id: string;
  team_key: string;
  workspace_key: string;
  http_status: number;
  outcome: string;
  subscription_count: number;
  task_ids?: string[];
};

export type LinearWebhooksResponse = {
  items: LinearWebhookReceipt[];
  total: number;
  page: number;
  limit: number;
};

export function linearWebhookOutcomeLabel(outcome: string) {
  switch (outcome) {
    case "accepted":
      return "Accepted";
    case "ignored":
      return "Ignored";
    case "no_subscription":
      return "No subscription";
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

export function linearWebhookIssueLabel(receipt: LinearWebhookReceipt) {
  return receipt.issue_identifier || receipt.issue_id || "";
}
