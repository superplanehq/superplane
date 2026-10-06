import { useCallback, useEffect, useState } from "react";

import {
  SENTRY_WEBHOOK_PAGE_SIZE,
  SENTRY_WEBHOOKS_LOAD_ERROR,
  type SentryWebhooksResponse,
} from "./sentryWebhookReceipts";

const PROJECT_FILTER_PAUSE_MS = 200;

function sentryWebhooksURL(offset: number, project: string) {
  const page = Math.floor(offset / SENTRY_WEBHOOK_PAGE_SIZE) + 1;
  const params = new URLSearchParams({
    page: String(page),
    limit: String(SENTRY_WEBHOOK_PAGE_SIZE),
  });
  const trimmedProject = project.trim();
  if (trimmedProject !== "") {
    params.set("project", trimmedProject);
  }
  return `/admin/api/sentry/webhooks?${params.toString()}`;
}

export function useSentryWebhooks() {
  const [offset, setOffset] = useState(0);
  const [project, setProjectValue] = useState("");
  const [appliedProject, setAppliedProject] = useState("");
  const [data, setData] = useState<SentryWebhooksResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  const setProject = useCallback((value: string) => {
    setProjectValue(value);
    setOffset(0);
    setData(null);
    setLoadError("");
    setLoading(true);
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => setAppliedProject(project), PROJECT_FILTER_PAUSE_MS);
    return () => window.clearTimeout(timeout);
  }, [project]);

  const load = useCallback(async (nextOffset: number, nextProject: string, signal?: AbortSignal) => {
    setLoading(true);
    setLoadError("");
    const response = await fetch(sentryWebhooksURL(nextOffset, nextProject), {
      credentials: "include",
      signal,
    });
    if (!response.ok) {
      throw new Error(SENTRY_WEBHOOKS_LOAD_ERROR);
    }
    return (await response.json()) as SentryWebhooksResponse;
  }, []);

  useEffect(() => {
    if (project !== appliedProject) {
      return;
    }
    const controller = new AbortController();
    load(offset, appliedProject, controller.signal)
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
  }, [appliedProject, load, offset, project]);

  return { offset, setOffset, project, setProject, data, loading, loadError };
}
