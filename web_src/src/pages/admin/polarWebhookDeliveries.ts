export const POLAR_WEBHOOKS_TITLE = "Polar Webhooks";
export const POLAR_WEBHOOKS_HELP =
  "Result is this attempt. Event status is the latest Polar result for the same event.";
export const POLAR_WEBHOOKS_NOT_CONFIGURED = "Polar is not configured. Set POLAR_ACCESS_TOKEN on the app server.";
export const POLAR_WEBHOOKS_EMPTY = "No Polar webhook deliveries match this filter.";
export const POLAR_WEBHOOKS_REDELIVER = "Redeliver";
export const POLAR_WEBHOOKS_REDELIVER_FAILED = "Redeliver failed";
export const POLAR_WEBHOOKS_SENDING_AGAIN = "Sending again";
export const POLAR_WEBHOOKS_UNAUTHORIZED =
  "Polar rejected the access token. Add webhooks:read and webhooks:write scopes.";

export const POLAR_WEBHOOK_PAGE_SIZE = 50;
export const POLAR_WEBHOOK_ALL_VALUE = "all";
export const POLAR_WEBHOOK_POLL_INTERVAL_MS = 5000;
export const POLAR_WEBHOOK_REDELIVER_TIMEOUT_MS = 2 * 60 * 1000;

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

export type PolarWebhookEventStatus = "sending" | "succeeded" | "failed";

export type PolarWebhookDelivery = {
  id: string;
  created_at: string;
  succeeded: boolean;
  http_code?: number | null;
  response: string;
  event_type: string;
  event_id: string;
  event_succeeded?: boolean | null;
  payload: string;
};

export type PolarWebhooksResponse = {
  configured: boolean;
  items: PolarWebhookDelivery[];
  total: number;
  page: number;
  limit: number;
};

export type PendingPolarRedeliver = {
  startedAt: number;
  knownDeliveryIds: readonly string[];
};

export function uniqueFailedEventIds(items: PolarWebhookDelivery[]): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const item of items) {
    const eventId = item.event_id.trim();
    if (item.succeeded || item.event_succeeded === true || eventId === "" || seen.has(eventId)) {
      continue;
    }
    seen.add(eventId);
    ids.push(eventId);
  }
  return ids;
}

export function polarWebhookEventStatus(
  item: PolarWebhookDelivery,
  pendingEventIds: ReadonlySet<string>,
): PolarWebhookEventStatus {
  if (item.event_id !== "" && pendingEventIds.has(item.event_id)) {
    return "sending";
  }
  if (item.event_succeeded === true) {
    return "succeeded";
  }
  return "failed";
}

export function polarWebhookEventStatusLabel(status: PolarWebhookEventStatus): string {
  if (status === "sending") {
    return POLAR_WEBHOOKS_SENDING_AGAIN;
  }
  if (status === "succeeded") {
    return "Succeeded";
  }
  return "Failed";
}

export function knownDeliveryIdsForEvent(items: PolarWebhookDelivery[], eventId: string): string[] {
  return items.filter((item) => item.event_id === eventId).map((item) => item.id);
}

export function shouldClearPendingPolarRedeliver(
  eventId: string,
  pending: PendingPolarRedeliver,
  items: PolarWebhookDelivery[],
  now: number,
  timeoutMs = POLAR_WEBHOOK_REDELIVER_TIMEOUT_MS,
): boolean {
  if (now - pending.startedAt >= timeoutMs) {
    return true;
  }

  const matching = items.filter((item) => item.event_id === eventId);
  if (matching.some((item) => item.event_succeeded === true)) {
    return true;
  }

  const knownIds = new Set(pending.knownDeliveryIds);
  return matching.some((item) => !knownIds.has(item.id));
}

export function prunePendingPolarRedelivers(
  pending: ReadonlyMap<string, PendingPolarRedeliver>,
  items: PolarWebhookDelivery[],
  now: number,
  timeoutMs = POLAR_WEBHOOK_REDELIVER_TIMEOUT_MS,
): Map<string, PendingPolarRedeliver> {
  const next = new Map<string, PendingPolarRedeliver>();
  for (const [eventId, entry] of pending) {
    if (!shouldClearPendingPolarRedeliver(eventId, entry, items, now, timeoutMs)) {
      next.set(eventId, entry);
    }
  }
  return next;
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
