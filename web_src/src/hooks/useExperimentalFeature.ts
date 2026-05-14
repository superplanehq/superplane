import { useCallback, useMemo } from "react";
import { useExperimentalFeaturesRegistry } from "./useExperimentalFeatures";
import { useOrganization } from "./useOrganizationData";
import { useOrganizationId } from "./useOrganizationId";

export interface ExperimentalFeatureAccess {
  has: (featureId: string) => boolean;
  enabledExperimentalFeatures: string[];
}

export function useExperimentalFeature(organizationId?: string): ExperimentalFeatureAccess {
  const _organizationId = useOrganizationId();
  const { data: organization } = useOrganization(organizationId || _organizationId || "");
  const { data: features } = useExperimentalFeaturesRegistry();

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

  const has = useCallback((featureId: string) => availableFeatures.has(featureId), [availableFeatures]);

  return { has, enabledExperimentalFeatures: [...availableFeatureIds] };
}
