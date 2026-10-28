import {
  factoriesCancelFactoryCodexConnection,
  factoriesDescribeFactoryCodexConnection,
  factoriesStartFactoryCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { usePermissions } from "@/contexts/usePermissions";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

export function factoryCodexConnectionKey(organizationId: string, factoryId: string) {
  return ["factory-codex-connection", organizationId, factoryId] as const;
}

export function codexSubscriptionReady(connection: FactoriesFactoryCodexConnection | undefined): boolean {
  return connection?.status === "ready" && connection.reservationHeld !== true;
}

export function useFactoryCodexConnection(organizationId: string, factoryId: string, enabled: boolean) {
  const { canAct } = usePermissions();
  const canRefreshCatalog = canAct("factories", "update");
  const queryClient = useQueryClient();
  const queryKey = factoryCodexConnectionKey(organizationId, factoryId);
  const refreshAttempted = useRef(false);
  const invalidatedCatalog = useRef("");
  const query = useQuery({
    queryKey,
    enabled: enabled && Boolean(organizationId && factoryId),
    queryFn: async ({ signal }) => {
      const response = await factoriesDescribeFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          signal: AbortSignal.any([signal, AbortSignal.timeout(15000)]),
        }),
      );
      return response.data?.connection;
    },
    refetchInterval: (current) => {
      if (current.state.status === "error") return 2000;
      const status = current.state.data?.status;
      return status === "pending" || status === "finishing" || current.state.data?.reservationHeld ? 2000 : false;
    },
    refetchIntervalInBackground: true,
  });

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey });
  };

  const start = useMutation({
    mutationFn: async () => {
      const response = await factoriesStartFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {},
          signal: AbortSignal.timeout(15000),
        }),
      );
      return response.data?.connection;
    },
    onSettled: refresh,
    onError: () => showErrorToast("SuperPlane could not start ChatGPT sign-in. Try again."),
  });

  const refreshCatalog = useMutation({
    mutationFn: async () => {
      const response = await factoriesStartFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: { refreshCatalog: true },
          signal: AbortSignal.timeout(15000),
        }),
      );
      return response.data?.connection;
    },
    onSettled: refresh,
    retry: 2,
    onError: () => showErrorToast("SuperPlane could not load your ChatGPT models. Reload the page to try again."),
  });

  const cancel = useMutation({
    mutationFn: async () => {
      const response = await factoriesCancelFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {},
          signal: AbortSignal.timeout(15000),
        }),
      );
      return response.data?.connection;
    },
    onSettled: refresh,
    onError: () => showErrorToast("SuperPlane could not cancel ChatGPT sign-in. Try again."),
  });

  useEffect(() => {
    if (!enabled || query.data?.status !== "ready") {
      refreshAttempted.current = false;
      return;
    }
    const availableModels = query.data.availableModels ?? [];
    if (availableModels.length > 0) {
      const catalog = availableModels.join("\n");
      if (invalidatedCatalog.current !== catalog) {
        invalidatedCatalog.current = catalog;
        void queryClient.invalidateQueries({ queryKey: ["organizations", organizationId, "selectable-llm-models"] });
      }
      return;
    }
    if (!canRefreshCatalog) return;
    if (!query.data.reservationHeld && !refreshAttempted.current && !refreshCatalog.isPending) {
      refreshAttempted.current = true;
      refreshCatalog.mutate();
    }
  }, [canRefreshCatalog, enabled, organizationId, query.data, queryClient, refreshCatalog]);

  return { query, start, cancel, refreshCatalog };
}

export function useCodexSubscriptionModel(organizationId: string, factoryId: string) {
  const features = useExperimentalFeature(organizationId);
  const enabled = !features.isLoading && features.has(FEATURE_CODEX_SUBSCRIPTIONS);
  const codex = useFactoryCodexConnection(organizationId, factoryId, enabled);
  const connection = codex.query.data;
  return {
    codexSubscriptionOffered: enabled && connection?.eligible === true,
    codexSubscriptionLoading: features.isLoading || (enabled && codex.query.isLoading),
    codexConnection: connection,
    codexSubscriptionReady: codexSubscriptionReady(connection),
    startCodexConnection: () => {
      codex.start.mutate();
    },
    cancelCodexConnection: () => {
      codex.cancel.mutate();
    },
    codexStarting: codex.start.isPending,
    codexCanceling: codex.cancel.isPending,
  };
}
