import { useCallback, useMemo } from "react";
import { useExperimentalFeaturesRegistry } from "./useExperimentalFeatures";
import { useOrganization } from "./useOrganizationData";
import { useOrganizationId } from "./useOrganizationId";

export interface ExperimentalFeatureAccess {
  has: (featureId: string) => boolean;
  enabledExperimentalFeatures: string[];
  isLoading: boolean;
  /** True when the feature registry request failed. */
  lookupFailed?: boolean;
}

export function useExperimentalFeature(organizationId?: string): ExperimentalFeatureAccess {
  const _organizationId = useOrganizationId();
  const resolvedOrganizationId = organizationId || _organizationId || "";
  const { data: organization, isLoading: organizationLoading } = useOrganization(resolvedOrganizationId);
  const {
    data: features,
    isLoading: registryLoading,
    isError: registryLookupFailed,
  } = useExperimentalFeaturesRegistry();
  const isLoading = registryLoading || (!!resolvedOrganizationId && organizationLoading);

  const enabledFeatures = useMemo(
    () => new Set(organization?.spec?.enabledExperimentalFeatures ?? []),
    [organization?.spec?.enabledExperimentalFeatures],
  );

  const availableFeatureIds = useMemo(() => {
    if (registryLookupFailed) {
      // A failed registry lookup must not hide organization-enabled flags.
      return [...enabledFeatures];
    }
    return (
      features?.features
        .filter((feature) => feature.released || enabledFeatures.has(feature.id))
        .map((feature) => feature.id) ?? []
    );
  }, [enabledFeatures, features?.features, registryLookupFailed]);

  const availableFeatures = useMemo(() => new Set(availableFeatureIds), [availableFeatureIds]);

  const has = useCallback((featureId: string) => availableFeatures.has(featureId), [availableFeatures]);

  return {
    has,
    enabledExperimentalFeatures: [...availableFeatureIds],
    isLoading,
    lookupFailed: registryLookupFailed,
  };
}
