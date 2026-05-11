import { hasExperimentalFeature } from "@/lib/features";
import { useOrganization } from "./useOrganizationData";
import { useOrganizationId } from "./useOrganizationId";

export function useExperimentalFeature(featureId: string): boolean {
  const organizationId = useOrganizationId();
  const { data: organization } = useOrganization(organizationId || "");
  return hasExperimentalFeature(organization, featureId);
}
