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
  /**
   * True only after the organization request succeeds and returns an organization.
   * A failed or empty organization lookup does not confirm that a feature is off.
   */
  organizationReady: boolean;
}

export function useExperimentalFeature(organizationId?: string): ExperimentalFeatureAccess {
  const _organizationId = useOrganizationId();
  const resolvedOrganizationId = organizationId || _organizationId || "";
  const {
    data: organization,
    isLoading: organizationLoading,
    isSuccess: organizationLoaded,
  } = useOrganization(resolvedOrganizationId);
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

  const has = useCallback(
    (featureId: string) => {
      return availableFeatures.has(featureId);
    },
    [availableFeatures],
  );

  return {
    has,
    enabledExperimentalFeatures: [...availableFeatureIds],
    isLoading,
    lookupFailed: registryLookupFailed,
    organizationReady: organizationLoaded && organization != null,
  };
}
