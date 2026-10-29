import type { BillingPlan } from "./billingPlansApi";
import { utcCalendarDate, type OrganizationBillingPlan } from "./useOrgLLMCredit";
import { Text } from "@/components/Text/text";
import { Input } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { AlertTriangle, ExternalLink } from "lucide-react";

export const ADMIN_POLAR_MANAGED_PLAN_COPY =
  "This organization uses Polar for billing. Cancel or change the subscription in Polar.";
export const ADMIN_LOCAL_PLAN_COPY = "Select a plan without a payment requirement. Manage paid subscriptions in Polar.";
export const ADMIN_PLAN_UNKNOWN_COPY = "SuperPlane could not load the billing plan. Refresh the page and try again.";
export const ADMIN_TRIAL_ENDS_LABEL = "Trial ends";
export const ADMIN_TRIAL_ENDS_HELP = "Trial credit stays usable until this date. Dates use UTC.";
export const ADMIN_TRIAL_ENDS_ERROR = "Choose a future date.";

export function OrgBillingPlanField(args: {
  catalog: BillingPlan[];
  catalogError: string;
  polarManaged: boolean;
  planKnown: boolean;
  savedPlan: OrganizationBillingPlan | null;
  planValue: string;
  setPlanValue: (value: string) => void;
  trialEndsOn: string;
  setTrialEndsOn: (value: string) => void;
  trialEndInvalid: boolean;
  savingPlan: boolean;
  busy?: boolean;
  savePlan: () => void;
}) {
  const { editable: planEditable, help } = billingPlanState(args.planKnown, args.polarManaged, args.savedPlan);
  const catalogLocked = !!args.catalogError || args.catalog.length === 0;
  const controlsBusy = args.savingPlan || args.busy;
  const selectedCategory = args.catalog.find((plan) => plan.key === args.planValue)?.category;
  const showTrialEnd = planEditable && selectedCategory === "trial";
  return (
    <div className="mb-5 space-y-4 border-b border-gray-200 pb-5 dark:border-gray-700">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0 space-y-2">
          <Label htmlFor="admin-org-billing-plan" className="block text-left">
            Billing plan
          </Label>
          {args.catalogError && <p role="alert">{args.catalogError}</p>}
          <div className="flex flex-wrap items-center gap-2">
            <div className="w-64 max-w-full">
              <Select
                disabled={!planEditable || catalogLocked || controlsBusy}
                value={args.planValue}
                onValueChange={args.setPlanValue}
              >
                <SelectTrigger id="admin-org-billing-plan" data-testid="admin-org-billing-plan" className="w-full">
                  <SelectValue placeholder="Select a plan" />
                </SelectTrigger>
                <SelectContent>
                  {args.catalog.map((plan) => (
                    <SelectItem
                      key={plan.key}
                      value={plan.key}
                      disabled={plan.key === "business_legacy" || plan.require_payment_method}
                    >
                      {plan.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {planEditable ? (
              <Button
                type="button"
                data-testid="admin-org-billing-plan-save"
                onClick={args.savePlan}
                disabled={
                  controlsBusy ||
                  catalogLocked ||
                  args.catalog.some(
                    (plan) =>
                      plan.key === args.planValue && (plan.key === "business_legacy" || plan.require_payment_method),
                  )
                }
              >
                {args.savingPlan ? "Saving..." : "Save plan"}
              </Button>
            ) : null}
          </div>
          <Text className="max-w-lg text-xs text-gray-500 dark:text-gray-400">{help}</Text>
        </div>
        <OrgSubscriptionDetails plan={args.savedPlan} />
      </div>
      <OrgSubscriptionWarning plan={args.savedPlan} catalog={args.catalog} />
      {showTrialEnd ? (
        <div className="max-w-sm">
          <Label className="mb-2 block text-left" htmlFor="admin-org-trial-ends-input">
            {ADMIN_TRIAL_ENDS_LABEL}
          </Label>
          <Input
            id="admin-org-trial-ends-input"
            type="date"
            data-testid="admin-org-trial-ends-input"
            value={args.trialEndsOn}
            min={utcCalendarDate(new Date())}
            aria-invalid={args.trialEndInvalid}
            disabled={controlsBusy}
            onChange={(event) => args.setTrialEndsOn(event.target.value)}
          />
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">{ADMIN_TRIAL_ENDS_HELP}</Text>
          {args.trialEndInvalid ? (
            <Text className="mt-1 text-xs text-red-600 dark:text-red-400">{ADMIN_TRIAL_ENDS_ERROR}</Text>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function OrgSubscriptionDetails({ plan }: { plan: OrganizationBillingPlan | null }) {
  if (!plan?.polar_subscription_id) return null;
  return (
    <div className="min-w-0 space-y-1 text-xs lg:shrink-0 lg:text-right">
      <p className="font-medium text-gray-600 dark:text-gray-300">Polar subscription</p>
      <code className="block break-all text-gray-500 dark:text-gray-400">{plan.polar_subscription_id}</code>
      {plan.polar_subscription_url ? (
        <Button variant="link" size="sm" className="h-auto gap-1 p-0 text-xs" asChild>
          <a href={plan.polar_subscription_url} target="_blank" rel="noopener noreferrer">
            View subscription in Polar
            <ExternalLink className="size-3" aria-hidden="true" />
          </a>
        </Button>
      ) : null}
    </div>
  );
}

function OrgSubscriptionWarning({ plan, catalog }: { plan: OrganizationBillingPlan | null; catalog: BillingPlan[] }) {
  const savedCatalog = catalog.find((entry) => entry.key === plan?.catalog_key);
  const requiresSubscription = Boolean(savedCatalog?.polar_product_id) || (savedCatalog?.price_cents ?? 0) > 0;
  if (plan?.polar_subscription_verification !== "none" || !requiresSubscription) return null;
  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200"
    >
      <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
      <div className="space-y-1">
        <p className="font-medium">No active Polar subscription</p>
        <p className="text-xs leading-relaxed">
          This plan requires a subscription, but Polar reports none. Check Polar or assign a plan that does not require
          a subscription.
        </p>
      </div>
    </div>
  );
}

function billingPlanState(
  planKnown: boolean,
  polarManaged: boolean,
  savedPlan: OrganizationBillingPlan | null,
): { editable: boolean; help: string } {
  const verification = savedPlan?.polar_subscription_verification;
  if (!planKnown) {
    return { editable: false, help: ADMIN_PLAN_UNKNOWN_COPY };
  }
  if (verification === "not_checked") {
    return { editable: false, help: "Select Check Polar to verify subscriptions before changing the plan." };
  }
  if (verification === "failed") {
    return {
      editable: false,
      help: "SuperPlane could not verify the Polar subscription. Select Check Polar to try again.",
    };
  }
  if (verification === "none") {
    return { editable: true, help: ADMIN_LOCAL_PLAN_COPY };
  }
  if (verification === "active" || polarManaged) {
    return { editable: false, help: ADMIN_POLAR_MANAGED_PLAN_COPY };
  }
  return { editable: true, help: ADMIN_LOCAL_PLAN_COPY };
}
