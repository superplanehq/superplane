import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  factoriesCreateFactoryWebhook,
  factoriesDeleteFactoryWebhook,
  factoriesListFactoryWebhooks,
  factoriesRegenerateFactoryWebhookSecret,
  factoriesUpdateFactoryWebhook,
  type FactoriesCreateFactoryWebhookBody,
} from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { factoryQueryKeys } from "./useFactoryData";

export function factoryWebhooksKey(organizationId: string, factoryId: string) {
  return [...factoryQueryKeys.detail(organizationId, factoryId), "webhooks"] as const;
}

export function useFactoryWebhooks(organizationId: string, factoryId: string, enabled = true) {
  return useQuery({
    queryKey: factoryWebhooksKey(organizationId, factoryId),
    queryFn: async () => {
      const response = await factoriesListFactoryWebhooks(
        withOrganizationHeader({ organizationId, path: { factoryId } }),
      );
      return response.data?.webhooks ?? [];
    },
    enabled: Boolean(organizationId && factoryId) && enabled,
  });
}

export function useFactoryWebhookMutations(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: factoryWebhooksKey(organizationId, factoryId) });

  const create = useMutation({
    mutationFn: async (body: FactoriesCreateFactoryWebhookBody) => {
      const response = await factoriesCreateFactoryWebhook(
        withOrganizationHeader({ organizationId, path: { factoryId }, body }),
      );
      return response.data;
    },
    onSuccess: invalidate,
  });
  const update = useMutation({
    mutationFn: async ({ webhookId, body }: { webhookId: string; body: FactoriesCreateFactoryWebhookBody }) => {
      const response = await factoriesUpdateFactoryWebhook(
        withOrganizationHeader({ organizationId, path: { factoryId, webhookId }, body }),
      );
      return response.data?.webhook;
    },
    onSuccess: invalidate,
  });
  const remove = useMutation({
    mutationFn: async (webhookId: string) => {
      await factoriesDeleteFactoryWebhook(withOrganizationHeader({ organizationId, path: { factoryId, webhookId } }));
    },
    onSuccess: invalidate,
  });
  const regenerateSecret = useMutation({
    mutationFn: async (webhookId: string) => {
      const response = await factoriesRegenerateFactoryWebhookSecret(
        withOrganizationHeader({ organizationId, path: { factoryId, webhookId }, body: {} }),
      );
      const secret = response.data?.secret;
      if (!secret) throw new Error("SuperPlane could not regenerate the webhook secret.");
      return secret;
    },
  });

  return { create, update, remove, regenerateSecret };
}
