import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input, InputGroup } from "@/components/Input/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { bpsToPercentInput, parseDollarInputToCents } from "@/lib/hostedCredit";
import { formatCreditGrantAmount } from "@/pages/factories/lib/hostedCreditGrants";
import { formatUsdCents } from "@/pages/factories/lib/workOrderUsage";
import { CreditGrantTable } from "@/pages/factories/pages/organizationSettings/BillingCreditHistoryCard";
import type { OrganizationsOrganizationCreditGrant } from "@/api-client";
import { Wallet } from "lucide-react";

import {
  creditRemainingCents,
  formatUtcTrialEnd,
  useOrgLLMCredit,
  utcCalendarDate,
  type CreditBalanceBucket,
  type CreditBalanceInputs,
  type OrganizationLLMCredit,
} from "./useOrgLLMCredit";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export const ADMIN_POLAR_MANAGED_PLAN_COPY =
  "This organization uses Polar for billing. Cancel or change the subscription in Polar.";
export const ADMIN_LOCAL_PLAN_COPY = "No Polar subscription. Set Trial, Business, or None for this organization.";
export const ADMIN_PLAN_UNKNOWN_COPY = "SuperPlane could not load the billing plan. Refresh the page and try again.";
export const ADMIN_SET_BALANCE_HELP_COPY =
  "Enter the new remaining balance. SuperPlane adds the difference to the credit history as an adjustment.";
export const ADMIN_NO_ACTIVE_TRIAL_COPY = "This organization has no active trial credit.";
export const ADMIN_INCLUDED_USAGE_COPY = "The Business subscription sets this value.";
export const ADMIN_TRIAL_ENDS_LABEL = "Trial ends";
export const ADMIN_TRIAL_ENDS_HELP = "Trial credit stays usable until this date. Dates use UTC.";
export const ADMIN_TRIAL_ENDS_ERROR = "Choose a future date.";
export const ADMIN_CREDIT_HISTORY_ERROR_COPY = "SuperPlane could not load the credit history.";

const EDITABLE_BALANCES: Array<{ bucket: CreditBalanceBucket; label: string; action: string }> = [
  { bucket: "trial", label: "Trial credit", action: "Set trial credit" },
  { bucket: "topup", label: "Top-up credit", action: "Set top-up credit" },
  { bucket: "grant", label: "SuperPlane grant", action: "Set SuperPlane grant" },
];

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
          savedTrialEndsAt={credit.plan?.trial_ends_at ?? null}
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
  savedTrialEndsAt: string | null;
  trialEndsOn: string;
  setTrialEndsOn: (value: string) => void;
  trialEndInvalid: boolean;
  savingPlan: boolean;
  savePlan: () => void;
}) {
  const planLocked = !args.planKnown || args.polarManaged;
  const showTrialEndField = args.planKnown && !args.polarManaged && args.planValue === "trial";
  const savedTrialEnd = formatUtcTrialEnd(args.savedTrialEndsAt);
  const showTrialEnd = args.planKnown && !args.polarManaged && (showTrialEndField || savedTrialEnd != null);
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
      {showTrialEnd ? (
        <div className="mt-3">
          <Label className="mb-2 block text-left" htmlFor="admin-org-trial-ends-input">
            {ADMIN_TRIAL_ENDS_LABEL}
          </Label>
          {showTrialEndField ? (
            <>
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
            </>
          ) : (
            <Text data-testid="admin-org-trial-ends" className="text-sm text-gray-900 dark:text-gray-100">
              {savedTrialEnd}
            </Text>
          )}
        </div>
      ) : null}
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
  grants: OrganizationsOrganizationCreditGrant[];
  polarManaged: boolean;
  planKnown: boolean;
  balanceInputs: CreditBalanceInputs;
  setBalanceInput: (bucket: CreditBalanceBucket, value: string) => void;
  note: string;
  setNote: (value: string) => void;
  markupPercent: string;
  setMarkupPercent: (value: string) => void;
  planValue: string;
  setPlanValue: (value: string) => void;
  savingBalance: CreditBalanceBucket | null;
  savedTrialEndsAt: string | null;
  trialEndsOn: string;
  setTrialEndsOn: (value: string) => void;
  trialEndInvalid: boolean;
  grantsLoadFailed: boolean;
  reloadGrants: () => void;
  savingMarkup: boolean;
  savingPlan: boolean;
  saveBalance: (bucket: CreditBalanceBucket) => void;
  saveMarkup: () => void;
  savePlan: () => void;
}) {
  const trialActive = isTrialCreditActive(args.credit.welcome_credit_expires_at);

  return (
    <div className="bg-white rounded-md shadow-sm outline outline-slate-950/10 p-4 dark:bg-gray-900 dark:outline-gray-700/70">
      <OrgBillingPlanField
        polarManaged={args.polarManaged}
        planKnown={args.planKnown}
        planValue={args.planValue}
        setPlanValue={args.setPlanValue}
        savedTrialEndsAt={args.savedTrialEndsAt}
        trialEndsOn={args.trialEndsOn}
        setTrialEndsOn={args.setTrialEndsOn}
        trialEndInvalid={args.trialEndInvalid}
        savingPlan={args.savingPlan}
        savePlan={args.savePlan}
      />
      <CreditBalanceMetrics credit={args.credit} trialActive={trialActive} />
      <div className="mt-6">
        <Heading level={3} className="text-gray-800 text-sm dark:text-gray-100">
          Set balances
        </Heading>
        <Text className="mt-1 text-xs text-gray-500 dark:text-gray-400">{ADMIN_SET_BALANCE_HELP_COPY}</Text>
        <div className="mt-3 grid gap-4 md:grid-cols-3">
          {EDITABLE_BALANCES.map(({ bucket, label, action }) => (
            <CreditBalanceField
              key={bucket}
              bucket={bucket}
              label={label}
              action={action}
              currentCents={creditRemainingCents(args.credit, bucket)}
              value={args.balanceInputs[bucket]}
              onChange={(value) => args.setBalanceInput(bucket, value)}
              disabledReason={bucket === "trial" && !trialActive ? ADMIN_NO_ACTIVE_TRIAL_COPY : null}
              saving={args.savingBalance === bucket}
              busy={args.savingBalance !== null}
              onSave={() => args.saveBalance(bucket)}
            />
          ))}
        </div>
        <Label htmlFor="admin-org-credit-note" className="mb-2 mt-4 block text-left">
          Note (optional)
        </Label>
        <Textarea
          id="admin-org-credit-note"
          data-testid="admin-org-credit-note"
          value={args.note}
          onChange={(event) => args.setNote(event.target.value)}
          placeholder="Reason for this change"
        />
      </div>
      <MarkupOverrideField
        markupBps={args.credit.markup_bps}
        markupPercent={args.markupPercent}
        setMarkupPercent={args.setMarkupPercent}
        savingMarkup={args.savingMarkup}
        saveMarkup={args.saveMarkup}
      />
      <CreditHistory grants={args.grants} loadFailed={args.grantsLoadFailed} onRetry={() => void args.reloadGrants()} />
    </div>
  );
}

function CreditBalanceMetrics({ credit, trialActive }: { credit: OrganizationLLMCredit; trialActive: boolean }) {
  return (
    <>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <CreditMetric
          label="Remaining hosted credit"
          value={formatUsdCents(credit.remaining_credit_cents)}
          testId="admin-org-credit-total-remaining"
        />
        <CreditMetric
          label="Trial credit"
          value={formatUsdCents(creditRemainingCents(credit, "trial"))}
          testId="admin-org-credit-trial-remaining"
          footer={trialCreditFooter(credit.welcome_credit_expires_at, trialActive)}
        />
        <CreditMetric
          label="Included usage"
          value={formatUsdCents(credit.included_remaining_cents ?? 0)}
          testId="admin-org-credit-included-remaining"
          footer={ADMIN_INCLUDED_USAGE_COPY}
        />
        <CreditMetric
          label="Top-up credit"
          value={formatUsdCents(creditRemainingCents(credit, "topup"))}
          testId="admin-org-credit-topup-remaining"
        />
        <CreditMetric
          label="SuperPlane grant"
          value={formatUsdCents(creditRemainingCents(credit, "grant"))}
          testId="admin-org-credit-grant-remaining"
        />
      </div>
      {credit.warning ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Remaining hosted credit is at or below the warning threshold.
        </Text>
      ) : null}
      {credit.remaining_credit_cents <= 0 ? (
        <Text className="mt-3 text-sm text-amber-700 dark:text-amber-400">
          Hosted credit is empty. Set a balance to restore SuperPlane-hosted runs.
        </Text>
      ) : null}
    </>
  );
}

function MarkupOverrideField(args: {
  markupBps: number;
  markupPercent: string;
  setMarkupPercent: (value: string) => void;
  savingMarkup: boolean;
  saveMarkup: () => void;
}) {
  return (
    <div className="mt-6 max-w-sm">
      <Label className="mb-2 block text-left">Markup override percent</Label>
      <InputGroup>
        <Input
          data-testid="admin-org-markup-override"
          value={args.markupPercent}
          onChange={(event) => args.setMarkupPercent(event.target.value)}
          placeholder={`Installation default (${bpsToPercentInput(args.markupBps)})`}
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
  );
}

function CreditHistory({
  grants,
  loadFailed,
  onRetry,
}: {
  grants: OrganizationsOrganizationCreditGrant[];
  loadFailed: boolean;
  onRetry: () => void;
}) {
  return (
    <div className="mt-6" data-testid="admin-org-credit-history">
      <Heading level={3} className="text-gray-800 text-sm dark:text-gray-100">
        Credit history
      </Heading>
      {loadFailed ? (
        <>
          <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">{ADMIN_CREDIT_HISTORY_ERROR_COPY}</Text>
          <Button type="button" className="mt-3" variant="outline" size="sm" onClick={onRetry}>
            Try again
          </Button>
        </>
      ) : null}
      {!loadFailed && grants.length === 0 ? (
        <Text className="mt-1 text-sm text-gray-500 dark:text-gray-400">No credit grants yet.</Text>
      ) : null}
      {!loadFailed && grants.length > 0 ? <CreditGrantTable grants={grants} /> : null}
    </div>
  );
}

function CreditBalanceField(args: {
  bucket: CreditBalanceBucket;
  label: string;
  action: string;
  currentCents: number;
  value: string;
  onChange: (value: string) => void;
  disabledReason: string | null;
  saving: boolean;
  busy: boolean;
  onSave: () => void;
}) {
  const inputId = `admin-org-credit-${args.bucket}-target`;
  const targetCents = parseDollarInputToCents(args.value);
  const changeCents = targetCents === null ? null : targetCents - args.currentCents;
  const disabled = args.disabledReason !== null;

  return (
    <div>
      <Label htmlFor={inputId} className="mb-2 block text-left">
        {args.label} (USD)
      </Label>
      <InputGroup>
        <Input
          id={inputId}
          data-testid={inputId}
          value={args.value}
          disabled={disabled}
          onChange={(event) => args.onChange(event.target.value)}
          placeholder="0.00"
        />
      </InputGroup>
      <Text
        className="mt-1 text-xs text-gray-500 dark:text-gray-400"
        data-testid={`admin-org-credit-${args.bucket}-change`}
      >
        {args.disabledReason ?? balanceChangeCopy(changeCents)}
      </Text>
      <Button
        type="button"
        className="mt-2"
        data-testid={`admin-org-credit-${args.bucket}-save`}
        onClick={args.onSave}
        disabled={disabled || args.busy || changeCents === null || changeCents === 0}
      >
        {args.saving ? "Saving..." : args.action}
      </Button>
    </div>
  );
}

function balanceChangeCopy(changeCents: number | null): string {
  if (changeCents === null) {
    return "Enter a balance of 0 or more.";
  }
  if (changeCents === 0) {
    return "No change.";
  }
  return `Ledger change: ${formatCreditGrantAmount(changeCents)}`;
}

function isTrialCreditActive(expiresAt: string | null | undefined, now: Date = new Date()): boolean {
  if (!expiresAt) {
    return false;
  }
  const parsed = new Date(expiresAt);
  return !Number.isNaN(parsed.getTime()) && parsed.getTime() > now.getTime();
}

function trialCreditFooter(expiresAt: string | null | undefined, active: boolean): string {
  if (!expiresAt) {
    return "No trial credit.";
  }
  const dateLabel = new Date(expiresAt).toLocaleDateString();
  return active ? `Expires on ${dateLabel}` : `Expired on ${dateLabel}`;
}

function CreditMetric({
  label,
  value,
  testId,
  footer,
}: {
  label: string;
  value: string;
  testId: string;
  footer?: string;
}) {
  return (
    <div>
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
      <p className="mt-1 text-base font-semibold text-gray-900 dark:text-gray-100" data-testid={testId}>
        {value}
      </p>
      {footer ? <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{footer}</p> : null}
    </div>
  );
}
