import { useCallback, useEffect, useState } from "react";

import {
  DATADOG_WEBHOOK_PAGE_SIZE,
  DATADOG_WEBHOOKS_LOAD_ERROR,
  type DatadogWebhooksResponse,
} from "./datadogWebhookReceipts";

export function useDatadogWebhooks() {
  const [offset, setOffset] = useState(0);
  const [data, setData] = useState<DatadogWebhooksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const load = useCallback(async (nextOffset: number, signal?: AbortSignal) => {
    setLoading(true);
    setLoadError("");
    const page = Math.floor(nextOffset / DATADOG_WEBHOOK_PAGE_SIZE) + 1;
    const response = await fetch(`/admin/api/datadog/webhooks?page=${page}&limit=${DATADOG_WEBHOOK_PAGE_SIZE}`, {
      credentials: "include",
      signal,
    });
    if (!response.ok) {
      throw new Error(DATADOG_WEBHOOKS_LOAD_ERROR);
    }
    return (await response.json()) as DatadogWebhooksResponse;
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    load(offset, controller.signal)
      .then((next) => {
        if (!controller.signal.aborted) {
          setData(next);
          setLoading(false);
        }
      })
      .catch((error: unknown) => {
        if (controller.signal.aborted) {
          return;
        }
        setLoadError(error instanceof Error ? error.message : DATADOG_WEBHOOKS_LOAD_ERROR);
        setLoading(false);
      });
    return () => controller.abort();
  }, [load, offset]);

  return { offset, setOffset, data, loading, loadError };
}
