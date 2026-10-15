import {
  factoriesCancelFactoryCodexConnection,
  factoriesDescribeFactoryCodexConnection,
  factoriesStartFactoryCodexConnection,
  type FactoriesFactoryCodexConnection,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

export function factoryCodexConnectionKey(organizationId: string, factoryId: string) {
  return ["factory-codex-connection", organizationId, factoryId] as const;
}

export function codexSubscriptionReady(connection: FactoriesFactoryCodexConnection | undefined): boolean {
  return connection?.status === "ready" && connection.reservationHeld !== true;
}

export function useFactoryCodexConnection(organizationId: string, factoryId: string, enabled: boolean) {
  const queryClient = useQueryClient();
  const queryKey = factoryCodexConnectionKey(organizationId, factoryId);
  const query = useQuery({
    queryKey,
    enabled: enabled && Boolean(organizationId && factoryId),
    queryFn: async () => {
      const response = await factoriesDescribeFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
        }),
      );
      return response.data?.connection;
    },
    refetchInterval: (current) => {
      const status = current.state.data?.status;
      return status === "pending" || status === "finishing" ? 2000 : false;
    },
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
        }),
      );
      return response.data?.connection;
    },
    onSuccess: refresh,
  });

  const cancel = useMutation({
    mutationFn: async () => {
      const response = await factoriesCancelFactoryCodexConnection(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {},
        }),
      );
      return response.data?.connection;
    },
    onSuccess: refresh,
  });

  return { query, start, cancel };
}
