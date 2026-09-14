import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  knownDeliveryIdsForEvent,
  POLAR_WEBHOOK_ALL_VALUE,
  POLAR_WEBHOOK_POLL_INTERVAL_MS,
  polarWebhookListQuery,
  prunePendingPolarRedelivers,
  readPolarAdminError,
  uniqueFailedEventIds,
  type PendingPolarRedeliver,
  type PolarWebhookDelivery,
  type PolarWebhooksResponse,
  type PolarWebhookStatusFilter,
} from "./polarWebhookDeliveries";

async function redeliverPolarEvent(eventId: string) {
  const response = await fetch(`/admin/api/polar/webhooks/${encodeURIComponent(eventId)}/redeliver`, {
    method: "POST",
    credentials: "include",
  });
  if (!response.ok) {
    throw new Error(await readPolarAdminError(response, "SuperPlane could not ask Polar to send the event again."));
  }
}

async function redeliverFailedPolarEvents(eventIds: string[]) {
  const accepted: string[] = [];
  const failed: string[] = [];
  for (const eventId of eventIds) {
    try {
      await redeliverPolarEvent(eventId);
      accepted.push(eventId);
    } catch {
      failed.push(eventId);
    }
  }
  return { accepted, failed };
}

function reportRedeliverCounts(accepted: number, failed: number) {
  if (accepted > 0) {
    showSuccessToast(accepted === 1 ? "Polar will send 1 event again." : `Polar will send ${accepted} events again.`);
  }
  if (failed > 0) {
    showErrorToast(
      failed === 1 ? "SuperPlane could not redeliver 1 event." : `SuperPlane could not redeliver ${failed} events.`,
    );
  }
}

function addPendingRedelivers(
  current: Map<string, PendingPolarRedeliver>,
  eventIds: string[],
  items: PolarWebhookDelivery[],
  startedAt: number,
): Map<string, PendingPolarRedeliver> {
  const next = new Map(current);
  for (const eventId of eventIds) {
    next.set(eventId, {
      startedAt,
      knownDeliveryIds: knownDeliveryIdsForEvent(items, eventId),
    });
  }
  return next;
}

function removePendingEventIds(
  current: Map<string, PendingPolarRedeliver>,
  eventIds: string[],
): Map<string, PendingPolarRedeliver> {
  const next = new Map(current);
  for (const eventId of eventIds) {
    next.delete(eventId);
  }
  return next;
}

async function fetchPolarWebhooksPage(
  page: number,
  statusFilter: PolarWebhookStatusFilter,
  eventType: string,
  signal: AbortSignal,
): Promise<PolarWebhooksResponse> {
  const response = await fetch(`/admin/api/polar/webhooks?${polarWebhookListQuery(page, statusFilter, eventType)}`, {
    credentials: "include",
    signal,
  });
  if (!response.ok) {
    throw new Error(await readPolarAdminError(response, "SuperPlane could not load Polar webhook deliveries."));
  }
  return response.json();
}

function usePolarWebhookList() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [items, setItems] = useState<PolarWebhookDelivery[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<PolarWebhookStatusFilter>("failed");
  const [eventType, setEventType] = useState(POLAR_WEBHOOK_ALL_VALUE);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingRedelivers, setPendingRedelivers] = useState<Map<string, PendingPolarRedeliver>>(new Map());
  const loadAbort = useRef<AbortController | null>(null);
  const loadGeneration = useRef(0);

  const loadDeliveries = useCallback(
    async (showLoading: boolean) => {
      loadAbort.current?.abort();
      const controller = new AbortController();
      loadAbort.current = controller;
      const generation = ++loadGeneration.current;
      const isCurrentLoad = () => generation === loadGeneration.current;
      if (showLoading) {
        setLoading(true);
      }

      try {
        const data = await fetchPolarWebhooksPage(page, statusFilter, eventType, controller.signal);
        if (!isCurrentLoad()) {
          return;
        }
        const nextItems = data.items ?? [];
        setConfigured(data.configured);
        setItems(nextItems);
        setTotal(data.total ?? 0);
        setLoadError(null);
        setPendingRedelivers((current) => prunePendingPolarRedelivers(current, nextItems, Date.now()));
      } catch (error) {
        if (controller.signal.aborted || !isCurrentLoad()) {
          return;
        }
        const message = error instanceof Error ? error.message : "SuperPlane could not load Polar webhook deliveries.";
        setLoadError(message);
        showErrorToast(message);
      } finally {
        if (isCurrentLoad()) {
          setLoading(false);
        }
      }
    },
    [eventType, page, statusFilter],
  );

  useEffect(() => {
    void loadDeliveries(true);
    const abortRef = loadAbort;
    return () => {
      abortRef.current?.abort();
    };
  }, [loadDeliveries]);

  useEffect(() => {
    if (configured !== true) {
      return;
    }
    const intervalId = window.setInterval(() => {
      void loadDeliveries(false);
    }, POLAR_WEBHOOK_POLL_INTERVAL_MS);
    return () => {
      window.clearInterval(intervalId);
    };
  }, [configured, loadDeliveries]);

  return {
    configured,
    items,
    total,
    page,
    setPage,
    statusFilter,
    setStatusFilter,
    eventType,
    setEventType,
    loading,
    loadError,
    pendingRedelivers,
    setPendingRedelivers,
    loadDeliveries,
  };
}

export function usePolarWebhooks() {
  const list = usePolarWebhookList();
  const [bulkBusy, setBulkBusy] = useState(false);
  const failedEventIds = useMemo(() => uniqueFailedEventIds(list.items), [list.items]);
  const redelivering = useMemo(() => new Set(list.pendingRedelivers.keys()), [list.pendingRedelivers]);

  const handleRedeliver = async (eventId: string) => {
    list.setPendingRedelivers((current) => addPendingRedelivers(current, [eventId], list.items, Date.now()));
    try {
      await redeliverPolarEvent(eventId);
      showSuccessToast("Polar will send the event again.");
      await list.loadDeliveries(false);
    } catch (error) {
      list.setPendingRedelivers((current) => removePendingEventIds(current, [eventId]));
      showErrorToast(
        error instanceof Error ? error.message : "SuperPlane could not ask Polar to send the event again.",
      );
    }
  };

  const handleRedeliverFailed = async () => {
    if (failedEventIds.length === 0) {
      return;
    }
    list.setPendingRedelivers((current) => addPendingRedelivers(current, failedEventIds, list.items, Date.now()));
    setBulkBusy(true);
    try {
      const result = await redeliverFailedPolarEvents(failedEventIds);
      list.setPendingRedelivers((current) => removePendingEventIds(current, result.failed));
      reportRedeliverCounts(result.accepted.length, result.failed.length);
      await list.loadDeliveries(false);
    } finally {
      setBulkBusy(false);
    }
  };

  return {
    configured: list.configured,
    items: list.items,
    total: list.total,
    page: list.page,
    setPage: list.setPage,
    statusFilter: list.statusFilter,
    changeStatusFilter: (value: PolarWebhookStatusFilter) => {
      list.setPage(1);
      list.setStatusFilter(value);
    },
    eventType: list.eventType,
    changeEventType: (value: string) => {
      list.setPage(1);
      list.setEventType(value);
    },
    loading: list.loading,
    loadError: list.loadError,
    redelivering,
    bulkBusy,
    failedEventIds,
    handleRedeliver,
    handleRedeliverFailed,
  };
}
