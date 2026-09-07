import { TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { hostedCreditEmptyBannerCopy } from "./lib/hostedCreditEmpty";

interface HostedCreditEmptyBannerProps {
  billingEnabled: boolean;
  /** Whether the signed-in user can start hosted credit checkout (`org:update`). Defaults to `true`. */
  canManageBilling?: boolean;
  spendingHref: string;
  className?: string;
}

export function HostedCreditEmptyBanner({
  billingEnabled,
  canManageBilling = true,
  spendingHref,
  className,
}: HostedCreditEmptyBannerProps) {
  const copy = hostedCreditEmptyBannerCopy(billingEnabled, canManageBilling);

  return (
    <div
      role="status"
      data-testid="hosted-credit-empty-banner"
      className={cn(
        "flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border border-amber-200 bg-amber-50/90 px-3.5 py-2.5 text-sm text-amber-950",
        "dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100",
        className,
      )}
    >
      <div className="flex min-w-0 items-start gap-2">
        <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-700 dark:text-amber-300" aria-hidden />
        <p>
          <span className="font-medium">{copy.title}. </span>
          {copy.description}
        </p>
      </div>
      <Button
        asChild
        variant="outline"
        size="sm"
        className="border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100 dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60"
      >
        <Link to={spendingHref}>{copy.actionLabel}</Link>
      </Button>
    </div>
  );
}
