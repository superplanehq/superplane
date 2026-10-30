import type { OrganizationBillingPlan } from "./billingDiagnostics";
export type { OrganizationBillingPlan, BillingPolarDiagnostics } from "./billingDiagnostics";
import { billingCatalogRequest, type BillingPlan } from "./billingPlansApi";
import type { OrganizationsOrganizationCreditGrant } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { bpsToPercentInput, centsToDollarInput, parseDollarInputToCents, percentInputToBps } from "@/lib/hostedCredit";
import { useCallback, useEffect, useRef, useState } from "react";
import { useBillingCustomerSync } from "./useBillingCustomerSync";

export type OrganizationLLMCredit = {
  remaining_credit_cents: number;
  grant_total_cents: number;
  superplane_grant_cents: number;
  purchased_credit_cents: number;
  hosted_billed_cents: number;
  welcome_remaining_cents: number;
  included_remaining_cents: number;
  purchased_remaining_cents: number;
  admin_remaining_cents: number;
  welcome_credit_expires_at: string | null;
  markup_bps: number;
  markup_override_bps: number | null;
  warning: boolean;
};

export type CreditBalanceBucket = "trial" | "topup" | "grant";

export type CreditBalanceInputs = Record<CreditBalanceBucket, string>;

export const CREDIT_BALANCE_CHANGED_COPY = "The balance changed. Reload and try again.";

const EMPTY_BALANCE_INPUTS: CreditBalanceInputs = { trial: "", topup: "", grant: "" };
const TRIAL_LENGTH_DAYS = 14;

export function creditRemainingCents(credit: OrganizationLLMCredit, bucket: CreditBalanceBucket): number {
  switch (bucket) {
    case "trial":
      return credit.welcome_remaining_cents ?? 0;
    case "topup":
      return credit.purchased_remaining_cents ?? 0;
    case "grant":
      return credit.admin_remaining_cents ?? 0;
  }
}

function balanceInputsFromCredit(credit: OrganizationLLMCredit): CreditBalanceInputs {
  return {
    trial: centsToDollarInput(creditRemainingCents(credit, "trial")),
    topup: centsToDollarInput(creditRemainingCents(credit, "topup")),
    grant: centsToDollarInput(creditRemainingCents(credit, "grant")),
  };
}

class CreditBalanceChangedError extends Error {}

export function utcCalendarDate(value: Date): string {
  return value.toISOString().slice(0, 10);
}

export function prefilledTrialEndDate(trialEndsAt: string | null, now = new Date()): string {
  if (trialEndsAt) {
    const parsed = new Date(trialEndsAt);
    if (!Number.isNaN(parsed.getTime()) && parsed.getTime() > now.getTime()) {
      return utcCalendarDate(parsed);
    }
  }
  const day = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
  day.setUTCDate(day.getUTCDate() + TRIAL_LENGTH_DAYS);
  return utcCalendarDate(day);
}

export function utcTrialEndTimestamp(calendarDate: string): string {
  return `${calendarDate}T23:59:59Z`;
}

export function isFutureUtcTrialEnd(calendarDate: string, now = new Date()): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(calendarDate)) {
    return false;
  }
  const end = new Date(utcTrialEndTimestamp(calendarDate));
  return !Number.isNaN(end.getTime()) && end.getTime() > now.getTime();
}

function billingPlanRequest(
  plan: string,
  category: string | undefined,
  trialEndsOn: string,
): { plan: string; trial_ends_at?: string } {
  if (category !== "trial") {
    return { plan };
  }
  return { plan, trial_ends_at: utcTrialEndTimestamp(trialEndsOn) };
}

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  const text = await response.text();
  if (text.trim() === "") {
    return fallback;
  }
  return text;
}

async function fetchOrganizationBillingPlan(
  orgId: string,
  checkPolar = false,
  signal?: AbortSignal,
): Promise<OrganizationBillingPlan> {
  const response = await fetch(
    `/admin/api/organizations/${orgId}/billing-plan${checkPolar ? "?check_polar=true" : ""}`,
    { credentials: "include", signal },
  );
  if (!response.ok) throw new Error("Could not load billing diagnostics. Try again.");
  return (await response.json()) as OrganizationBillingPlan;
}

async function putOrganizationBillingPlan(
  orgId: string,
  body: { plan: string; trial_ends_at?: string },
): Promise<OrganizationBillingPlan> {
  const response = await fetch(`/admin/api/organizations/${orgId}/billing-plan`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to set billing plan"));
  }
  return (await response.json()) as OrganizationBillingPlan;
}

async function fetchOrganizationCreditGrants(orgId: string): Promise<OrganizationsOrganizationCreditGrant[]> {
  const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit/grants`, { credentials: "include" });
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to load credit history"));
  }
  const data = (await response.json()) as { grants?: OrganizationsOrganizationCreditGrant[] };
  return data.grants ?? [];
}

async function putOrganizationCreditBalance(
  orgId: string,
  body: {
    bucket: CreditBalanceBucket;
    target_cents: number;
    expected_remaining_cents: number;
    note: string;
  },
): Promise<OrganizationLLMCredit> {
  const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit/balances`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(body),
  });
  if (response.status === 409) {
    throw new CreditBalanceChangedError(CREDIT_BALANCE_CHANGED_COPY);
  }
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to set credit balance"));
  }
  return (await response.json()) as OrganizationLLMCredit;
}

async function loadOrganizationCreditState(orgId: string, applyCredit: (data: OrganizationLLMCredit) => void) {
  const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit`, { credentials: "include" });
  if (!response.ok) throw new Error(await readErrorMessage(response, "Failed to load organization credit"));
  applyCredit(await response.json());
}

function useBillingPlanCatalog(enabled: boolean) {
  const [catalog, setCatalog] = useState<BillingPlan[]>([]);
  const [catalogError, setCatalogError] = useState("");
  useEffect(() => {
    if (!enabled) return;
    let cancelled = false;
    setCatalog([]);
    setCatalogError("");
    billingCatalogRequest<BillingPlan[]>("billing-plans")
      .then((plans) => {
        if (!cancelled) setCatalog(plans);
      })
      .catch(() => {
        if (!cancelled) setCatalogError("Failed to load billing plans. Refresh and try again.");
      });
    return () => {
      cancelled = true;
    };
  }, [enabled]);
  return { catalog, catalogError };
}

function useOrgBillingPlan(orgId: string, enabled: boolean) {
  const { catalog, catalogError } = useBillingPlanCatalog(enabled);
  const [plan, setPlan] = useState<OrganizationBillingPlan | null>(null);
  const [planValue, setPlanValue] = useState("trial");
  const [trialEndsOn, setTrialEndsOn] = useState("");
  const [savingPlan, setSavingPlan] = useState(false);

  const applyPlan = useCallback((nextPlan: OrganizationBillingPlan) => {
    setPlan(nextPlan);
    setPlanValue(nextPlan.catalog_key || nextPlan.plan || "none");
    setTrialEndsOn(prefilledTrialEndDate(nextPlan.trial_ends_at));
  }, []);

  const [diagnosticsLoading, setDiagnosticsLoading] = useState(enabled);
  const [diagnosticsError, setDiagnosticsError] = useState("");
  const [checkingPolar, setCheckingPolar] = useState(false);
  const request = useRef<AbortController | null>(null);
  const customerSync = useBillingCustomerSync(orgId, {
    enabled,
    plan,
    busy: savingPlan || diagnosticsLoading || checkingPolar,
    applyPlan,
    onStart: () => setDiagnosticsError(""),
  });
  const { isCustomerSyncing, clearCustomerSyncError } = customerSync;
  const loadDiagnostics = useCallback(
    async (checkPolar = false) => {
      if (isCustomerSyncing()) return;
      request.current?.abort();
      const controller = new AbortController();
      request.current = controller;
      setDiagnosticsError("");
      clearCustomerSyncError();
      setDiagnosticsLoading(!checkPolar);
      setCheckingPolar(checkPolar);
      if (checkPolar)
        setPlan((current) => (current ? { ...current, polar_subscription_verification: "not_checked" } : current));
      try {
        const next = await fetchOrganizationBillingPlan(orgId, checkPolar, controller.signal);
        if (!controller.signal.aborted) applyPlan(next);
      } catch (error) {
        if (!controller.signal.aborted) {
          setDiagnosticsError(
            error instanceof Error ? error.message : "Could not load billing diagnostics. Try again.",
          );
          if (checkPolar)
            setPlan((current) => (current ? { ...current, polar_subscription_verification: "failed" } : current));
        }
      } finally {
        if (!controller.signal.aborted) {
          setDiagnosticsLoading(false);
          setCheckingPolar(false);
        }
      }
    },
    [orgId, applyPlan, isCustomerSyncing, clearCustomerSyncError],
  );
  useEffect(() => {
    setPlan(null);
    if (enabled) void loadDiagnostics(true);
    return () => request.current?.abort();
  }, [enabled, loadDiagnostics]);

  const selectedCategory = catalog.find((entry) => entry.key === planValue)?.category;

  const savePlan = async () => {
    if (
      savingPlan ||
      isCustomerSyncing() ||
      checkingPolar ||
      diagnosticsLoading ||
      !plan ||
      !["none", "not_configured"].includes(plan.polar_subscription_verification ?? "") ||
      catalogError ||
      !catalog.some(
        (entry) => entry.key === planValue && entry.key !== "business_legacy" && !entry.require_payment_method,
      )
    )
      return;
    if (selectedCategory === "trial" && !isFutureUtcTrialEnd(trialEndsOn)) {
      return;
    }
    setSavingPlan(true);
    try {
      applyPlan(await putOrganizationBillingPlan(orgId, billingPlanRequest(planValue, selectedCategory, trialEndsOn)));
      await loadDiagnostics();
      showSuccessToast("Billing plan updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to set billing plan");
    } finally {
      setSavingPlan(false);
    }
  };

  return {
    catalog,
    catalogError,
    plan,
    planValue,
    setPlanValue,
    trialEndsOn,
    setTrialEndsOn,
    trialEndInvalid: selectedCategory === "trial" && !isFutureUtcTrialEnd(trialEndsOn),
    savingPlan,
    savePlan,
    applyPlan,
    diagnosticsLoading,
    diagnosticsError,
    checkingPolar,
    loadDiagnostics,
    ...customerSync,
  };
}

function useOrgMarkupOverride(orgId: string, setCredit: (credit: OrganizationLLMCredit) => void) {
  const [markupPercent, setMarkupPercent] = useState("");
  const [savingMarkup, setSavingMarkup] = useState(false);

  const resetMarkup = useCallback((data: OrganizationLLMCredit) => {
    setMarkupPercent(data.markup_override_bps == null ? "" : bpsToPercentInput(data.markup_override_bps));
  }, []);

  const saveMarkup = async () => {
    setSavingMarkup(true);
    try {
      const body =
        markupPercent.trim() === "" ? { markup_bps: null } : { markup_bps: percentInputToBps(markupPercent) };
      const response = await fetch(`/admin/api/organizations/${orgId}/llm-settings`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response, "Failed to update markup override"));
      }
      const next = (await response.json()) as OrganizationLLMCredit;
      setCredit(next);
      resetMarkup(next);
      showSuccessToast("Markup override updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update markup override");
    } finally {
      setSavingMarkup(false);
    }
  };

  return { markupPercent, setMarkupPercent, savingMarkup, saveMarkup, resetMarkup };
}

export function useOrgLLMCredit(orgId: string, options: { billingEnabled: boolean }) {
  const [credit, setCredit] = useState<OrganizationLLMCredit | null>(null);
  const [grants, setGrants] = useState<OrganizationsOrganizationCreditGrant[]>([]);
  const [grantsLoadFailed, setGrantsLoadFailed] = useState(false);
  const [loading, setLoading] = useState(true);
  const [balanceInputs, setBalanceInputs] = useState<CreditBalanceInputs>(EMPTY_BALANCE_INPUTS);
  const [note, setNote] = useState("");
  const [savingBalance, setSavingBalance] = useState<CreditBalanceBucket | null>(null);
  const billingPlan = useOrgBillingPlan(orgId, options.billingEnabled);
  const markup = useOrgMarkupOverride(orgId, setCredit);
  const { resetMarkup } = markup;

  const applyCredit = useCallback(
    (data: OrganizationLLMCredit) => {
      setCredit(data);
      resetMarkup(data);
    },
    [resetMarkup],
  );

  const applyCreditAndBalances = useCallback(
    (data: OrganizationLLMCredit) => {
      applyCredit(data);
      setBalanceInputs(balanceInputsFromCredit(data));
    },
    [applyCredit],
  );

  const loadGrants = useCallback(async () => {
    try {
      setGrants(await fetchOrganizationCreditGrants(orgId));
      setGrantsLoadFailed(false);
    } catch (error) {
      setGrantsLoadFailed(true);
      showErrorToast(error instanceof Error ? error.message : "Failed to load credit history");
    }
  }, [orgId]);

  const loadCredit = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([loadOrganizationCreditState(orgId, applyCreditAndBalances), loadGrants()]);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load organization credit");
    } finally {
      setLoading(false);
    }
  }, [applyCreditAndBalances, loadGrants, orgId]);

  useEffect(() => {
    loadCredit();
  }, [loadCredit]);

  const setBalanceInput = useCallback((bucket: CreditBalanceBucket, value: string) => {
    setBalanceInputs((current) => ({ ...current, [bucket]: value }));
  }, []);

  const saveBalance = async (bucket: CreditBalanceBucket) => {
    const targetCents = parseDollarInputToCents(balanceInputs[bucket]);
    if (!credit || targetCents === null) {
      return;
    }

    setSavingBalance(bucket);
    try {
      const next = await putOrganizationCreditBalance(orgId, {
        bucket,
        target_cents: targetCents,
        expected_remaining_cents: creditRemainingCents(credit, bucket),
        note: note.trim(),
      });
      applyCredit(next);
      setBalanceInput(bucket, centsToDollarInput(creditRemainingCents(next, bucket)));
      setNote("");
      await loadGrants();
      showSuccessToast("Credit balance updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to set credit balance");
      if (error instanceof CreditBalanceChangedError) {
        await loadCredit();
      }
    } finally {
      setSavingBalance(null);
    }
  };

  return {
    ...billingPlan,
    ...markup,
    credit,
    grants,
    grantsLoadFailed,
    reloadGrants: loadGrants,
    loading,
    balanceInputs,
    setBalanceInput,
    note,
    setNote,
    savingBalance,
    saveBalance,
  };
}
