import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

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

async function fetchAdminExperimentalFeatures(orgId: string): Promise<ExperimentalFeaturesRegistry> {
  const res = await fetch(`/admin/api/organizations/${orgId}/experimental-features`, {
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`Failed to load experimental features (${res.status})`);
  }
  const data = (await res.json()) as Partial<ExperimentalFeaturesRegistry>;
  return {
    features: data.features ?? [],
    enabled: data.enabled ?? [],
  };
}

export const useAdminExperimentalFeaturesRegistry = (orgId: string, enabled = true) => {
  return useQuery({
    queryKey: adminExperimentalFeaturesKeys.registry(orgId),
    queryFn: () => fetchAdminExperimentalFeatures(orgId),
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: !!orgId && enabled,
  });
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

export interface ToggleAdminExperimentalFeaturesVariables {
  featureIds: string[];
  enabled: boolean;
}

interface ToggleMutationContext {
  previous?: ExperimentalFeaturesRegistry;
}

function setEnabledFeatureIds(
  previous: ExperimentalFeaturesRegistry,
  featureIds: string[],
  enabled: boolean,
): ExperimentalFeaturesRegistry {
  const next = new Set(previous.enabled);
  for (const featureId of featureIds) {
    if (enabled) next.add(featureId);
    else next.delete(featureId);
  }
  return { ...previous, enabled: Array.from(next) };
}

async function toggleAdminExperimentalFeatures(orgId: string, featureIds: string[], enabled: boolean): Promise<void> {
  for (const featureId of featureIds) {
    await toggleAdminExperimentalFeature(orgId, featureId, enabled);
  }
}

function useAdminExperimentalFeatureMutation<TVariables>(
  orgId: string,
  mutationFn: (variables: TVariables) => Promise<void>,
  featureIdsFrom: (variables: TVariables) => { featureIds: string[]; enabled: boolean },
) {
  const queryClient = useQueryClient();
  const queryKey = adminExperimentalFeaturesKeys.registry(orgId);

  return useMutation<void, Error, TVariables, ToggleMutationContext>({
    mutationFn,
    onMutate: async (variables) => {
      const { featureIds, enabled } = featureIdsFrom(variables);
      await queryClient.cancelQueries({ queryKey });
      const previous = queryClient.getQueryData<ExperimentalFeaturesRegistry>(queryKey);
      if (previous) {
        queryClient.setQueryData<ExperimentalFeaturesRegistry>(
          queryKey,
          setEnabledFeatureIds(previous, featureIds, enabled),
        );
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
}

export const useToggleAdminExperimentalFeature = (orgId: string) => {
  return useAdminExperimentalFeatureMutation(
    orgId,
    ({ featureId, enabled }: ToggleAdminExperimentalFeatureVariables) =>
      toggleAdminExperimentalFeature(orgId, featureId, enabled),
    ({ featureId, enabled }) => ({ featureIds: [featureId], enabled }),
  );
};

export const useToggleAdminExperimentalFeatures = (orgId: string) => {
  return useAdminExperimentalFeatureMutation(
    orgId,
    ({ featureIds, enabled }: ToggleAdminExperimentalFeaturesVariables) =>
      toggleAdminExperimentalFeatures(orgId, featureIds, enabled),
    ({ featureIds, enabled }) => ({ featureIds, enabled }),
  );
};
