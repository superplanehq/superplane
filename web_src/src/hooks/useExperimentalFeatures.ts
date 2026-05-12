import { useQuery } from "@tanstack/react-query";

export interface ExperimentalFeature {
  id: string;
  label: string;
  description: string;
  released: boolean;
}

export interface ExperimentalFeaturesRegistry {
  features: ExperimentalFeature[];
}

export const experimentalFeaturesKeys = {
  all: ["experimentalFeatures"] as const,
  registry: () => [...experimentalFeaturesKeys.all, "registry"] as const,
};

async function fetchExperimentalFeatures(): Promise<ExperimentalFeaturesRegistry> {
  const res = await fetch("/account/experimental-features", {
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`Failed to load experimental features (${res.status})`);
  }
  const data = (await res.json()) as Partial<ExperimentalFeaturesRegistry>;
  return {
    features: data.features ?? [],
  };
}

export const useExperimentalFeaturesRegistry = (enabled = true) => {
  return useQuery({
    queryKey: experimentalFeaturesKeys.registry(),
    queryFn: fetchExperimentalFeatures,
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled,
  });
};
