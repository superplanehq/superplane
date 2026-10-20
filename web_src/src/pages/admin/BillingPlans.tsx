import { useReportPageReady } from "@/hooks/useReportPageReady";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { showErrorToast } from "@/lib/toast";
import { billingCatalogRequest, type BillingPlan, type FeatureDefinition } from "./billingPlansApi";

import { BillingPlanFeatures } from "./BillingPlanFeatures";

const emptyPlan: BillingPlan = {
  key: "",
  name: "",
  category: "paid",
  visibility: "private",
  require_payment_method: false,
  base_price_cents: 0,
  seat_price_cents: 0,
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
  const [plans, setPlans] = useState<BillingPlan[]>([]);
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
        billingCatalogRequest<BillingPlan[]>("billing-plans"),
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
  const remove = async (plan: BillingPlan) => {
    try {
      await billingCatalogRequest(`billing-plans/${plan.id}`, "DELETE");
      await load();
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to delete billing plan");
    }
  };
  return (
    <div className="space-y-6 p-6">
      <h1 className="text-xl font-semibold">Billing plans</h1>
      <p>Plan limits, features, and markup apply immediately. Credit changes apply to future grants.</p>
      {error && (
        <div role="alert">
          {error} <Button onClick={() => void load()}>Try again</Button>
        </div>
      )}
      {loading && <p>Loading billing plans...</p>}
      {!loading && plans.length === 0 && !error && <p>No billing plans. Create a plan to start.</p>}
      {!loading && !error && <BillingPlanFeatures features={features} onSaved={load} />}
      <Button onClick={() => setEditing({ ...emptyPlan })}>Create plan</Button>
      <div className="space-y-2">
        {plans.map((plan) => (
          <div key={plan.id} className="flex items-center gap-3 rounded border p-3">
            <span className="grow">
              {plan.name} · {plan.key} · {plan.visibility}
            </span>
            <Button variant="outline" onClick={() => setEditing({ ...plan })}>
              Edit {plan.name}
            </Button>
            {!plan.polar_product_id && !["none", "trial", "business_legacy"].includes(plan.key) && (
              <Button variant="outline" onClick={() => void remove(plan)}>
                Delete {plan.name}
              </Button>
            )}
          </div>
        ))}
      </div>
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

function BillingPlanEditor({
  plan,
  features,
  onChange,
  onSave,
  onCancel,
  saving,
}: {
  plan: BillingPlan;
  features: FeatureDefinition[];
  onChange: (plan: BillingPlan) => void;
  onSave: () => void;
  onCancel: () => void;
  saving: boolean;
}) {
  const update = (patch: Partial<BillingPlan>) => onChange({ ...plan, ...patch });
  const linked = !!plan.id && !!plan.polar_product_id;
  const legacy = plan.key === "business_legacy";
  const numbers: { key: keyof BillingPlan; label: string; nullable?: boolean; locked?: boolean }[] = [
    { key: "base_price_cents", label: "Base price (cents)", locked: linked },
    { key: "seat_price_cents", label: "Seat price (cents)", locked: linked },
    { key: "seat_limit", label: "Seat limit", nullable: true },
    { key: "parallel_task_limit", label: "Parallel task limit", nullable: true },
    { key: "credit_amount_cents", label: "Included credit (cents)" },
    { key: "credit_validity_days", label: "Credit validity (days)", nullable: true, locked: legacy },
    { key: "trial_credit_cents", label: "Trial credit (cents)" },
    { key: "trial_length_days", label: "Trial length (days)" },
    { key: "markup_bps", label: "Markup (basis points)", nullable: true },
  ];
  return (
    <section aria-label="Billing plan editor" className="space-y-4 rounded border p-4">
      <p>
        Automatic credit delivery is available only for trials and legacy Business. Other credit policies are stored for
        future delivery.
      </p>
      <p>
        One-time credit is eligible once per organization and plan. Recurring credit follows subscription periods, or
        monthly periods for admin plans.
      </p>
      <p>Leave limits empty for unlimited access. Leave markup empty to use the installation default.</p>
      <div className="grid gap-4 md:grid-cols-2">
        <div>
          <Label htmlFor="plan-key">Key</Label>
          <Input
            id="plan-key"
            value={plan.key}
            disabled={!!plan.id}
            onChange={(event) => update({ key: event.target.value })}
          />
        </div>
        <div>
          <Label htmlFor="plan-name">Name</Label>
          <Input id="plan-name" value={plan.name} onChange={(event) => update({ name: event.target.value })} />
        </div>
        <PlanSelect
          label="Category"
          value={plan.category}
          options={["none", "trial", "paid"]}
          disabled={!!plan.id && ["none", "trial", "business_legacy"].includes(plan.key)}
          onChange={(category) => update({ category })}
        />
        <PlanSelect
          label="Visibility"
          value={plan.visibility}
          options={["public", "private"]}
          onChange={(visibility) => update({ visibility })}
        />
        {numbers.map(({ key, label, nullable, locked }) => (
          <div key={key}>
            <Label htmlFor={`plan-${key}`}>{label}</Label>
            <Input
              id={`plan-${key}`}
              type="number"
              min={0}
              step={1}
              disabled={locked}
              value={String(plan[key] ?? "")}
              onChange={(event) =>
                update({ [key]: nullable && event.target.value === "" ? null : Number(event.target.value) })
              }
            />
          </div>
        ))}
        <div>
          <Label htmlFor="plan-polar-product">Polar product ID</Label>
          <Input
            id="plan-polar-product"
            disabled={linked}
            value={plan.polar_product_id ?? ""}
            onChange={(event) => update({ polar_product_id: event.target.value || null })}
          />
        </div>
        <PlanSelect
          label="Credit frequency"
          value={plan.credit_frequency}
          disabled={legacy}
          options={["none", "once", "billing_period"]}
          onChange={(credit_frequency) => update({ credit_frequency })}
        />
        <PlanSelect
          label="Credit expiration"
          value={plan.credit_expiration}
          disabled={legacy}
          options={["period_end", "after_days", "never"]}
          onChange={(credit_expiration) => update({ credit_expiration })}
        />
      </div>
      <PlanAccessFields plan={plan} features={features} onChange={onChange} />
      <div className="flex gap-2">
        <Button disabled={saving} onClick={onSave}>
          {saving ? "Saving..." : "Save plan"}
        </Button>
        <Button variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </section>
  );
}

function PlanSelect({
  label,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <div>
      <Label>{label}</Label>
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger aria-label={label}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option} value={option}>
              {option.replace(/_/g, " ")}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function PlanAccessFields({
  plan,
  features,
  onChange,
}: {
  plan: BillingPlan;
  features: FeatureDefinition[];
  onChange: (plan: BillingPlan) => void;
}) {
  const update = (patch: Partial<BillingPlan>) => onChange({ ...plan, ...patch });
  return (
    <>
      <div className="space-y-2">
        <Label className="flex items-center gap-2">
          <Checkbox
            checked={plan.require_payment_method}
            onChange={(event) => update({ require_payment_method: event.target.checked })}
          />
          Require payment method
        </Label>
        <p className="text-sm text-muted-foreground">
          Require a saved card before usage. Card setup does not charge the customer.
        </p>
      </div>
      <fieldset className="space-y-2">
        <legend>Plan features</legend>
        {features.map((feature) => (
          <Label key={feature.key} className="flex items-center gap-2">
            <Checkbox
              checked={plan.features.includes(feature.key)}
              onChange={(event) =>
                update({
                  features: event.target.checked
                    ? [...plan.features, feature.key]
                    : plan.features.filter((key) => key !== feature.key),
                })
              }
            />
            {feature.label} · <code>{feature.key}</code>
            {!feature.gated && " (available to all organizations)"}
          </Label>
        ))}
      </fieldset>
    </>
  );
}
