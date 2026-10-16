import { useAccount } from "@/contexts/useAccount";

/**
 * Product billing stays on unless the account response says it is off.
 * An unset account field keeps billing, so local development and SuperPlane
 * Cloud do not hide Billing while the account is loading.
 */
export function useBillingProductEnabled(): boolean {
  const { account, loading } = useAccount();
  if (loading || !account) {
    return true;
  }
  return account.billing_enabled !== false;
}
