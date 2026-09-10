import { Sparkles, TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SUPERPLANE_PRICING_URL } from "@/lib/pricing";
import { cn } from "@/lib/utils";

import { formatUsdCents } from "./lib/workOrderUsage";
import {
  hostedCreditBannerCopy,
  parseWelcomeCreditExpiresAt,
  welcomeCreditHeaderLabel,
  type HostedCreditBannerKind,
} from "./lib/hostedCreditEmpty";

interface HostedCreditEmptyBannerProps {
  billingEnabled: boolean;
  /** Whether the signed-in user can start hosted credit checkout (`org:update`). Defaults to `true`. */
  canManageBilling?: boolean;
  spendingHref: string;
  className?: string;
  kind?: HostedCreditBannerKind;
  remainingCreditCents?: number;
  welcomeCreditExpiresAt?: string;
  subscriptionCheckoutEnabled?: boolean;
}

const BANNER_TONE_CLASS = {
  info: "border-border bg-muted/40 text-foreground",
  warning: "border-border bg-muted/40 text-foreground shadow-[inset_3px_0_0_0_var(--status-waiting-dot)]",
} as const;

const BANNER_ICON_CLASS = {
  info: "text-muted-foreground",
  warning: "text-[color:var(--status-waiting-fg)]",
} as const;

export function HostedCreditEmptyBanner({
  billingEnabled,
  spendingHref,
  className,
  kind = "empty",
  remainingCreditCents = 0,
  welcomeCreditExpiresAt,
  subscriptionCheckoutEnabled,
}: HostedCreditEmptyBannerProps) {
  const copy = hostedCreditBannerCopy({
    kind,
    billingEnabled,
    remainingCreditCents,
    welcomeCreditExpiresAt: parseWelcomeCreditExpiresAt(welcomeCreditExpiresAt) ?? undefined,
    subscriptionCheckoutEnabled,
  });
  const isWarning = copy.tone === "warning";
  const Icon = isWarning ? TriangleAlert : Sparkles;
  const hasHint = Boolean(copy.consequenceHint);

  return (
    <div
      role="status"
      data-testid="hosted-credit-empty-banner"
      data-tone={copy.tone}
      className={cn(
        "flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border px-3.5 py-2 text-sm",
        BANNER_TONE_CLASS[copy.tone],
        className,
      )}
    >
      <div className={cn("flex min-w-0 gap-2.5", hasHint ? "items-start" : "items-center")}>
        <Icon className={cn("h-4 w-4 shrink-0", hasHint && "mt-0.5", BANNER_ICON_CLASS[copy.tone])} aria-hidden />
        {copy.remainingLabel ? (
          <TrialCreditSummary
            title={copy.title}
            remainingLabel={copy.remainingLabel}
            expiryLabel={copy.expiryLabel}
            consequenceHint={copy.consequenceHint}
          />
        ) : (
          <p>
            <span className="font-medium">{copy.title}. </span>
            {copy.description}
          </p>
        )}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {copy.showPricingLink ? (
          <Button asChild variant="ghost" size="sm">
            <a href={SUPERPLANE_PRICING_URL} target="_blank" rel="noreferrer">
              See pricing
            </a>
          </Button>
        ) : null}
        {copy.showAction !== false ? (
          <Button asChild variant="outline" size="sm">
            <Link to={spendingHref}>{copy.actionLabel}</Link>
          </Button>
        ) : null}
      </div>
    </div>
  );
}

/** Compact trial chip to the right of the workspace page title. */
export function HostedCreditHeaderKicker({
  spendingHref,
  welcomeCreditExpiresAt,
  remainingCreditCents = 0,
}: {
  spendingHref: string;
  welcomeCreditExpiresAt?: string;
  remainingCreditCents?: number;
}) {
  const expiresAt = parseWelcomeCreditExpiresAt(welcomeCreditExpiresAt);
  const duration = expiresAt ? welcomeCreditHeaderLabel(expiresAt) : "14 days";
  const remaining = formatUsdCents(remainingCreditCents);

  return (
    <Link
      to={spendingHref}
      aria-label="Subscribe"
      data-testid="hosted-credit-header-kicker"
      className="inline-flex h-8 shrink-0 items-center gap-2 rounded-full bg-violet-100 py-1 pl-2.5 pr-1.5 text-[12px] hover:bg-violet-200/80 dark:bg-violet-950 dark:hover:bg-violet-900"
    >
      <span className="whitespace-nowrap font-medium text-violet-800 dark:text-violet-200">Trial</span>
      <ChipDot />
      <span className="whitespace-nowrap font-medium text-violet-800 dark:text-violet-200">{duration}</span>
      <ChipDot />
      <span className="whitespace-nowrap font-semibold tabular-nums text-violet-950 dark:text-violet-50">
        {remaining}
      </span>
      <span className="inline-flex h-5 items-center rounded-full bg-violet-600 px-2.5 text-[11px] leading-none font-medium text-white">
        Subscribe
      </span>
    </Link>
  );
}

function ChipDot() {
  return (
    <span className="select-none text-violet-400 dark:text-violet-600" aria-hidden>
      ·
    </span>
  );
}

function TrialCreditSummary({
  title,
  remainingLabel,
  expiryLabel,
  consequenceHint,
}: {
  title: string;
  remainingLabel: string;
  expiryLabel?: string;
  consequenceHint?: string;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <div className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1">
        <Badge variant="outline" className="font-medium text-muted-foreground">
          {title}
        </Badge>
        <p className="flex min-w-0 flex-wrap items-baseline gap-x-2 text-sm">
          <span className="font-medium tabular-nums">{remainingLabel}</span>
          {expiryLabel ? (
            <>
              <span className="select-none text-muted-foreground" aria-hidden>
                ·
              </span>
              <span className="text-muted-foreground">{expiryLabel}</span>
            </>
          ) : null}
        </p>
      </div>
      {consequenceHint ? <p className="text-xs text-muted-foreground">{consequenceHint}</p> : null}
    </div>
  );
}
