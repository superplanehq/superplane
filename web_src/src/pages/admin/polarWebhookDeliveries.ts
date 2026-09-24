export const POLAR_WEBHOOKS_TITLE = "Polar Webhooks";
export const POLAR_WEBHOOKS_HELP =
  "One row per Polar event. Expand a row to see delivery attempts. The Failed list hides events Polar already delivered.";
export const POLAR_WEBHOOKS_NOT_CONFIGURED = "Polar is not configured. Set POLAR_ACCESS_TOKEN on the app server.";
export const POLAR_WEBHOOKS_EMPTY = "No Polar webhook events match this filter.";
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

export type PolarWebhookEventGroup = {
  eventId: string;
  eventType: string;
  eventSucceeded: boolean;
  deliveries: PolarWebhookDelivery[];
  latest: PolarWebhookDelivery;
};

export function uniqueFailedEventIds(
  items: PolarWebhookDelivery[],
  pendingEventIds: ReadonlySet<string> = new Set(),
): string[] {
  const ids: string[] = [];
  const seen = new Set<string>();
  for (const item of items) {
    const eventId = item.event_id.trim();
    if (
      item.succeeded ||
      item.event_succeeded === true ||
      eventId === "" ||
      seen.has(eventId) ||
      pendingEventIds.has(eventId)
    ) {
      continue;
    }
    seen.add(eventId);
    ids.push(eventId);
  }
  return ids;
}

export function groupPolarWebhookEvents(items: PolarWebhookDelivery[]): PolarWebhookEventGroup[] {
  const deliveriesByEvent = new Map<string, PolarWebhookDelivery[]>();
  const eventOrder: string[] = [];

  for (const item of items) {
    const eventId = item.event_id.trim() || item.id;
    const existing = deliveriesByEvent.get(eventId);
    if (existing === undefined) {
      eventOrder.push(eventId);
      deliveriesByEvent.set(eventId, [item]);
      continue;
    }
    existing.push(item);
  }

  return eventOrder.flatMap((eventId) => {
    const deliveries = [...(deliveriesByEvent.get(eventId) ?? [])].sort((left, right) =>
      right.created_at.localeCompare(left.created_at),
    );
    const latest = deliveries[0];
    if (!latest) {
      return [];
    }

    return [
      {
        eventId,
        eventType: latest.event_type,
        eventSucceeded: deliveries.some((delivery) => delivery.event_succeeded === true),
        deliveries,
        latest,
      },
    ];
  });
}

export function visiblePolarWebhookEvents(
  groups: PolarWebhookEventGroup[],
  statusFilter: PolarWebhookStatusFilter,
): PolarWebhookEventGroup[] {
  if (statusFilter !== "failed") {
    return groups;
  }
  return groups.filter((group) => !group.eventSucceeded);
}

export function polarWebhookAttemptLabel(count: number): string {
  if (count === 1) {
    return "1 attempt";
  }
  return `${count} attempts`;
}

export function polarWebhookEventStatus(
  item: PolarWebhookDelivery,
  pendingEventIds: ReadonlySet<string>,
): PolarWebhookEventStatus {
  return polarWebhookGroupStatus(
    {
      eventId: item.event_id,
      eventType: item.event_type,
      eventSucceeded: item.event_succeeded === true,
      deliveries: [item],
      latest: item,
    },
    pendingEventIds,
  );
}

export function polarWebhookGroupStatus(
  group: PolarWebhookEventGroup,
  pendingEventIds: ReadonlySet<string>,
): PolarWebhookEventStatus {
  if (group.eventId !== "" && pendingEventIds.has(group.eventId)) {
    return "sending";
  }
  if (group.eventSucceeded) {
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
