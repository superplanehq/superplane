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

export function usePolarWebhooks() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [items, setItems] = useState<PolarWebhookDelivery[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<PolarWebhookStatusFilter>("failed");
  const [eventType, setEventType] = useState(POLAR_WEBHOOK_ALL_VALUE);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingRedelivers, setPendingRedelivers] = useState<Map<string, PendingPolarRedeliver>>(new Map());
  const [bulkBusy, setBulkBusy] = useState(false);
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
        const response = await fetch(
          `/admin/api/polar/webhooks?${polarWebhookListQuery(page, statusFilter, eventType)}`,
          { credentials: "include", signal: controller.signal },
        );
        if (!response.ok) {
          throw new Error(await readPolarAdminError(response, "SuperPlane could not load Polar webhook deliveries."));
        }

        const data: PolarWebhooksResponse = await response.json();
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

  const failedEventIds = useMemo(() => uniqueFailedEventIds(items), [items]);
  const redelivering = useMemo(() => new Set(pendingRedelivers.keys()), [pendingRedelivers]);

  const handleRedeliver = async (eventId: string) => {
    setPendingRedelivers((current) => addPendingRedelivers(current, [eventId], items, Date.now()));
    try {
      await redeliverPolarEvent(eventId);
      showSuccessToast("Polar will send the event again.");
      await loadDeliveries(false);
    } catch (error) {
      setPendingRedelivers((current) => {
        const next = new Map(current);
        next.delete(eventId);
        return next;
      });
      showErrorToast(
        error instanceof Error ? error.message : "SuperPlane could not ask Polar to send the event again.",
      );
    }
  };

  const handleRedeliverFailed = async () => {
    if (failedEventIds.length === 0) {
      return;
    }

    setPendingRedelivers((current) => addPendingRedelivers(current, failedEventIds, items, Date.now()));
    setBulkBusy(true);
    try {
      const result = await redeliverFailedPolarEvents(failedEventIds);
      if (result.failed.length > 0) {
        setPendingRedelivers((current) => {
          const next = new Map(current);
          for (const eventId of result.failed) {
            next.delete(eventId);
          }
          return next;
        });
      }
      reportRedeliverCounts(result.accepted.length, result.failed.length);
      await loadDeliveries(false);
    } finally {
      setBulkBusy(false);
    }
  };

  return {
    configured,
    items,
    total,
    page,
    setPage,
    statusFilter,
    changeStatusFilter: (value: PolarWebhookStatusFilter) => {
      setPage(1);
      setStatusFilter(value);
    },
    eventType,
    changeEventType: (value: string) => {
      setPage(1);
      setEventType(value);
    },
    loading,
    loadError,
    redelivering,
    bulkBusy,
    failedEventIds,
    handleRedeliver,
    handleRedeliverFailed,
  };
}
