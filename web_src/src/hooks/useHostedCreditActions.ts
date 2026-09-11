import { useOrganization, useOrganizationUsers } from "@/hooks/useOrganizationData";
import {
  useCreateBillingPortalSession,
  useCreateHostedCreditCheckout,
  useHostedCreditProducts,
} from "@/hooks/useLLMModelAllowlists";
import { useCreateBusinessCheckout } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { clearHostedCreditGrantSnapshot, rememberHostedCreditGrantSnapshot } from "@/lib/hostedCredit";
import { hostedCreditOwnerContactCopy } from "@/lib/hostedCreditOwnerContact";
import { showErrorToast } from "@/lib/toast";

export function useHostedCreditActions(
  organizationId: string,
  billingEnabled: boolean,
  grantTotalCents: number,
  creditPurchaseAllowed = false,
) {
  const productsQuery = useHostedCreditProducts(
    organizationId,
    Boolean(organizationId) && billingEnabled && creditPurchaseAllowed,
  );
  const checkout = useCreateHostedCreditCheckout(organizationId);
  const businessCheckout = useCreateBusinessCheckout(organizationId);
  const portal = useCreateBillingPortalSession(organizationId);

  return {
    products: productsQuery.data?.products ?? [],
    productsLoading: productsQuery.isLoading,
    checkoutPending: checkout.isPending,
    businessCheckoutPending: businessCheckout.isPending,
    portalPending: portal.isPending,
    startCheckout: async (productId: string) => {
      rememberHostedCreditGrantSnapshot(organizationId, grantTotalCents);
      try {
        const url = await checkout.mutateAsync(productId);
        window.location.assign(url);
      } catch (checkoutError) {
        clearHostedCreditGrantSnapshot(organizationId);
        showErrorToast(getApiErrorMessage(checkoutError, "Unable to start checkout."));
      }
    },
    startBusinessCheckout: async () => {
      try {
        const url = await businessCheckout.mutateAsync();
        window.location.assign(url);
      } catch (checkoutError) {
        showErrorToast(getApiErrorMessage(checkoutError, "Unable to start checkout."));
      }
    },
    openInvoices: async () => {
      try {
        const url = await portal.mutateAsync();
        window.location.assign(url);
      } catch (portalError) {
        showErrorToast(getApiErrorMessage(portalError, "Add hosted credit first."));
      }
    },
  };
}

/**
 * Builds the "contact an owner" sentence shown to members without billing
 * permission in place of the hosted credit checkout control. Only fetched when
 * Polar is on and the signed-in user cannot manage billing.
 */
export function useHostedCreditOwnerContactMessage(organizationId: string, needed: boolean): string | undefined {
  const { data: organization } = useOrganization(organizationId, needed);
  const { data: users = [] } = useOrganizationUsers(organizationId, true, needed);

  if (!needed) {
    return undefined;
  }

  const owners = users
    .filter((user) => user.status?.roles?.some((role) => role.roleName === "org_owner"))
    .map((user) => ({ name: user.spec?.displayName, email: user.metadata?.email }));

  return hostedCreditOwnerContactCopy({
    organizationName: organization?.metadata?.name,
    owners,
  });
}
