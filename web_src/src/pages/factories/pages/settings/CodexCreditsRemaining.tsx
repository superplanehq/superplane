import type { FactoriesCodexUsageCredits } from "@/api-client";

export function CodexCreditsRemaining({ credits }: { credits: FactoriesCodexUsageCredits }) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-2">
      <span className="text-muted-foreground">Available credits</span>
      <span className="shrink-0 font-medium">{formatCreditBalance(credits)}</span>
    </div>
  );
}

function formatCreditBalance(credits: FactoriesCodexUsageCredits) {
  if (credits.unlimited) return "Unlimited";
  if (credits.balance !== undefined) {
    const balance = new Intl.NumberFormat(undefined, { maximumFractionDigits: 20 }).format(Number(credits.balance));
    return `${balance} credits`;
  }
  return credits.hasCredits ? "Available" : "0 credits";
}
