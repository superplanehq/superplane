import { useCallback, useEffect, useRef, useState } from "react";
import { showSuccessToast } from "@/lib/toast";
import type { OrganizationBillingPlan } from "./billingDiagnostics";

const SYNC_FAILED_COPY = "Could not sync the Polar customer. Try again.";

async function postCustomerSync(orgId: string, signal: AbortSignal) {
  const response = await fetch(`/admin/api/organizations/${orgId}/billing-customer/sync`, {
    method: "POST",
    credentials: "include",
    signal,
  });
  const result = (await response.json().catch(() => null)) as {
    billing_plan?: OrganizationBillingPlan;
    error?: string;
  } | null;
  return { ok: response.ok, ...result };
}

export function useBillingCustomerSync(
  orgId: string,
  options: {
    enabled: boolean;
    plan: OrganizationBillingPlan | null;
    busy: boolean;
    applyPlan: (plan: OrganizationBillingPlan) => void;
    onStart: () => void;
  },
) {
  const [syncingCustomer, setSyncingCustomer] = useState(false);
  const [customerSyncError, setCustomerSyncError] = useState("");
  const request = useRef<AbortController | null>(null);
  const isCustomerSyncing = useCallback(() => request.current !== null, []);
  const clearCustomerSyncError = useCallback(() => setCustomerSyncError(""), []);
  const polar = options.plan?.polar;
  const canSync =
    options.enabled &&
    !options.busy &&
    polar?.configured &&
    polar.customer_status === "found" &&
    polar.card_status === "usable";

  useEffect(() => {
    setSyncingCustomer(false);
    setCustomerSyncError("");
    return () => {
      request.current?.abort();
      request.current = null;
    };
  }, [orgId, options.enabled]);

  const syncCustomer = async () => {
    if (!canSync || request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setSyncingCustomer(true);
    setCustomerSyncError("");
    options.onStart();
    try {
      const result = await postCustomerSync(orgId, controller.signal);
      if (controller.signal.aborted) return;
      if (result.billing_plan) options.applyPlan(result.billing_plan);
      if (!result.ok || !result.billing_plan) throw new Error(result.error || SYNC_FAILED_COPY);
      showSuccessToast("Polar customer synced");
    } catch (error) {
      if (!controller.signal.aborted) setCustomerSyncError(error instanceof Error ? error.message : SYNC_FAILED_COPY);
    } finally {
      if (!controller.signal.aborted) {
        request.current = null;
        setSyncingCustomer(false);
      }
    }
  };

  return { syncingCustomer, customerSyncError, syncCustomer, isCustomerSyncing, clearCustomerSyncError };
}
