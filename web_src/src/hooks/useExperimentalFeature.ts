import { useOrganization } from "./useOrganizationData";
import { useOrganizationId } from "./useOrganizationId";
import { useExperimentalFeaturesRegistry } from "./useAdminExperimentalFeatures";

export function useExperimentalFeature(featureId: string): boolean {
  const organizationId = useOrganizationId();
  const { data: organization } = useOrganization(organizationId || "");
  const { data: features } = useExperimentalFeaturesRegistry(organizationId || "");

  const exists = features?.features.some((f) => f.id === featureId);
  const released = features?.features.some((f) => f.id === featureId && f.released);
  if (!exists) {
    return false;
  }
  if (released) {
    return true;
  }
  return organization?.spec?.enabledExperimentalFeatures?.includes(featureId) ?? false;
}
