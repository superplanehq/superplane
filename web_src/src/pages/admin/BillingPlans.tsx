import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useEffect, useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { showErrorToast } from "@/lib/toast";
import {
  billingCatalogRequest,
  toEditableBillingPlan,
  type BillingPlan,
  type BillingPlanListItem,
  type FeatureDefinition,
} from "./billingPlansApi";
import { BillingPlanEditor } from "./BillingPlanEditor";
import { BillingPlansTable } from "./BillingPlansTable";

const emptyPlan: BillingPlan = {
  key: "",
  name: "",
  category: "paid",
  visibility: "private",
  require_payment_method: false,
  price_cents: 0,
  per_seat: false,
  polar_product_id: null,
  seat_limit: null,
  parallel_task_limit: null,
  credit_amount_cents: 0,
  credit_frequency: "none",
  credit_expiration: "never",
  credit_validity_days: null,
  trial_credit_cents: 0,
  trial_length_days: 14,
  markup_bps: null,
  features: [],
};

export function BillingPlans() {
  const [plans, setPlans] = useState<BillingPlanListItem[]>([]);
  const [features, setFeatures] = useState<FeatureDefinition[]>([]);
  const [editing, setEditing] = useState<BillingPlan | null>(null);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  useReportPageReady(!loading, { failed: !!error });
  const load = async () => {
    setLoading(true);
    try {
      const [catalog, definitions] = await Promise.all([
        billingCatalogRequest<BillingPlanListItem[]>("billing-plans"),
        billingCatalogRequest<FeatureDefinition[]>("feature-definitions"),
      ]);
      setPlans(catalog);
      setFeatures(definitions);
      setError("");
    } catch (error) {
      setError(error instanceof Error ? error.message : "Failed to load billing plans");
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => {
    void load();
  }, []);
  const save = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      await billingCatalogRequest(
        `billing-plans${editing.id ? `/${editing.id}` : ""}`,
        editing.id ? "PUT" : "POST",
        editing,
      );
      setEditing(null);
      await load();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to save billing plan");
    } finally {
      setSaving(false);
    }
  };
  const remove = async (plan: BillingPlanListItem) => {
    try {
      await billingCatalogRequest(`billing-plans/${plan.id}`, "DELETE");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to delete billing plan");
    }
    // The server is authoritative. Reload so each row shows its current protection status.
    await load();
  };
  return (
    <div className="space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-xl font-semibold">Billing plans</h1>
        <p className="text-sm text-muted-foreground">Manage plan pricing, usage limits, and feature access.</p>
      </header>
      {error && (
        <div role="alert" className="flex items-center justify-between rounded-lg border p-4 text-sm">
          {error}{" "}
          <Button variant="outline" onClick={() => void load()}>
            Try again
          </Button>
        </div>
      )}
      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="space-y-1">
            <h2 className="font-semibold">Plans</h2>
            <p className="text-sm text-muted-foreground">Set the pricing and access available to each organization.</p>
          </div>
          <Button disabled={loading || !!error} onClick={() => setEditing({ ...emptyPlan })}>
            <Plus aria-hidden="true" />
            Create plan
          </Button>
        </div>
        {loading ? (
          <p className="py-12 text-center text-sm text-muted-foreground">Loading billing plans...</p>
        ) : (
          !error && (
            <BillingPlansTable
              plans={plans}
              onEdit={(plan) => setEditing(toEditableBillingPlan(plan))}
              onDelete={(plan) => void remove(plan)}
            />
          )
        )}
      </section>

      {editing && (
        <BillingPlanEditor
          plan={editing}
          features={features}
          onChange={setEditing}
          onSave={() => void save()}
          onCancel={() => setEditing(null)}
          saving={saving}
        />
      )}
    </div>
  );
}
