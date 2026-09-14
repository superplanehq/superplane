export const POLAR_WEBHOOKS_TITLE = "Polar Webhooks";
export const POLAR_WEBHOOKS_HELP = "Polar webhook deliveries. Redeliver asks Polar to send the event again.";
export const POLAR_WEBHOOKS_NOT_CONFIGURED = "Polar is not configured. Set POLAR_ACCESS_TOKEN on the app server.";
export const POLAR_WEBHOOKS_EMPTY = "No Polar webhook deliveries match this filter.";
export const POLAR_WEBHOOKS_REDELIVER = "Redeliver";
export const POLAR_WEBHOOKS_REDELIVER_FAILED = "Redeliver failed";
export const POLAR_WEBHOOKS_UNAUTHORIZED =
  "Polar rejected the access token. Add webhooks:read and webhooks:write scopes.";

export const POLAR_WEBHOOK_PAGE_SIZE = 50;
export const POLAR_WEBHOOK_ALL_VALUE = "all";

export const POLAR_WEBHOOK_EVENT_TYPES = [
  "order.paid",
  "order.refunded",
  "subscription.created",
  "subscription.updated",
  "subscription.active",
  "subscription.canceled",
  "subscription.uncanceled",
  "subscription.revoked",
] as const;

export type PolarWebhookStatusFilter = "failed" | "succeeded" | "all";

export type PolarWebhookDelivery = {
  id: string;
  created_at: string;
  succeeded: boolean;
  http_code?: number | null;
  response: string;
  event_type: string;
  event_id: string;
  payload: string;
};

export type PolarWebhooksResponse = {
  configured: boolean;
  items: PolarWebhookDelivery[];
  total: number;
  page: number;
  limit: number;
};

export function uniqueFailedEventIds(items: PolarWebhookDelivery[]): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const item of items) {
    const eventId = item.event_id.trim();
    if (item.succeeded || eventId === "" || seen.has(eventId)) {
      continue;
    }
    seen.add(eventId);
    ids.push(eventId);
  }
  return ids;
}

export function formatPolarPayload(value: string): string {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return value;
  }
}

export async function readPolarAdminError(response: Response, fallback: string): Promise<string> {
  const text = (await response.text()).trim();
  return text || fallback;
}

export function polarWebhookListQuery(page: number, statusFilter: PolarWebhookStatusFilter, eventType: string): string {
  const params = new URLSearchParams();
  params.set("page", String(page));
  params.set("limit", String(POLAR_WEBHOOK_PAGE_SIZE));
  if (statusFilter === "failed") {
    params.set("succeeded", "false");
  } else if (statusFilter === "succeeded") {
    params.set("succeeded", "true");
  }
  if (eventType !== POLAR_WEBHOOK_ALL_VALUE) {
    params.set("event_type", eventType);
  }
  return params.toString();
}
