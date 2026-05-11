import type { OrganizationsOrganization } from "@/api-client";

export function hasExperimentalFeature(
  organization: OrganizationsOrganization | null | undefined,
  featureId: string,
): boolean {
  return organization?.spec?.enabledExperimentalFeatures?.includes(featureId) ?? false;
}
