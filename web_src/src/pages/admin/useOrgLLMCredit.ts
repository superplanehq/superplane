import type { OrganizationsOrganizationCreditGrant } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { bpsToPercentInput, centsToDollarInput, parseDollarInputToCents, percentInputToBps } from "@/lib/hostedCredit";
import { useCallback, useEffect, useState } from "react";

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

export type OrganizationBillingPlan = {
  plan: string;
  plan_source: string;
  polar_subscription_status: string;
  polar_managed: boolean;
  trial_ends_at: string | null;
  current_period_end: string | null;
};

export type CreditBalanceBucket = "trial" | "topup" | "grant";

export type CreditBalanceInputs = Record<CreditBalanceBucket, string>;

export const CREDIT_BALANCE_CHANGED_COPY = "The balance changed. Reload and try again.";

const EMPTY_BALANCE_INPUTS: CreditBalanceInputs = { trial: "", topup: "", grant: "" };

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

async function readErrorMessage(response: Response, fallback: string): Promise<string> {
  const text = await response.text();
  if (text.trim() === "") {
    return fallback;
  }
  return text;
}

async function fetchOrganizationBillingPlan(orgId: string): Promise<OrganizationBillingPlan | null> {
  const response = await fetch(`/admin/api/organizations/${orgId}/billing-plan`, { credentials: "include" });
  if (!response.ok) {
    return null;
  }
  return (await response.json()) as OrganizationBillingPlan;
}

async function putOrganizationBillingPlan(orgId: string, plan: string): Promise<OrganizationBillingPlan> {
  const response = await fetch(`/admin/api/organizations/${orgId}/billing-plan`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ plan }),
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
  body: { bucket: CreditBalanceBucket; target_cents: number; expected_remaining_cents: number; note: string },
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

async function loadOrganizationCreditState(
  orgId: string,
  applyCredit: (data: OrganizationLLMCredit) => void,
  applyPlan: (plan: OrganizationBillingPlan) => void,
  clearPlan: () => void,
) {
  const [creditResponse, nextPlan] = await Promise.all([
    fetch(`/admin/api/organizations/${orgId}/llm-credit`, { credentials: "include" }),
    fetchOrganizationBillingPlan(orgId),
  ]);
  if (!creditResponse.ok) {
    throw new Error(await readErrorMessage(creditResponse, "Failed to load organization credit"));
  }
  applyCredit(await creditResponse.json());
  if (nextPlan) {
    applyPlan(nextPlan);
    return;
  }
  clearPlan();
  showErrorToast("Failed to load billing plan");
}

function useOrgBillingPlan(orgId: string) {
  const [plan, setPlan] = useState<OrganizationBillingPlan | null>(null);
  const [planValue, setPlanValue] = useState("trial");
  const [savingPlan, setSavingPlan] = useState(false);

  const applyPlan = useCallback((nextPlan: OrganizationBillingPlan) => {
    setPlan(nextPlan);
    setPlanValue(nextPlan.plan || "none");
  }, []);

  const clearPlan = useCallback(() => setPlan(null), []);

  const savePlan = async () => {
    setSavingPlan(true);
    try {
      applyPlan(await putOrganizationBillingPlan(orgId, planValue));
      showSuccessToast("Billing plan updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to set billing plan");
    } finally {
      setSavingPlan(false);
    }
  };

  return { plan, planValue, setPlanValue, savingPlan, savePlan, applyPlan, clearPlan };
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

export function useOrgLLMCredit(orgId: string) {
  const [credit, setCredit] = useState<OrganizationLLMCredit | null>(null);
  const [grants, setGrants] = useState<OrganizationsOrganizationCreditGrant[]>([]);
  const [loading, setLoading] = useState(true);
  const [balanceInputs, setBalanceInputs] = useState<CreditBalanceInputs>(EMPTY_BALANCE_INPUTS);
  const [note, setNote] = useState("");
  const [savingBalance, setSavingBalance] = useState<CreditBalanceBucket | null>(null);
  const billingPlan = useOrgBillingPlan(orgId);
  const markup = useOrgMarkupOverride(orgId, setCredit);
  const { applyPlan, clearPlan } = billingPlan;
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
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load credit history");
    }
  }, [orgId]);

  const loadCredit = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([
        loadOrganizationCreditState(orgId, applyCreditAndBalances, applyPlan, clearPlan),
        loadGrants(),
      ]);
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load organization credit");
    } finally {
      setLoading(false);
    }
  }, [applyCreditAndBalances, applyPlan, clearPlan, loadGrants, orgId]);

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
    loading,
    balanceInputs,
    setBalanceInput,
    note,
    setNote,
    savingBalance,
    saveBalance,
  };
}
