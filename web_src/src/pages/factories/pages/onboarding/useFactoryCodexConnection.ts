import {
  meCancelCodexConnection,
  meDisconnectCodexConnection,
  meDescribeCodexConnection,
  meStartCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { usePermissions } from "@/contexts/usePermissions";
import { showErrorToast } from "@/lib/toast";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { invalidateWorkspaceModelQueries } from "@/hooks/workspaceModelQueries";

export function factoryCodexConnectionKey(organizationId: string, _factoryId: string, memberId = "") {
  return ["factory-codex-connection", organizationId, memberId] as const;
}

export function codexSubscriptionReady(connection: FactoriesFactoryCodexConnection | undefined): boolean {
  return connection?.status === "ready" && connection.reservationHeld !== true;
}

function invalidateSubscriptionCatalogs(queryClient: QueryClient, organizationId: string, factoryId: string) {
  return Promise.all([
    invalidateWorkspaceModelQueries(queryClient, organizationId, factoryId),
    queryClient.invalidateQueries({ queryKey: ["factories", organizationId], refetchType: "all" }),
    queryClient.invalidateQueries({
      queryKey: ["factories", organizationId, factoryId, "owner-eligibility"],
      refetchType: "all",
    }),
    queryClient.invalidateQueries({
      queryKey: ["factories", organizationId, factoryId, "line-runner-models"],
      refetchType: "all",
    }),
    queryClient.invalidateQueries({
      queryKey: ["organizations", organizationId, "selectable-llm-models"],
      refetchType: "all",
    }),
  ]);
}

export function useFactoryCodexConnection(
  organizationId: string,
  factoryId: string,
  enabled: boolean,
  refreshModels = true,
) {
  const { currentUserId } = usePermissions();
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
    enabled: enabled && Boolean(organizationId),
    queryFn: async ({ signal }) => {
      const response = await meDescribeCodexConnection(
        withOrganizationHeader({
          organizationId,
          signal: AbortSignal.any([signal, AbortSignal.timeout(15000)]),
        }),
      );
      return response.data?.connection;
    },
    refetchInterval: (current) => codexConnectionPollInterval(current.state.data, current.state.status === "error"),
    refetchIntervalInBackground: true,
  });

  const refresh = async () => {
    await queryClient.invalidateQueries({ queryKey });
    await invalidateSubscriptionCatalogs(queryClient, organizationId, factoryId);
    await queryClient.invalidateQueries({ queryKey: ["me", organizationId] });
  };

  const start = useMutation({
    mutationFn: async () => {
      const response = await meStartCodexConnection(
        withOrganizationHeader({
          organizationId,
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
      const response = await meStartCodexConnection(
        withOrganizationHeader({
          organizationId,
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
      const response = await meCancelCodexConnection(
        withOrganizationHeader({
          organizationId,
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
    mutationFn: async () => meDisconnectCodexConnection(withOrganizationHeader({ organizationId })),
    onSettled: refresh,
    onError: () => showErrorToast("Stop your tasks before you disconnect ChatGPT. Then try again."),
  });

  useEffect(() => {
    if (!enabled || !refreshModels || query.data?.status !== "ready") {
      refreshAttempted.current = false;
      return;
    }
    const availableModels = query.data.availableModels ?? [];
    const catalog = JSON.stringify(availableModels);
    if (invalidatedCatalog.current !== catalog) {
      invalidatedCatalog.current = catalog;
      void invalidateSubscriptionCatalogs(queryClient, organizationId, factoryId);
    }
    if (availableModels.length > 0) {
      return;
    }
    if (!query.data.reservationHeld && !refreshAttempted.current && !refreshCatalog.isPending) {
      refreshAttempted.current = true;
      refreshCatalog.mutate();
    }
  }, [enabled, refreshModels, organizationId, factoryId, query.data, queryClient, refreshCatalog]);

  return { query, start, cancel, refreshCatalog, disconnect, refresh };
}

function codexConnectionPollInterval(connection: FactoriesFactoryCodexConnection | undefined, failed: boolean) {
  if (failed) return 2000;
  const active =
    connection?.revocationPending ||
    ["pending", "canceling", "finishing"].includes(connection?.status ?? "") ||
    connection?.reservationHeld;
  return active ? 2000 : false;
}
