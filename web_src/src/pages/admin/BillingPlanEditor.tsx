import { Link } from "react-router";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Checkbox } from "@/components/ui/checkbox";
import { bpsToPercentInput, centsToDollarInput, dollarInputToCents, percentInputToBps } from "@/lib/hostedCredit";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { BillingHelp } from "./BillingHelp";
import { PolarProductSelect } from "./PolarProductSelect";
import type { BillingPlan, FeatureDefinition } from "./billingPlansApi";
type PlanFieldsProps = { plan: BillingPlan; onChange: (plan: BillingPlan) => void };
type PlanPricingProps = PlanFieldsProps & { assigned: boolean };

export function BillingPlanEditor({
  plan,
  assigned,
  features,
  onChange,
  onSave,
  onCancel,
  saving,
  error,
}: {
  plan: BillingPlan;
  assigned: boolean;
  features: FeatureDefinition[];
  onChange: (plan: BillingPlan) => void;
  onSave: () => void;
  onCancel: () => void;
  saving: boolean;
  error: string;
}) {
  const [returnFocus] = useState(() => document.activeElement);
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !saving) onCancel();
      }}
    >
      <DialogContent
        size="large"
        className="flex max-h-[90vh] w-[calc(100%-2rem)] max-w-3xl flex-col gap-0 overflow-hidden p-0"
        showCloseButton={!saving}
        onCloseAutoFocus={(event) => {
          if (returnFocus instanceof HTMLElement) {
            event.preventDefault();
            returnFocus.focus();
          }
        }}
      >
        <DialogHeader className="border-b px-6 py-5">
          <DialogTitle>{plan.id ? `Edit ${plan.name}` : "Create plan"}</DialogTitle>
          <DialogDescription>Configure pricing, limits, credits, and feature access.</DialogDescription>
        </DialogHeader>
        <form
          className="flex min-h-0 flex-1 flex-col overflow-hidden"
          onSubmit={(event) => {
            event.preventDefault();
            onSave();
          }}
        >
          <div className="min-h-0 flex-1 overflow-y-auto">
            <fieldset disabled={saving} className="space-y-6 p-6">
              <PlanDetails plan={plan} onChange={onChange} />
              <PlanPolicies plan={plan} assigned={assigned} onChange={onChange} />
              <PlanAccessFields plan={plan} features={features} onChange={onChange} />
            </fieldset>
          </div>
          {error && (
            <div role="alert" className="mx-6 mb-4 shrink-0 rounded-lg border p-4 text-sm text-destructive">
              {error}
            </div>
          )}
          <DialogFooter className="shrink-0 border-t bg-muted/20 px-6 py-4">
            <Button type="button" variant="outline" disabled={saving} onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : "Save plan"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function PlanSelect({
  label,
  help,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string;
  help: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5">
        <Label>{label}</Label>
        <BillingHelp label={label} text={help} />
      </div>
      <Select value={value} onValueChange={onChange} disabled={disabled}>
        <SelectTrigger className="w-full capitalize" aria-label={label}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option} value={option} className="capitalize">
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
      <div className="space-y-2 border-t pt-5">
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
      <fieldset className="space-y-3 border-t pt-5">
        <legend className="text-sm font-semibold">Plan features</legend>
        {features.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Create features on the{" "}
            <Link to="/admin/features" className="underline">
              Features page
            </Link>{" "}
            to add them to a plan.
          </p>
        )}
        {features.map((feature) => (
          <Label key={feature.key} className="flex items-center gap-3 rounded-lg border p-3">
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
            <span className="space-y-1">
              <span className="block">{feature.label}</span>
              <span className="block text-xs font-normal text-muted-foreground">
                <code>{feature.key}</code>
                {!feature.gated && " · Available to all organizations"}
              </span>
            </span>
          </Label>
        ))}
      </fieldset>
    </>
  );
}

function PlanDetails({ plan, onChange }: PlanFieldsProps) {
  const update = (patch: Partial<BillingPlan>) => onChange({ ...plan, ...patch });
  return (
    <section className="space-y-4" aria-labelledby="plan-details-heading">
      <h3 id="plan-details-heading" className="text-sm font-semibold">
        Plan details
      </h3>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="plan-name">Name</Label>
          <Input id="plan-name" required value={plan.name} onChange={(event) => update({ name: event.target.value })} />
        </div>
        <div className="space-y-2">
          <div className="flex items-center gap-1.5">
            <Label htmlFor="plan-key">Key</Label>
            <BillingHelp label="Key" text="A unique plan identifier. The key cannot change after creation." />
          </div>
          <Input
            id="plan-key"
            required
            value={plan.key}
            disabled={!!plan.id}
            onChange={(event) => update({ key: event.target.value })}
          />
        </div>
        <PlanSelect
          label="Category"
          help="Choose whether this plan has no billing, provides a trial, or supports paid usage."
          value={plan.category}
          options={["none", "trial", "paid"]}
          disabled={!!plan.id && ["none", "trial", "business_legacy", "business_manual"].includes(plan.key)}
          onChange={(category) => update({ category })}
        />
        <PlanSelect
          label="Visibility"
          help="Public plans appear in the customer plan list. Private plans are assigned by an administrator."
          value={plan.visibility}
          options={["public", "private"]}
          onChange={(visibility) => update({ visibility })}
        />
      </div>
    </section>
  );
}
function PlanPolicies({ plan, assigned, onChange }: PlanPricingProps) {
  const update = (patch: Partial<BillingPlan>) => onChange({ ...plan, ...patch });
  const locked = isPolarLocked(plan, assigned);
  const legacy = plan.key === "business_legacy";
  const manual = plan.key === "business_manual";
  return (
    <>
      {["Pricing", "Usage limits", "Trial"].map((section) => (
        <section key={section} className="space-y-4 border-t pt-5" aria-label={section}>
          <h3 className="text-sm font-semibold">{section}</h3>
          <div className="grid gap-4 sm:grid-cols-2">
            <PlanNumbers plan={plan} assigned={assigned} section={section} onChange={onChange} />
            {section === "Pricing" && (
              <div className="flex items-center gap-3">
                <Switch
                  id="plan-per-seat"
                  checked={plan.per_seat}
                  disabled={locked || legacy}
                  onCheckedChange={(per_seat) =>
                    update({ per_seat, seat_limit: per_seat ? (plan.seat_limit ?? 50) : plan.seat_limit })
                  }
                />
                <Label htmlFor="plan-per-seat">Per seat</Label>
              </div>
            )}
            {section === "Pricing" && (
              <PolarProductSelect
                kind="subscription"
                disabled={locked || manual}
                value={plan.polar_product_id ?? ""}
                onChange={(productID) => update({ polar_product_id: productID || null })}
                description={
                  manual
                    ? "Business Manual does not use Polar products."
                    : locked
                      ? "This plan is linked to Polar. Create a new plan to change the product or price."
                      : undefined
                }
              />
            )}
          </div>
        </section>
      ))}
    </>
  );
}
function PlanNumbers({ plan, assigned, section, onChange }: PlanPricingProps & { section: string }) {
  const update = (patch: Partial<BillingPlan>) => onChange({ ...plan, ...patch });
  const numberFields = planNumberFields(plan, assigned);
  return (
    <>
      {numberFields
        .filter((field) => field.section === section)
        .map(({ key, label, help, nullable, locked, placeholder, unit }) => (
          <div key={key} className="space-y-2">
            <div className="flex items-center gap-1.5">
              <Label htmlFor={`plan-${key}`}>{label}</Label>
              <BillingHelp label={label} text={help} />
            </div>
            <Input
              id={`plan-${key}`}
              type="number"
              min={key === "seat_limit" && plan.per_seat ? 1 : 0}
              max={key === "seat_limit" && plan.per_seat ? 1000 : undefined}
              required={key === "seat_limit" && plan.per_seat}
              step={unit ? 0.01 : 1}
              disabled={locked}
              placeholder={placeholder}
              {...(key === "seat_limit"
                ? { value: plan.seat_limit ?? "" }
                : {
                    defaultValue:
                      plan[key] == null
                        ? ""
                        : unit === "USD"
                          ? centsToDollarInput(Number(plan[key]))
                          : unit === "%"
                            ? bpsToPercentInput(Number(plan[key]))
                            : String(plan[key]),
                  })}
              onChange={(event) =>
                update({
                  [key]:
                    nullable && event.target.value === ""
                      ? null
                      : unit === "USD"
                        ? dollarInputToCents(event.target.value)
                        : unit === "%"
                          ? percentInputToBps(event.target.value)
                          : Number(event.target.value),
                })
              }
            />
          </div>
        ))}
    </>
  );
}
// A Polar link is locked only while an organization uses the plan. The server
// checks assignments again on save.
function isPolarLocked(plan: BillingPlan, assigned: boolean) {
  return !!plan.id && !!plan.polar_product_id && assigned;
}
function planNumberFields(plan: BillingPlan, assigned: boolean) {
  const locked = isPolarLocked(plan, assigned);
  const legacy = plan.key === "business_legacy";
  const perSeat = plan.per_seat;
  const fields: {
    key: keyof BillingPlan;
    label: string;
    help: string;
    nullable?: boolean;
    placeholder?: string;
    locked?: boolean;
    section: string;
    unit?: "USD" | "%";
  }[] = [
    {
      key: "price_cents",
      label: "Price (USD)",
      unit: "USD",
      help: locked
        ? "Pricing is locked because this plan is linked to a Polar product."
        : "The plan price in US dollars.",
      locked: locked || legacy,
      section: "Pricing",
    },
    {
      key: "markup_bps",
      label: "Markup (%)",
      unit: "%",
      help: "The usage markup percentage. Leave empty to use the installation default.",
      nullable: true,
      placeholder: "Installation default",
      section: "Pricing",
    },
    {
      key: "seat_limit",
      label: perSeat ? "Maximum seats" : "Seat limit",
      help: perSeat
        ? "Maximum purchasable seats. Enter an integer from 1 to 1,000."
        : legacy
          ? "Legacy Business seat limit cannot change."
          : "Maximum organization members. Leave empty for unlimited seats.",
      nullable: true,
      placeholder: perSeat ? undefined : "Unlimited",
      locked: legacy,
      section: "Usage limits",
    },
    {
      key: "parallel_task_limit",
      label: "Parallel runner limit",
      help: legacy
        ? "Legacy Business parallel runner limit cannot change."
        : "Maximum reserved or running runner jobs across the organization. Leave empty for unlimited jobs.",
      nullable: true,
      placeholder: "Unlimited",
      locked: legacy,
      section: "Usage limits",
    },
    {
      key: "trial_credit_cents",
      label: "Trial credit (USD)",
      unit: "USD",
      help: "Credit granted for a trial, in US dollars.",
      section: "Trial",
    },
    {
      key: "trial_length_days",
      label: "Trial length (days)",
      help: "Number of days before a trial ends.",
      section: "Trial",
    },
  ];
  return fields;
}
