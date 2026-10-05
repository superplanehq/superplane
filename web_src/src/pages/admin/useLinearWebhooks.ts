import { useCallback, useEffect, useState } from "react";

import {
  LINEAR_WEBHOOK_PAGE_SIZE,
  LINEAR_WEBHOOKS_LOAD_ERROR,
  type LinearWebhooksResponse,
} from "./linearWebhookReceipts";

export function useLinearWebhooks() {
  const [offset, setOffset] = useState(0);
  const [data, setData] = useState<LinearWebhooksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const load = useCallback(async (nextOffset: number, signal?: AbortSignal) => {
    setLoading(true);
    setLoadError("");
    const page = Math.floor(nextOffset / LINEAR_WEBHOOK_PAGE_SIZE) + 1;
    const response = await fetch(`/admin/api/linear/webhooks?page=${page}&limit=${LINEAR_WEBHOOK_PAGE_SIZE}`, {
      credentials: "include",
      signal,
    });
    if (!response.ok) {
      throw new Error(LINEAR_WEBHOOKS_LOAD_ERROR);
    }
    return (await response.json()) as LinearWebhooksResponse;
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
        setLoadError(error instanceof Error ? error.message : LINEAR_WEBHOOKS_LOAD_ERROR);
        setLoading(false);
      });
    return () => controller.abort();
  }, [load, offset]);

  return { offset, setOffset, data, loading, loadError };
}
