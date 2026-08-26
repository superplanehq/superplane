import {
  factoriesCreateFactoryPrFeedbackHandler,
  factoriesDeleteFactoryPrFeedbackHandler,
  factoriesListFactoryPrFeedbackHandlerRuns,
  factoriesListFactoryPrFeedbackHandlers,
  factoriesUpdateFactoryPrFeedbackHandler,
} from "@/api-client";
import type {
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPrFeedbackHandlerRun,
  FactoriesFactoryPrFeedbackHandlerSettings,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { factoryAppsKey } from "./useFactoryData";

const factoryPRFeedbackQueryKeys = {
  list: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "pr-feedback-handlers"] as const,
  runs: (organizationId: string, factoryId: string, handlerId: string) =>
    ["factories", organizationId, factoryId, "pr-feedback-handlers", handlerId, "runs"] as const,
};

export function factoryPRFeedbackHandlersKey(organizationId: string, factoryId: string) {
  return factoryPRFeedbackQueryKeys.list(organizationId, factoryId);
}

export function factoryPRFeedbackHandlerRunsKey(organizationId: string, factoryId: string, handlerId: string) {
  return factoryPRFeedbackQueryKeys.runs(organizationId, factoryId, handlerId);
}

export async function fetchFactoryPRFeedbackHandlers(
  organizationId: string,
  factoryId: string,
): Promise<FactoriesFactoryPrFeedbackHandler[]> {
  const response = await factoriesListFactoryPrFeedbackHandlers(
    withOrganizationHeader({
      organizationId,
      path: { factoryId },
    }),
  );
  return response.data?.handlers ?? [];
}

export function useFactoryPRFeedbackHandlers(organizationId: string, factoryId: string) {
  return useQuery({
    queryKey: factoryPRFeedbackHandlersKey(organizationId, factoryId),
    queryFn: () => fetchFactoryPRFeedbackHandlers(organizationId, factoryId),
    enabled: Boolean(organizationId && factoryId),
  });
}

export async function fetchFactoryPRFeedbackHandlerRuns(
  organizationId: string,
  factoryId: string,
  handlerId: string,
): Promise<FactoriesFactoryPrFeedbackHandlerRun[]> {
  const response = await factoriesListFactoryPrFeedbackHandlerRuns(
    withOrganizationHeader({
      organizationId,
      path: { factoryId, handlerId },
    }),
  );
  return response.data?.runs ?? [];
}

export function useFactoryPRFeedbackHandlerRuns(
  organizationId: string,
  factoryId: string,
  handlerId: string | undefined,
  enabled = true,
) {
  return useQuery({
    queryKey: factoryPRFeedbackHandlerRunsKey(organizationId, factoryId, handlerId ?? ""),
    queryFn: () => fetchFactoryPRFeedbackHandlerRuns(organizationId, factoryId, handlerId ?? ""),
    enabled: Boolean(organizationId && factoryId && handlerId) && enabled,
    refetchInterval: 10_000,
  });
}

export function useCreateFactoryPRFeedbackHandler(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: { name?: string; repository?: string }) => {
      const response = await factoriesCreateFactoryPrFeedbackHandler(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            name: input.name,
            repository: input.repository,
          },
        }),
      );
      if (!response.data?.handler) {
        throw new Error("Failed to create PR feedback handler");
      }
      return response.data.handler;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryPRFeedbackHandlersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

export function useUpdateFactoryPRFeedbackHandler(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      handlerId: string;
      name?: string;
      settings?: FactoriesFactoryPrFeedbackHandlerSettings;
    }) => {
      const response = await factoriesUpdateFactoryPrFeedbackHandler(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, handlerId: input.handlerId },
          body: {
            name: input.name,
            settings: input.settings,
          },
        }),
      );
      if (!response.data?.handler) {
        throw new Error("Failed to update PR feedback handler");
      }
      return response.data.handler;
    },
    onSuccess: (_handler, variables) => {
      void queryClient.invalidateQueries({ queryKey: factoryPRFeedbackHandlersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({
        queryKey: factoryPRFeedbackQueryKeys.runs(organizationId, factoryId, variables.handlerId),
      });
    },
  });
}

export function useDeleteFactoryPRFeedbackHandler(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (handlerId: string) => {
      await factoriesDeleteFactoryPrFeedbackHandler(
        withOrganizationHeader({
          organizationId,
          path: { factoryId, handlerId },
        }),
      );
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: factoryPRFeedbackHandlersKey(organizationId, factoryId) });
      void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}
