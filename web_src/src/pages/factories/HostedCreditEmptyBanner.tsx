import { Sparkles, TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import {
  hostedCreditBannerCopy,
  parseWelcomeCreditExpiresAt,
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
}

export function HostedCreditEmptyBanner({
  billingEnabled,
  spendingHref,
  className,
  kind = "empty",
  remainingCreditCents = 0,
  welcomeCreditExpiresAt,
}: HostedCreditEmptyBannerProps) {
  const copy = hostedCreditBannerCopy({
    kind,
    billingEnabled,
    remainingCreditCents,
    welcomeCreditExpiresAt: parseWelcomeCreditExpiresAt(welcomeCreditExpiresAt) ?? undefined,
  });
  const isWarning = copy.tone === "warning";
  const Icon = isWarning ? TriangleAlert : Sparkles;

  return (
    <div
      role="status"
      data-testid="hosted-credit-empty-banner"
      data-tone={copy.tone}
      className={cn(
        "flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border px-3.5 py-2 text-sm",
        isWarning
          ? "border-amber-200 bg-amber-50/90 text-amber-950 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100"
          : "border-border bg-muted/40 text-foreground",
        className,
      )}
    >
      <div className="flex min-w-0 items-center gap-2.5">
        <Icon
          className={cn("h-4 w-4 shrink-0", isWarning ? "text-amber-700 dark:text-amber-300" : "text-muted-foreground")}
          aria-hidden
        />
        {copy.remainingLabel ? (
          <TrialCreditSummary
            title={copy.title}
            remainingLabel={copy.remainingLabel}
            expiryLabel={copy.expiryLabel}
            isWarning={isWarning}
          />
        ) : (
          <p>
            <span className="font-medium">{copy.title}. </span>
            {copy.description}
          </p>
        )}
      </div>
      <Button
        asChild
        variant="outline"
        size="sm"
        className={
          isWarning
            ? "border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60"
            : undefined
        }
      >
        <Link to={spendingHref}>{copy.actionLabel}</Link>
      </Button>
    </div>
  );
}

function TrialCreditSummary({
  title,
  remainingLabel,
  expiryLabel,
  isWarning,
}: {
  title: string;
  remainingLabel: string;
  expiryLabel?: string;
  isWarning: boolean;
}) {
  const mutedClass = isWarning ? "text-amber-900/80 dark:text-amber-100/80" : "text-muted-foreground";
  return (
    <div className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1">
      <Badge
        variant="outline"
        className={cn(
          "font-medium",
          isWarning
            ? "border-amber-300/80 bg-amber-100/50 text-amber-950 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50"
            : "text-muted-foreground",
        )}
      >
        {title}
      </Badge>
      <p className="flex min-w-0 flex-wrap items-baseline gap-x-2 text-sm">
        <span className="font-medium tabular-nums">{remainingLabel}</span>
        {expiryLabel ? (
          <>
            <span className={cn("select-none", mutedClass)} aria-hidden>
              ·
            </span>
            <span className={mutedClass}>{expiryLabel}</span>
          </>
        ) : null}
      </p>
    </div>
  );
}
