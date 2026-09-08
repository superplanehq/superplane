import { TriangleAlert } from "lucide-react";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { hostedCreditEmptyBannerCopy, type HostedCreditWarningLevel } from "./lib/hostedCreditEmpty";

interface HostedCreditEmptyBannerProps {
  /** `"empty"` renders severe (red) styling; `"low"` renders warning (amber) styling. */
  level: HostedCreditWarningLevel;
  billingEnabled: boolean;
  /** Whether the signed-in user can start hosted credit checkout (`org:update`). Defaults to `true`. */
  canManageBilling?: boolean;
  /** Links the action button to Organization Spending. Ignored when `onGoToBilling` is set. */
  spendingHref?: string;
  /** Renders the action as a button instead of a link to Spending, e.g. on the board (no billing page yet). */
  onGoToBilling?: () => void;
  className?: string;
}

const LEVEL_STYLES: Record<HostedCreditWarningLevel, { container: string; icon: string; button: string }> = {
  empty: {
    container: cn(
      "border-red-200 bg-red-50/90 text-red-950",
      "dark:border-red-700/50 dark:bg-red-950/30 dark:text-red-100",
    ),
    icon: "text-red-700 dark:text-red-300",
    button: cn(
      "border-red-300 bg-red-100/70 text-red-950 hover:bg-red-100",
      "dark:border-red-600 dark:bg-red-900/40 dark:text-red-50 dark:hover:bg-red-900/60",
    ),
  },
  low: {
    container: cn(
      "border-amber-200 bg-amber-50/90 text-amber-950",
      "dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-100",
    ),
    icon: "text-amber-700 dark:text-amber-300",
    button: cn(
      "border-amber-300 bg-amber-100/70 text-amber-950 hover:bg-amber-100",
      "dark:border-amber-600 dark:bg-amber-900/40 dark:text-amber-50 dark:hover:bg-amber-900/60",
    ),
  },
};

export function HostedCreditEmptyBanner({
  level,
  billingEnabled,
  spendingHref,
  onGoToBilling,
  className,
}: HostedCreditEmptyBannerProps) {
  const copy = hostedCreditEmptyBannerCopy(level, billingEnabled);
  const styles = LEVEL_STYLES[level];

  return (
    <div
      role="status"
      data-testid="hosted-credit-empty-banner"
      className={cn(
        "flex w-full flex-wrap items-center justify-between gap-3 rounded-lg border px-3.5 py-2.5 text-sm",
        styles.container,
        className,
      )}
    >
      <div className="flex min-w-0 items-start gap-2">
        <TriangleAlert className={cn("mt-0.5 h-4 w-4 shrink-0", styles.icon)} aria-hidden />
        <p>
          <span className="font-medium">{copy.title}. </span>
          {copy.description}
        </p>
      </div>
      {spendingHref ? (
        <Button asChild variant="outline" size="sm" className={styles.button}>
          <Link to={spendingHref}>View spending</Link>
        </Button>
      ) : (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className={styles.button}
          onClick={onGoToBilling ?? undefined}
        >
          Go to billing
        </Button>
      )}
    </div>
  );
}
