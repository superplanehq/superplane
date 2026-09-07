import {
  factoriesCreateFactoryPrFeedbackHandler,
  factoriesDeleteFactoryPrFeedbackHandler,
  factoriesListFactoryPrFeedbackHandlers,
  factoriesUpdateFactoryPrFeedbackHandler,
} from "@/api-client";
import type {
  FactoriesFactoryPrFeedbackHandler,
  FactoriesFactoryPrFeedbackHandlerSettings,
  FactoriesFactoryPrFeedbackHandlerSource,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { factoryAppsKey, factoryQueryKeys } from "./useFactoryData";

const factoryPRFeedbackQueryKeys = {
  list: (organizationId: string, factoryId: string) =>
    ["factories", organizationId, factoryId, "pr-feedback-handlers"] as const,
};

export function factoryPRFeedbackHandlersKey(organizationId: string, factoryId: string) {
  return factoryPRFeedbackQueryKeys.list(organizationId, factoryId);
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

function invalidatePRFeedbackQueries(queryClient: QueryClient, organizationId: string, factoryId: string) {
  void queryClient.invalidateQueries({ queryKey: factoryPRFeedbackHandlersKey(organizationId, factoryId) });
  void queryClient.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
  void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.workOrders(organizationId, factoryId) });
}

export function useCreateFactoryPRFeedbackHandler(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: {
      name?: string;
      repository?: string;
      source?: FactoriesFactoryPrFeedbackHandlerSource;
    }) => {
      const response = await factoriesCreateFactoryPrFeedbackHandler(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            name: input.name,
            source: input.source,
            settings: input.repository ? { subject: { repository: input.repository } } : undefined,
          },
        }),
      );
      if (!response.data?.handler) {
        throw new Error("Failed to create PR feedback handler");
      }
      return response.data.handler;
    },
    onSuccess: () => {
      invalidatePRFeedbackQueries(queryClient, organizationId, factoryId);
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
    onSuccess: () => {
      invalidatePRFeedbackQueries(queryClient, organizationId, factoryId);
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
      invalidatePRFeedbackQueries(queryClient, organizationId, factoryId);
    },
  });
}
