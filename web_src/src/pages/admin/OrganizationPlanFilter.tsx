import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { billingCatalogRequest, type BillingPlan } from "./billingPlansApi";

type PlanOption = Pick<BillingPlan, "key" | "name">;

const ALL_PLANS = "*";

export function OrganizationPlanFilter({
  planKey,
  onPlanChange,
}: {
  planKey: string;
  onPlanChange: (key: string) => void;
}) {
  const [plans, setPlans] = useState<PlanOption[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);

  const loadPlans = async () => {
    setLoading(true);
    setFailed(false);
    try {
      const catalog = await billingCatalogRequest<PlanOption[]>("billing-plans");
      setPlans(catalog);
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Select
      value={planKey || ALL_PLANS}
      onValueChange={(value) => onPlanChange(value === ALL_PLANS ? "" : value)}
      onOpenChange={(open) => {
        if (open && plans === null && !loading) void loadPlans();
      }}
    >
      <SelectTrigger aria-label="Filter by plan" className="w-48">
        <SelectValue placeholder="All plans" />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ALL_PLANS}>All plans</SelectItem>
        {plans?.map((plan) => (
          <SelectItem key={plan.key} value={plan.key}>
            {plan.name}
          </SelectItem>
        ))}
        {loading ? <div className="px-2 py-1.5 text-sm text-muted-foreground">Loading plans...</div> : null}
        {failed ? (
          <div className="space-y-2 p-2">
            <p role="alert" className="text-sm text-red-600 dark:text-red-400">
              Could not load plans.
            </p>
            <Button type="button" variant="outline" size="sm" onClick={() => void loadPlans()}>
              Try again
            </Button>
          </div>
        ) : null}
      </SelectContent>
    </Select>
  );
}
