import { Link } from "react-router";

import { cn } from "@/lib/utils";

import { formatUsdCents } from "./lib/workOrderUsage";
import {
  hostedCreditHeaderKickerActionLabel,
  hostedCreditHeaderKickerLabel,
  parseWelcomeCreditExpiresAt,
  welcomeCreditHeaderLabel,
  type HostedCreditHeaderKickerKind,
} from "./lib/hostedCreditEmpty";

interface HostedCreditHeaderKickerProps {
  spendingHref: string;
  welcomeCreditExpiresAt?: string;
  remainingCreditCents?: number;
  kind?: HostedCreditHeaderKickerKind;
  /** Whether the signed-in organization can buy credit. Hides the action and link for low and empty credit. */
  canAddCredit?: boolean;
  /** Phone board chip. Hides the action pill and lets the row truncate. */
  compact?: boolean;
}

/** Credit chip next to the workspace page title. */
export function HostedCreditHeaderKicker({
  spendingHref,
  welcomeCreditExpiresAt,
  remainingCreditCents = 0,
  kind = "trial",
  canAddCredit = true,
  compact = false,
}: HostedCreditHeaderKickerProps) {
  const palette = KICKER_PALETTE[kind];
  const label = hostedCreditHeaderKickerLabel(kind);
  const actionLabel = hostedCreditHeaderKickerActionLabel(kind);
  const details = kickerDetails(kind, welcomeCreditExpiresAt, remainingCreditCents);
  const showAction = kind !== "low" && kind !== "empty" ? true : canAddCredit;
  const showActionPill = showAction && !compact;
  const className = cn(
    "inline-flex h-8 items-center rounded-full py-1 text-[12px]",
    compact ? "min-w-0 max-w-full overflow-hidden px-2.5" : "shrink-0 pl-2.5 pr-1.5",
    palette.shell,
    showAction && palette.hover,
  );
  const content = (
    <span className={cn(compact ? "block min-w-0 truncate" : "inline-flex items-center gap-2")}>
      <span className={cn("whitespace-nowrap font-medium", palette.text)}>{label}</span>
      {details.map((detail) => (
        <span
          key={detail.key}
          className={cn("whitespace-nowrap", compact ? "inline" : "inline-flex items-center gap-2")}
        >
          <ChipDot className={cn(palette.dot, compact && "mx-2")} />
          <span
            className={cn(
              "whitespace-nowrap",
              detail.emphasis ? cn("font-semibold tabular-nums", palette.amount) : cn("font-medium", palette.text),
            )}
          >
            {detail.text}
          </span>
        </span>
      ))}
      {showActionPill ? (
        <span
          className={cn(
            "inline-flex h-5 items-center rounded-full px-2.5 text-[11px] leading-none font-medium text-white",
            palette.action,
          )}
        >
          {actionLabel}
        </span>
      ) : null}
    </span>
  );

  if (!showAction) {
    return (
      <span data-testid="hosted-credit-header-kicker" data-kind={kind} className={className}>
        {content}
      </span>
    );
  }

  return (
    <Link
      to={spendingHref}
      aria-label={compact ? compactAccessibleName(label, details, actionLabel) : actionLabel}
      data-testid="hosted-credit-header-kicker"
      data-kind={kind}
      className={className}
    >
      {content}
    </Link>
  );
}

function compactAccessibleName(label: string, details: Array<{ text: string }>, actionLabel: string): string {
  return [label, ...details.map((detail) => detail.text), actionLabel].join(", ");
}

type KickPalette = {
  shell: string;
  hover: string;
  text: string;
  amount: string;
  dot: string;
  action: string;
};

const TRIAL_KICKER_PALETTE: KickPalette = {
  shell: "bg-violet-100 dark:bg-violet-950",
  hover: "hover:bg-violet-200/80 dark:hover:bg-violet-900",
  text: "text-violet-800 dark:text-violet-200",
  amount: "text-violet-950 dark:text-violet-50",
  dot: "text-violet-400 dark:text-violet-600",
  action: "bg-violet-600",
};

const LAPSED_KICKER_PALETTE: KickPalette = {
  shell: "bg-amber-100 dark:bg-amber-950",
  hover: "hover:bg-amber-200/80 dark:hover:bg-amber-900",
  text: "text-amber-900 dark:text-amber-200",
  amount: "text-amber-950 dark:text-amber-50",
  dot: "text-amber-400 dark:text-amber-600",
  action: "bg-amber-600",
};

const KICKER_PALETTE: Record<HostedCreditHeaderKickerKind, KickPalette> = {
  trial: TRIAL_KICKER_PALETTE,
  "trial-expired": LAPSED_KICKER_PALETTE,
  lapsed: LAPSED_KICKER_PALETTE,
  low: LAPSED_KICKER_PALETTE,
  empty: LAPSED_KICKER_PALETTE,
};

function kickerDetails(
  kind: HostedCreditHeaderKickerKind,
  welcomeCreditExpiresAt: string | undefined,
  remainingCreditCents: number,
): Array<{ key: string; text: string; emphasis?: boolean }> {
  if (kind === "trial") {
    return trialKickerDetails(welcomeCreditExpiresAt, remainingCreditCents);
  }
  if (kind === "low") {
    return [{ key: "remaining", text: formatUsdCents(remainingCreditCents), emphasis: true }];
  }
  return [];
}

function trialKickerDetails(
  welcomeCreditExpiresAt: string | undefined,
  remainingCreditCents: number,
): Array<{ key: string; text: string; emphasis?: boolean }> {
  const expiresAt = parseWelcomeCreditExpiresAt(welcomeCreditExpiresAt);
  return [
    { key: "duration", text: expiresAt ? welcomeCreditHeaderLabel(expiresAt) : "14 days" },
    { key: "remaining", text: formatUsdCents(remainingCreditCents), emphasis: true },
  ];
}

function ChipDot({ className }: { className: string }) {
  return (
    <span className={cn("select-none", className)} aria-hidden>
      ·
    </span>
  );
}
