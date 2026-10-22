import { useExperimentalFeature } from "./useExperimentalFeature";
import { useOrganizationBilling } from "./useOrganizationBilling";
import { FEATURE_ORGANIZATION_BYOK, FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER } from "@/lib/experimentalFeatures";

/** Product permissions are resolved by the server, including self-host defaults. */
export function useBYOKAccess(organizationId: string | undefined) {
  const rollout = useExperimentalFeature(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const ready =
    rollout.organizationReady &&
    !rollout.isLoading &&
    !rollout.lookupFailed &&
    billing.isSuccess &&
    !billing.isError &&
    billing.data.productFeatures != null;
  const allowed = ready && rollout.has(FEATURE_ORGANIZATION_BYOK) && billing.data?.productFeatures?.byo_key === true;
  return {
    allowed,
    customProviderAllowed: allowed && rollout.has(FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER),
    isLoading: rollout.isLoading || billing.isPending,
    ready,
    error: billing.error,
    async verify() {
      const result = await billing.refetch();
      if (
        result.error ||
        !result.data?.productFeatures ||
        !rollout.organizationReady ||
        rollout.isLoading ||
        rollout.lookupFailed
      ) {
        throw new Error("SuperPlane could not verify model access. Try again.");
      }
      return rollout.has(FEATURE_ORGANIZATION_BYOK) && result.data.productFeatures.byo_key === true;
    },
  };
}
