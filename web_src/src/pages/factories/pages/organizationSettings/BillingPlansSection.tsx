import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";

import {
  BILLING_BUSINESS_DESCRIPTION,
  BILLING_BUSINESS_NAME,
  BILLING_BUSINESS_PRICE,
  BILLING_BUSINESS_PRICE_PERIOD,
  BILLING_UPGRADE_LABEL,
  billingBusinessPlanAction,
  billingPlansUsageView,
  type BillingPlansUsageInput,
  type BillingPlansUsageView,
} from "../../lib/billingPlans";
import { FactorySettingsCard } from "../settings/FactorySettingsCard";

export function BillingPlansSection({
  usage,
  canManageBilling,
  creditPurchaseAllowed,
  pending,
  onSubscribe,
}: {
  usage: BillingPlansUsageInput;
  canManageBilling: boolean;
  creditPurchaseAllowed: boolean;
  pending: boolean;
  onSubscribe: () => void | Promise<void>;
}) {
  const usageView = billingPlansUsageView(usage);
  const businessAction = billingBusinessPlanAction({
    canManageBilling,
    creditPurchaseAllowed,
  });

  return (
    <FactorySettingsCard title="Plans" data-testid="billing-plans">
      <div className="grid auto-rows-fr gap-3 sm:grid-cols-2">
        <UsageCard usage={usageView} />
        <div
          className="flex h-full min-w-0 flex-col rounded-lg border border-border p-4"
          data-testid="billing-plan-business"
        >
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-[13px] font-medium tracking-[-0.01em]">{BILLING_BUSINESS_NAME}</h3>
            {businessAction === "current" ? (
              <Badge variant="outline" className="text-muted-foreground" data-testid="billing-current-plan">
                Current plan
              </Badge>
            ) : null}
          </div>
          <p className="mt-2 flex flex-wrap items-baseline gap-x-1.5">
            <span className="text-xl font-semibold tracking-[-0.02em]">{BILLING_BUSINESS_PRICE}</span>
            <span className="text-xs text-muted-foreground">{BILLING_BUSINESS_PRICE_PERIOD}</span>
          </p>
          <p className="mt-1 text-sm text-muted-foreground">{BILLING_BUSINESS_DESCRIPTION}</p>
          {businessAction === "subscribe" ? (
            <div className="mt-auto pt-4">
              <Button
                type="button"
                disabled={pending}
                data-testid="billing-subscribe"
                onClick={() => void onSubscribe()}
              >
                {pending ? "Opening checkout..." : BILLING_UPGRADE_LABEL}
              </Button>
            </div>
          ) : null}
        </div>
      </div>
      <p className="mt-3 text-xs text-muted-foreground">
        For air-gapped installation, audit logs, or invoice billing,{" "}
        <a
          href={SUPERPLANE_PRICING_URL}
          target="_blank"
          rel="noreferrer"
          className="underline underline-offset-2"
          data-testid="billing-talk-to-us"
        >
          Talk to us
        </a>
      </p>
    </FactorySettingsCard>
  );
}

function UsageCard({ usage }: { usage: BillingPlansUsageView }) {
  return (
    <div className="flex h-full min-w-0 flex-col rounded-lg border border-border p-4" data-testid="billing-plan-usage">
      <h3 className="text-[13px] font-medium tracking-[-0.01em]">{usage.heading}</h3>
      <p className="mt-2 text-xl font-semibold tracking-[-0.02em]">{usage.remainingLabel}</p>
      <div
        className="mt-3 h-1 overflow-hidden rounded-full bg-muted"
        role="progressbar"
        aria-label={usage.heading}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={usage.usedPercent}
        data-testid="billing-plan-usage-bar"
      >
        <div className="h-full rounded-full bg-foreground/80" style={{ width: `${usage.usedPercent}%` }} />
      </div>
      {usage.footer ? <p className="mt-2 text-xs text-muted-foreground">{usage.footer}</p> : null}
    </div>
  );
}
