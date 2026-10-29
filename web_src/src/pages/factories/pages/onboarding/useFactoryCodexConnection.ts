import {
  factoriesCancelFactoryCodexConnection,
  factoriesDisconnectFactoryCodexConnection,
  factoriesDescribeFactoryCodexConnection,
  factoriesStartFactoryCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { usePermissions } from "@/contexts/usePermissions";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { showErrorToast } from "@/lib/toast";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";

export function factoryCodexConnectionKey(organizationId: string, factoryId: string, memberId = "") {
  return ["factory-codex-connection", organizationId, factoryId, memberId] as const;
}

export function codexSubscriptionReady(connection: FactoriesFactoryCodexConnection | undefined): boolean {
  return connection?.status === "ready" && connection.reservationHeld !== true;
}

function invalidateSubscriptionCatalogs(queryClient: QueryClient, organizationId: string, factoryId: string) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "owner-eligibility"] }),
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId, factoryId, "line-runner-models"] }),
    queryClient.invalidateQueries({ queryKey: ["organizations", organizationId, "selectable-llm-models"] }),
  ]);
}

export function useFactoryCodexConnection(organizationId: string, factoryId: string, enabled: boolean) {
  const { canAct, currentUserId } = usePermissions();
  const canRefreshCatalog = canAct("factories", "read");
  const queryClient = useQueryClient();
  const queryKey = factoryCodexConnectionKey(organizationId, factoryId, currentUserId);
  const refreshAttempted = useRef(false);
  const invalidatedCatalog = useRef("");
  useEffect(() => {
    refreshAttempted.current = false;
    invalidatedCatalog.current = "";
  }, [organizationId, factoryId, currentUserId]);
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
      return current.state.data?.revocationPending ||
        ["pending", "canceling", "finishing"].includes(status ?? "") ||
        current.state.data?.reservationHeld
        ? 2000
        : false;
    },
    refetchIntervalInBackground: true,
  });

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey });
    await invalidateSubscriptionCatalogs(queryClient, organizationId, factoryId);
    await queryClient.invalidateQueries({ queryKey: ["me", organizationId] });
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
    onSuccess: async (connection) => {
      if (!connection) return;
      await queryClient.cancelQueries({ queryKey });
      queryClient.setQueryData(queryKey, connection);
    },
    onSettled: refresh,
    onError: () => showErrorToast("SuperPlane could not cancel ChatGPT sign-in. Try again."),
  });

  const disconnect = useMutation({
    mutationFn: async () =>
      factoriesDisconnectFactoryCodexConnection(withOrganizationHeader({ organizationId, path: { factoryId } })),
    onSettled: refresh,
    onError: () => showErrorToast("Stop your tasks before you disconnect ChatGPT. Then try again."),
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
        void invalidateSubscriptionCatalogs(queryClient, organizationId, factoryId);
      }
      return;
    }
    if (!canRefreshCatalog) return;
    if (!query.data.reservationHeld && !refreshAttempted.current && !refreshCatalog.isPending) {
      refreshAttempted.current = true;
      refreshCatalog.mutate();
    }
  }, [canRefreshCatalog, enabled, organizationId, factoryId, query.data, queryClient, refreshCatalog]);

  return { query, start, cancel, refreshCatalog, disconnect };
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
