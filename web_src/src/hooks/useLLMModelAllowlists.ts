import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  factoriesListFactoryLlmModels,
  factoriesUpdateFactoryLlmModels,
  organizationsCreateBillingPortalSession,
  organizationsCreateHostedCreditCheckout,
  organizationsListByokllmModels,
  organizationsListHostedCreditProducts,
  organizationsUpdateByokllmModels,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

const BYOK_PROVIDERS = ["anthropic", "openai", "openrouter"] as const;

export function byokLLMModelsQueryKey(organizationId: string, provider: string, factoryId?: string) {
  return ["organizations", organizationId, "byok-models", provider, factoryId ?? ""] as const;
}

export function useBYOKLLMModels(
  organizationId: string | undefined,
  provider: string | undefined,
  enabled: boolean,
  factoryId?: string,
) {
  return useQuery({
    queryKey: byokLLMModelsQueryKey(organizationId ?? "", provider ?? "", factoryId),
    queryFn: async () => {
      const response = await organizationsListByokllmModels(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
          query: { provider, factoryId },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && provider && enabled),
    staleTime: 30 * 1000,
  });
}

export function useUpdateBYOKLLMModels(organizationId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { provider: string; allowedModels: string[] }) => {
      const response = await organizationsUpdateByokllmModels(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: { provider: input.provider, allowedModels: input.allowedModels },
        }),
      );
      return response.data ?? {};
    },
    onSuccess: (_data, input) => {
      void queryClient.invalidateQueries({
        queryKey: ["organizations", organizationId, "byok-models", input.provider],
      });
    },
  });
}

export function hostedCreditProductsQueryKey(organizationId: string) {
  return ["organizations", organizationId, "hosted-credit-products"] as const;
}

export function useHostedCreditProducts(organizationId: string | undefined, enabled: boolean) {
  return useQuery({
    queryKey: hostedCreditProductsQueryKey(organizationId ?? ""),
    queryFn: async () => {
      const response = await organizationsListHostedCreditProducts(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { id: organizationId! },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && enabled),
    staleTime: 30 * 1000,
  });
}

export function useCreateHostedCreditCheckout(organizationId: string) {
  return useMutation({
    mutationFn: async (productId: string) => {
      const response = await organizationsCreateHostedCreditCheckout(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: { productId },
        }),
      );
      const checkoutUrl = response.data?.checkoutUrl;
      if (!checkoutUrl) {
        throw new Error("Checkout is not available");
      }
      return checkoutUrl;
    },
  });
}

export function useCreateBillingPortalSession(organizationId: string) {
  return useMutation({
    mutationFn: async () => {
      const response = await organizationsCreateBillingPortalSession(
        withOrganizationHeader({
          organizationId,
          path: { id: organizationId },
          body: {},
        }),
      );
      const portalUrl = response.data?.portalUrl;
      if (!portalUrl) {
        throw new Error("Add hosted credit first.");
      }
      return portalUrl;
    },
  });
}

export function factoryLLMModelsQueryKey(
  organizationId: string,
  factoryId: string,
  provider: string,
  fundingSource: string,
) {
  return [...factoryQueryKeys.detail(organizationId, factoryId), "llm-models", provider, fundingSource] as const;
}

export function useFactoryLLMModels(
  organizationId: string | undefined,
  factoryId: string | undefined,
  provider: string | undefined,
  fundingSource: string | undefined,
  enabled: boolean,
) {
  return useQuery({
    queryKey: factoryLLMModelsQueryKey(organizationId ?? "", factoryId ?? "", provider ?? "", fundingSource ?? ""),
    queryFn: async () => {
      const response = await factoriesListFactoryLlmModels(
        withOrganizationHeader({
          organizationId: organizationId!,
          path: { factoryId: factoryId! },
          query: { provider, fundingSource },
        }),
      );
      return response.data ?? {};
    },
    enabled: Boolean(organizationId && factoryId && provider && fundingSource && enabled),
    staleTime: 30 * 1000,
  });
}

export function useUpdateFactoryLLMModels(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: { provider: string; fundingSource: string; allowedModels: string[] }) => {
      const response = await factoriesUpdateFactoryLlmModels(
        withOrganizationHeader({
          organizationId,
          path: { factoryId },
          body: {
            provider: input.provider,
            fundingSource: input.fundingSource,
            allowedModels: input.allowedModels,
          },
        }),
      );
      return response.data ?? {};
    },
    onSuccess: (_data, input) => {
      void queryClient.invalidateQueries({
        queryKey: factoryLLMModelsQueryKey(organizationId, factoryId, input.provider, input.fundingSource),
      });
    },
  });
}

export { BYOK_PROVIDERS };
