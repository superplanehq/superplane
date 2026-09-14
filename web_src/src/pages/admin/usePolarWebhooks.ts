import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  POLAR_WEBHOOK_ALL_VALUE,
  polarWebhookListQuery,
  readPolarAdminError,
  uniqueFailedEventIds,
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
  let accepted = 0;
  let failed = 0;
  for (const eventId of eventIds) {
    try {
      await redeliverPolarEvent(eventId);
      accepted += 1;
    } catch {
      failed += 1;
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

export function usePolarWebhooks() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [items, setItems] = useState<PolarWebhookDelivery[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<PolarWebhookStatusFilter>("failed");
  const [eventType, setEventType] = useState(POLAR_WEBHOOK_ALL_VALUE);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [redelivering, setRedelivering] = useState<Set<string>>(new Set());
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

        setConfigured(data.configured);
        setItems(data.items ?? []);
        setTotal(data.total ?? 0);
        setLoadError(null);
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

  const failedEventIds = useMemo(() => uniqueFailedEventIds(items), [items]);

  const handleRedeliver = async (eventId: string) => {
    setRedelivering((current) => new Set(current).add(eventId));
    try {
      await redeliverPolarEvent(eventId);
      showSuccessToast("Polar will send the event again.");
      await loadDeliveries(false);
    } catch (error) {
      showErrorToast(
        error instanceof Error ? error.message : "SuperPlane could not ask Polar to send the event again.",
      );
    } finally {
      setRedelivering((current) => {
        const next = new Set(current);
        next.delete(eventId);
        return next;
      });
    }
  };

  const handleRedeliverFailed = async () => {
    if (failedEventIds.length === 0) {
      return;
    }

    setBulkBusy(true);
    try {
      const result = await redeliverFailedPolarEvents(failedEventIds);
      reportRedeliverCounts(result.accepted, result.failed);
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
