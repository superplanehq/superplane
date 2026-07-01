import { useMutation, useQueryClient } from "@tanstack/react-query";

import { organizationKeys } from "./useOrganizationData";

export interface ExperimentalFeature {
  id: string;
  label: string;
  description: string;
  released: boolean;
}

export interface ExperimentalFeaturesRegistry {
  features: ExperimentalFeature[];
  enabled: string[];
}

export const adminExperimentalFeaturesKeys = {
  all: ["adminExperimentalFeatures"] as const,
  registry: (orgId: string) => [...adminExperimentalFeaturesKeys.all, "registry", orgId] as const,
};

async function toggleAdminExperimentalFeature(orgId: string, featureId: string, enabled: boolean): Promise<void> {
  const res = await fetch(`/admin/api/organizations/${orgId}/experimental-features/${featureId}`, {
    method: enabled ? "POST" : "DELETE",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`Failed to ${enabled ? "enable" : "disable"} ${featureId}`);
  }
}

export interface ToggleAdminExperimentalFeatureVariables {
  featureId: string;
  enabled: boolean;
}

interface ToggleMutationContext {
  previous?: ExperimentalFeaturesRegistry;
}

export const useToggleAdminExperimentalFeature = (orgId: string) => {
  const queryClient = useQueryClient();
  const queryKey = adminExperimentalFeaturesKeys.registry(orgId);

  return useMutation<void, Error, ToggleAdminExperimentalFeatureVariables, ToggleMutationContext>({
    mutationFn: ({ featureId, enabled }) => toggleAdminExperimentalFeature(orgId, featureId, enabled),
    onMutate: async ({ featureId, enabled }) => {
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<ExperimentalFeaturesRegistry>(queryKey);
      if (previous) {
        const next = new Set(previous.enabled);
        if (enabled) next.add(featureId);
        else next.delete(featureId);
        queryClient.setQueryData<ExperimentalFeaturesRegistry>(queryKey, {
          ...previous,
          enabled: Array.from(next),
        });
      }
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(queryKey, context.previous);
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: organizationKeys.details(orgId) });
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey });
    },
  });
};
