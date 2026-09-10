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

  const loadCredit = useCallback(async () => {
    setLoading(true);
    try {
      const [creditResponse, planResponse] = await Promise.all([
        fetch(`/admin/api/organizations/${orgId}/llm-credit`, { credentials: "include" }),
        fetch(`/admin/api/organizations/${orgId}/billing-plan`, { credentials: "include" }),
      ]);
      if (!creditResponse.ok) {
        throw new Error(await readErrorMessage(creditResponse, "Failed to load organization credit"));
      }
      applyCredit(await creditResponse.json());
      if (planResponse.ok) {
        const nextPlan = (await planResponse.json()) as OrganizationBillingPlan;
        setPlan(nextPlan);
        setPlanValue(nextPlan.plan || "none");
      }
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to load organization credit");
    } finally {
      setLoading(false);
    }
  }, [applyCredit, orgId]);

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
      const response = await fetch(`/admin/api/organizations/${orgId}/billing-plan`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ plan: planValue }),
      });
      if (!response.ok) {
        throw new Error(await readErrorMessage(response, "Failed to set billing plan"));
      }
      const nextPlan = (await response.json()) as OrganizationBillingPlan;
      setPlan(nextPlan);
      setPlanValue(nextPlan.plan || "none");
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
