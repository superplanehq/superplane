import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { bpsToPercentInput, dollarInputToCents } from "@/lib/hostedCredit";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";
import { Wallet } from "lucide-react";

import { useOrgLLMCredit, type OrganizationLLMCredit } from "./useOrgLLMCredit";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export const ADMIN_POLAR_MANAGED_PLAN_COPY =
  "This organization uses Polar for billing. Cancel or change the subscription in Polar.";
export const ADMIN_LOCAL_PLAN_COPY = "No Polar subscription. Set Trial, Business, or None for this organization.";
export const ADMIN_PLAN_UNKNOWN_COPY = "SuperPlane could not load the billing plan. Refresh the page and try again.";

export function OrgLLMCreditSection({ orgId }: { orgId: string }) {
  const credit = useOrgLLMCredit(orgId);

  return (
    <div className="mb-8">
      <div className="flex items-center gap-2 mb-3">
        <Wallet size={16} className="text-gray-600 dark:text-gray-400" />
        <Heading level={2} className="text-gray-800 text-base dark:text-gray-100">
          Hosted credit
        </Heading>
      </div>
      {credit.loading && !credit.credit ? (
        <Text className="text-gray-500 text-sm dark:text-gray-400">Loading hosted credit...</Text>
      ) : credit.credit ? (
        <OrgHostedCreditCard
          {...credit}
          credit={credit.credit}
          polarManaged={credit.plan?.polar_managed === true}
          planKnown={credit.plan != null}
        />
      ) : null}
    </div>
  );
}

function OrgBillingPlanField(args: {
  polarManaged: boolean;
  planKnown: boolean;
  planValue: string;
  setPlanValue: (value: string) => void;
  savingPlan: boolean;
  savePlan: () => void;
}) {
  const planLocked = !args.planKnown || args.polarManaged;
  return (
    <div className="mb-4 max-w-sm">
      <Label className="mb-2 block text-left">Billing plan</Label>
      <Select disabled={planLocked} value={args.planValue} onValueChange={args.setPlanValue}>
        <SelectTrigger data-testid="admin-org-billing-plan">
          <SelectValue placeholder="Select a plan" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="trial">Trial</SelectItem>
          <SelectItem value="business">Business</SelectItem>
          <SelectItem value="none">None</SelectItem>
        </SelectContent>
      </Select>
      <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {billingPlanHelp(args.planKnown, args.polarManaged)}
      </Text>
      {planLocked ? null : (
        <Button
          type="button"
          className="mt-3"
          data-testid="admin-org-billing-plan-save"
          onClick={args.savePlan}
          disabled={args.savingPlan}
        >
          {args.savingPlan ? "Saving..." : "Save plan"}
        </Button>
      )}
    </div>
  );
}

function billingPlanHelp(planKnown: boolean, polarManaged: boolean): string {
  if (!planKnown) {
    return ADMIN_PLAN_UNKNOWN_COPY;
  }
  if (polarManaged) {
    return ADMIN_POLAR_MANAGED_PLAN_COPY;
  }
  return ADMIN_LOCAL_PLAN_COPY;
}

function OrgHostedCreditCard(args: {
  credit: OrganizationLLMCredit;
  polarManaged: boolean;
  planKnown: boolean;
  grantDollars: string;
  setGrantDollars: (value: string) => void;
  note: string;
  setNote: (value: string) => void;
  markupPercent: string;
  setMarkupPercent: (value: string) => void;
  planValue: string;
  setPlanValue: (value: string) => void;
  savingGrant: boolean;
  savingMarkup: boolean;
  savingPlan: boolean;
  addGrant: () => void;
  saveMarkup: () => void;
  savePlan: () => void;
}) {
  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 p-4 dark:bg-gray-900 dark:outline-gray-700/70">
      <OrgBillingPlanField
        polarManaged={args.polarManaged}
        planKnown={args.planKnown}
        planValue={args.planValue}
        setPlanValue={args.setPlanValue}
        savingPlan={args.savingPlan}
        savePlan={args.savePlan}
      />
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <CreditMetric label="Remaining hosted credit" value={formatUsdCents(args.credit.remaining_credit_cents)} />
        <CreditMetric label="SuperPlane grant" value={formatUsdCents(args.credit.superplane_grant_cents)} />
        <CreditMetric label="Purchased hosted credit" value={formatUsdCents(args.credit.purchased_credit_cents)} />
        <CreditMetric label="Hosted billed spend" value={formatUsdCents(args.credit.hosted_billed_cents)} />
      </div>
      {args.credit.warning ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Remaining hosted credit is at or below the warning threshold.
        </Text>
      ) : null}
      {args.credit.remaining_credit_cents <= 0 ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Hosted credit is empty. Increase credit to restore SuperPlane-hosted runs.
        </Text>
      ) : null}
      <div className="mt-6 grid gap-4 md:grid-cols-2">
        <div>
          <Label className="mb-2 block text-left">Increase credit (USD)</Label>
          <InputGroup>
            <Input
              data-testid="admin-org-credit-amount"
              value={args.grantDollars}
              onChange={(event) => args.setGrantDollars(event.target.value)}
              placeholder="10.00"
            />
          </InputGroup>
          <Label className="mb-2 mt-3 block text-left">Note (optional)</Label>
          <Textarea
            data-testid="admin-org-credit-note"
            value={args.note}
            onChange={(event) => args.setNote(event.target.value)}
            placeholder="Reason for this grant"
          />
          <Button
            type="button"
            className="mt-3"
            data-testid="admin-org-credit-grant"
            onClick={args.addGrant}
            disabled={args.savingGrant || dollarInputToCents(args.grantDollars) <= 0}
          >
            {args.savingGrant ? "Adding..." : "Increase credit"}
          </Button>
        </div>
        <div>
          <Label className="mb-2 block text-left">Markup override percent</Label>
          <InputGroup>
            <Input
              data-testid="admin-org-markup-override"
              value={args.markupPercent}
              onChange={(event) => args.setMarkupPercent(event.target.value)}
              placeholder={`Installation default (${bpsToPercentInput(args.credit.markup_bps)})`}
            />
          </InputGroup>
          <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">
            Leave empty to use the installation markup. Organization members cannot see this value.
          </Text>
          <Button
            type="button"
            className="mt-3"
            data-testid="admin-org-markup-save"
            onClick={args.saveMarkup}
            disabled={args.savingMarkup}
          >
            {args.savingMarkup ? "Saving..." : "Save markup override"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function CreditMetric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
      <p className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100">{value}</p>
    </div>
  );
}
