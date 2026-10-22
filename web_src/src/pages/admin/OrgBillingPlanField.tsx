import type { BillingPlan } from "./billingPlansApi";
import { utcCalendarDate, type OrganizationBillingPlan } from "./useOrgLLMCredit";
import { Text } from "@/components/Text/text";
import { Input } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

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
  savePlan: () => void;
}) {
  const { editable: planEditable, help } = billingPlanState(args.planKnown, args.polarManaged, args.savedPlan);
  const catalogLocked = !!args.catalogError || args.catalog.length === 0;
  const selectedCategory = args.catalog.find((plan) => plan.key === args.planValue)?.category;
  const showTrialEnd = planEditable && selectedCategory === "trial";
  return (
    <div className="mb-4 max-w-sm">
      <Label className="mb-2 block text-left">Billing plan</Label>
      {args.catalogError && <p role="alert">{args.catalogError}</p>}
      <Select disabled={!planEditable || catalogLocked} value={args.planValue} onValueChange={args.setPlanValue}>
        <SelectTrigger data-testid="admin-org-billing-plan">
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
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">{help}</Text>
      {args.savedPlan?.polar_subscription_id ? (
        <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
          <span>{args.savedPlan.polar_subscription_id}</span>
          {args.savedPlan.polar_subscription_url ? (
            <Button variant="link" size="sm" asChild>
              <a href={args.savedPlan.polar_subscription_url} target="_blank" rel="noopener noreferrer">
                View subscription in Polar
              </a>
            </Button>
          ) : null}
        </div>
      ) : null}
      {showTrialEnd ? (
        <div className="mt-3">
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
            onChange={(event) => args.setTrialEndsOn(event.target.value)}
          />
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">{ADMIN_TRIAL_ENDS_HELP}</Text>
          {args.trialEndInvalid ? (
            <Text className="mt-1 text-xs text-red-600 dark:text-red-400">{ADMIN_TRIAL_ENDS_ERROR}</Text>
          ) : null}
        </div>
      ) : null}
      {planEditable ? (
        <Button
          type="button"
          className="mt-3"
          data-testid="admin-org-billing-plan-save"
          onClick={args.savePlan}
          disabled={
            args.savingPlan ||
            catalogLocked ||
            args.catalog.some(
              (plan) => plan.key === args.planValue && (plan.key === "business_legacy" || plan.require_payment_method),
            )
          }
        >
          {args.savingPlan ? "Saving..." : "Save plan"}
        </Button>
      ) : null}
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
  if (verification === "failed") {
    return { editable: false, help: "SuperPlane could not verify the Polar subscription. Refresh and try again." };
  }
  if (verification === "none") {
    return {
      editable: true,
      help: polarManaged
        ? "SuperPlane shows an active Polar subscription, but Polar reports none. You can assign a plan manually."
        : ADMIN_LOCAL_PLAN_COPY,
    };
  }
  if (verification === "active" || polarManaged) {
    return { editable: false, help: ADMIN_POLAR_MANAGED_PLAN_COPY };
  }
  return { editable: true, help: ADMIN_LOCAL_PLAN_COPY };
}
