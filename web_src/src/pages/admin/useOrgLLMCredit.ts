import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { bpsToPercentInput, dollarInputToCents, percentInputToBps } from "@/lib/hostedCredit";
import { useCallback, useEffect, useState } from "react";

export type OrganizationLLMCredit = {
  remaining_credit_cents: number;
  grant_total_cents: number;
  superplane_grant_cents: number;
  purchased_credit_cents: number;
  hosted_billed_cents: number;
  markup_bps: number;
  markup_override_bps: number | null;
  warning: boolean;
};

export type OrganizationBillingPlan = {
  plan: string;
  plan_source: string;
  polar_subscription_status: string;
  trial_ends_at: string | null;
  current_period_end: string | null;
};

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

export function useOrgLLMCredit(orgId: string) {
  const [credit, setCredit] = useState<OrganizationLLMCredit | null>(null);
  const [plan, setPlan] = useState<OrganizationBillingPlan | null>(null);
  const [planValue, setPlanValue] = useState("trial");
  const [loading, setLoading] = useState(true);
  const [grantDollars, setGrantDollars] = useState("");
  const [note, setNote] = useState("");
  const [markupPercent, setMarkupPercent] = useState("");
  const [savingGrant, setSavingGrant] = useState(false);
  const [savingMarkup, setSavingMarkup] = useState(false);
  const [savingPlan, setSavingPlan] = useState(false);

  const applyCredit = useCallback((data: OrganizationLLMCredit) => {
    setCredit(data);
    setMarkupPercent(data.markup_override_bps == null ? "" : bpsToPercentInput(data.markup_override_bps));
  }, []);

  const applyPlan = useCallback((nextPlan: OrganizationBillingPlan) => {
    setPlan(nextPlan);
    setPlanValue(nextPlan.plan || "none");
  }, []);

  const loadCredit = useCallback(async () => {
    setLoading(true);
    try {
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
      }
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load organization credit");
    } finally {
      setLoading(false);
    }
  }, [applyCredit, applyPlan, orgId]);

  useEffect(() => {
    loadCredit();
  }, [loadCredit]);

  const addGrant = async () => {
    setSavingGrant(true);
    try {
      const response = await fetch(`/admin/api/organizations/${orgId}/llm-credit/grants`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({
          amount_cents: dollarInputToCents(grantDollars),
          note: note.trim(),
        }),
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response, "Failed to add hosted credit"));
      }
      applyCredit(await response.json());
      setGrantDollars("");
      setNote("");
      showSuccessToast("Hosted credit increased");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to add hosted credit");
    } finally {
      setSavingGrant(false);
    }
  };

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
      applyCredit(await response.json());
      showSuccessToast("Markup override updated");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to update markup override");
    } finally {
      setSavingMarkup(false);
    }
  };

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

  return {
    credit,
    plan,
    planValue,
    setPlanValue,
    loading,
    grantDollars,
    setGrantDollars,
    note,
    setNote,
    markupPercent,
    setMarkupPercent,
    savingGrant,
    savingMarkup,
    savingPlan,
    addGrant,
    saveMarkup,
    savePlan,
  };
}
