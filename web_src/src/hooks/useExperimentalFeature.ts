import { useCallback, useMemo } from "react";
import {
  FEATURE_WORKSPACE_AGENT_RESOURCES,
  FEATURE_WORKSPACE_MCP,
  FEATURE_WORKSPACE_SKILLS,
} from "@/lib/experimentalFeatures";
import { useExperimentalFeaturesRegistry } from "./useExperimentalFeatures";
import { useOrganization } from "./useOrganizationData";
import { useOrganizationId } from "./useOrganizationId";

export interface ExperimentalFeatureAccess {
  has: (featureId: string) => boolean;
  enabledExperimentalFeatures: string[];
  isLoading: boolean;
}

export function useExperimentalFeature(organizationId?: string): ExperimentalFeatureAccess {
  const _organizationId = useOrganizationId();
  const resolvedOrganizationId = organizationId || _organizationId || "";
  const { data: organization, isLoading: organizationLoading } = useOrganization(resolvedOrganizationId);
  const { data: features, isLoading: registryLoading } = useExperimentalFeaturesRegistry();
  const isLoading = registryLoading || (!!resolvedOrganizationId && organizationLoading);

  const enabledFeatures = useMemo(
    () => new Set(organization?.spec?.enabledExperimentalFeatures ?? []),
    [organization?.spec?.enabledExperimentalFeatures],
  );

  const availableFeatureIds = useMemo(() => {
    return (
      features?.features
        .filter((feature) => feature.released || enabledFeatures.has(feature.id))
        .map((feature) => feature.id) ?? []
    );
  }, [enabledFeatures, features?.features]);

  const availableFeatures = useMemo(() => new Set(availableFeatureIds), [availableFeatureIds]);

  const has = useCallback(
    (featureId: string) => {
      if (availableFeatures.has(featureId)) {
        return true;
      }
      if (featureId === FEATURE_WORKSPACE_MCP || featureId === FEATURE_WORKSPACE_SKILLS) {
        return availableFeatures.has(FEATURE_WORKSPACE_AGENT_RESOURCES);
      }
      return false;
    },
    [availableFeatures],
  );

  return { has, enabledExperimentalFeatures: [...availableFeatureIds], isLoading };
}
