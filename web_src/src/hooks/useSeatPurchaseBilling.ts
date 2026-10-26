import { useCallback, useEffect, useRef, useState } from "react";
import type { QueryClient } from "@tanstack/react-query";

import { organizationsDescribeOrganizationBilling } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { organizationBillingQueryKey, syncOrganizationBilling } from "./useOrganizationBilling";
import type { useSeatPurchaseState } from "./useSeatPurchaseState";

export function useSeatPurchaseBilling({
  organizationId,
  enabled,
  queryClient,
  state,
}: {
  organizationId: string;
  enabled: boolean;
  queryClient: QueryClient;
  state: ReturnType<typeof useSeatPurchaseState>;
}) {
  const [refreshing, setRefreshing] = useState(false);
  const lock = useRef(false);
  const { lifecycle, recordPurchase, setMessage } = state;
  useEffect(() => {
    setRefreshing(false);
    lock.current = false;
  }, [organizationId]);

  const reloadBilling = useCallback(async () => {
    try {
      return await syncOrganizationBilling(organizationId, queryClient);
    } catch {
      // The local billing response can expose a purchase recorded before Polar failed.
      const response = await organizationsDescribeOrganizationBilling(
        withOrganizationHeader({ organizationId, path: { id: organizationId } }),
      );
      if (!response.data) throw new Error("SuperPlane could not refresh billing. Try again.");
      queryClient.setQueryData(organizationBillingQueryKey(organizationId), response.data);
      return response.data;
    }
  }, [organizationId, queryClient]);

  const refreshBilling = useCallback(async () => {
    if (lock.current || !organizationId || !enabled) return;
    lock.current = true;
    const generation = lifecycle.current;
    setRefreshing(true);
    try {
      const billing = await reloadBilling();
      if (generation === lifecycle.current) recordPurchase(billing.lastSeatPurchase);
    } catch (error) {
      if (generation === lifecycle.current)
        setMessage(getApiErrorMessage(error, "SuperPlane could not refresh billing. Try again."));
    } finally {
      if (generation === lifecycle.current) {
        lock.current = false;
        setRefreshing(false);
      }
    }
  }, [organizationId, enabled, lifecycle, reloadBilling, recordPurchase, setMessage]);

  return { reloadBilling, refreshBilling, refreshing };
}
