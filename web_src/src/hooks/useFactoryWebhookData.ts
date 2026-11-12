import { factoriesListFactoryWebhooks, factoriesRetryFactoryWebhook } from "@/api-client";
import type { FactoriesFactoryWebhookStatus } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { showErrorToast } from "@/lib/toast";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

const RETRY_POLL_INTERVAL_MS = 2000;
const RETRY_POLL_WINDOW_MS = 60_000;

type RetryProgress = {
  webhook: FactoriesFactoryWebhookStatus;
  deadline: number;
  sawPending: boolean;
  timedOut: boolean;
};

function isRegistrationRunning(webhook: FactoriesFactoryWebhookStatus) {
  return webhook.state === "pending" || webhook.state === "provisioning";
}

export function useFactoryWebhookData(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  const queryKey = ["factories", organizationId, factoryId, "webhooks"];
  const query = useQuery({
    queryKey,
    queryFn: async () => {
      const response = await factoriesListFactoryWebhooks(
        withOrganizationHeader({ organizationId, path: { factoryId } }),
      );
      return response.data?.webhooks ?? [];
    },
    enabled: Boolean(organizationId && factoryId),
  });
  const webhooks = query.data;
  const refetch = query.refetch;
  const [progress, setProgress] = useState<Record<string, RetryProgress>>({});

  useEffect(() => setProgress({}), [organizationId, factoryId]);

  useEffect(() => {
    if (!webhooks) {
      return;
    }
    setProgress((current) => {
      const next = { ...current };
      let changed = false;
      for (const [id, retry] of Object.entries(current)) {
        const webhook = webhooks.find((item) => item.id === id);
        if (!webhook || webhook.state === "ready" || (webhook.state === "failed" && retry.sawPending)) {
          delete next[id];
          changed = true;
        } else if (isRegistrationRunning(webhook) && !retry.sawPending) {
          next[id] = { ...retry, sawPending: true };
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }, [webhooks, query.dataUpdatedAt]);

  const hasSetupInProgress = webhooks?.some(
    (webhook) => isRegistrationRunning(webhook) && !(webhook.id && progress[webhook.id]?.timedOut),
  );

  useEffect(() => {
    if (!hasSetupInProgress && !Object.values(progress).some((item) => !item.timedOut)) {
      return;
    }
    const timer = window.setInterval(() => {
      const now = Date.now();
      setProgress((current) => {
        let changed = false;
        const next = { ...current };
        for (const [id, item] of Object.entries(current)) {
          if (!item.timedOut && now >= item.deadline) {
            next[id] = { ...item, timedOut: true };
            changed = true;
          }
        }
        return changed ? next : current;
      });
      void refetch();
    }, RETRY_POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [hasSetupInProgress, progress, refetch]);

  const warnings = useMemo(() => {
    const listed = webhooks ?? [];
    const byId = new Map<string, FactoriesFactoryWebhookStatus>();
    for (const webhook of listed) {
      if (!webhook.id || (webhook.state !== "failed" && !progress[webhook.id])) {
        continue;
      }
      const retry = progress[webhook.id];
      byId.set(webhook.id, retry ? { ...webhook, message: webhook.message || retry.webhook.message } : webhook);
    }
    for (const [id, retry] of Object.entries(progress)) {
      if (!byId.has(id) && !listed.some((webhook) => webhook.id === id && webhook.state === "ready")) {
        byId.set(id, retry.webhook);
      }
    }
    return [...byId.values()];
  }, [webhooks, progress]);

  const retry = async (webhook: FactoriesFactoryWebhookStatus) => {
    if (!webhook.id || progress[webhook.id]) {
      return;
    }
    const id = webhook.id;
    setProgress((current) => ({
      ...current,
      [id]: { webhook, deadline: Date.now() + RETRY_POLL_WINDOW_MS, sawPending: false, timedOut: false },
    }));
    try {
      const response = await factoriesRetryFactoryWebhook(
        withOrganizationHeader({ organizationId, path: { factoryId, webhookId: id }, body: {} }),
      );
      await queryClient.cancelQueries({ queryKey });
      const updatedWebhook = response.data?.webhook;
      if (updatedWebhook) {
        queryClient.setQueryData<FactoriesFactoryWebhookStatus[]>(queryKey, (current) =>
          current?.map((item) => (item.id === id ? updatedWebhook : item)),
        );
      }
      setProgress((current) => {
        const pending = current[id];
        return pending ? { ...current, [id]: { ...pending, sawPending: true } } : current;
      });
      void refetch();
    } catch {
      setProgress((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
      showErrorToast("SuperPlane could not retry the webhook registration. Try again.");
    }
  };

  return {
    warnings,
    retry,
    retrying: (id: string | undefined) => Boolean(id && progress[id] && !progress[id].timedOut),
    retryTimedOut: (id: string | undefined) => Boolean(id && progress[id]?.timedOut),
    refresh: () => void refetch(),
  };
}
