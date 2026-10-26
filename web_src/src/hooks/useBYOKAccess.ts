import { useExperimentalFeature } from "./useExperimentalFeature";
import { useOrganizationBilling } from "./useOrganizationBilling";
import { useOrganizationWorkspaceUsage } from "./useOrganizationWorkspaceUsage";
import { FEATURE_ORGANIZATION_BYOK, FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER } from "@/lib/experimentalFeatures";
import { hasHostedDefaultModel } from "@/lib/hostedLLMModels";

/** Product permissions are resolved by the server, including self-host defaults. */
export function useBYOKAccess(organizationId: string | undefined) {
  const rollout = useExperimentalFeature(organizationId);
  const billing = useOrganizationBilling(organizationId);
  const usage = useOrganizationWorkspaceUsage(organizationId ?? "");
  const hostedDefaultReady = usage.isSuccess && !usage.isError && usage.data != null;
  const noHostedDefault = hostedDefaultReady && !hasHostedDefaultModel(usage.data);
  const product = productBYOKAccess(rollout, billing);
  const allowed = noHostedDefault || product.allowed;
  const ready = allowed || (hostedDefaultReady && product.ready);
  return {
    allowed,
    customProviderAllowed: product.allowed && rollout.has(FEATURE_ORGANIZATION_BYOK_CUSTOM_PROVIDER),
    isLoading: !ready && (usage.isPending || rollout.isLoading || billing.isPending),
    ready,
    error: allowed ? null : (usage.error ?? billing.error),
    async verify() {
      const hosted = await usage.refetch();
      const hostedLookupReady = !hosted.error && hosted.data != null;
      if (hostedLookupReady && !hasHostedDefaultModel(hosted.data)) {
        return true;
      }
      const result = await billing.refetch();
      const productFeatures = result.data?.productFeatures;
      const allowed = rollout.has(FEATURE_ORGANIZATION_BYOK) && productFeatures?.byo_key === true;
      if (
        result.error ||
        !productFeatures ||
        !rollout.organizationReady ||
        rollout.isLoading ||
        rollout.lookupFailed ||
        (!allowed && !hostedLookupReady)
      ) {
        throw new Error("SuperPlane could not verify model access. Try again.");
      }
      return allowed;
    },
  };
}

function productBYOKAccess(
  rollout: ReturnType<typeof useExperimentalFeature>,
  billing: ReturnType<typeof useOrganizationBilling>,
) {
  const ready =
    rollout.organizationReady &&
    !rollout.isLoading &&
    !rollout.lookupFailed &&
    billing.isSuccess &&
    !billing.isError &&
    billing.data.productFeatures != null;
  const allowed = ready && rollout.has(FEATURE_ORGANIZATION_BYOK) && billing.data?.productFeatures?.byo_key === true;
  return { ready, allowed };
}
