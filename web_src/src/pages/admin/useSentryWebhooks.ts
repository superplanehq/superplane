import { useCallback, useEffect, useState } from "react";

import {
  SENTRY_WEBHOOK_PAGE_SIZE,
  SENTRY_WEBHOOKS_LOAD_ERROR,
  type SentryWebhooksResponse,
} from "./sentryWebhookReceipts";

export function useSentryWebhooks() {
  const [offset, setOffset] = useState(0);
  const [data, setData] = useState<SentryWebhooksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const load = useCallback(async (nextOffset: number, signal?: AbortSignal) => {
    setLoading(true);
    setLoadError("");
    const page = Math.floor(nextOffset / SENTRY_WEBHOOK_PAGE_SIZE) + 1;
    const response = await fetch(`/admin/api/sentry/webhooks?page=${page}&limit=${SENTRY_WEBHOOK_PAGE_SIZE}`, {
      credentials: "include",
      signal,
    });
    if (!response.ok) {
      throw new Error(SENTRY_WEBHOOKS_LOAD_ERROR);
    }
    return (await response.json()) as SentryWebhooksResponse;
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
        setLoadError(error instanceof Error ? error.message : SENTRY_WEBHOOKS_LOAD_ERROR);
        setLoading(false);
      });
    return () => controller.abort();
  }, [load, offset]);

  return { offset, setOffset, data, loading, loadError };
}
