import type { FactoriesCodexUsageLimit } from "@/api-client";

export function CodexCreditsRemaining({
  credits,
  individualLimit,
}: Pick<FactoriesCodexUsageLimit, "credits" | "individualLimit">) {
  const monthlyLimit = !credits?.unlimited && credits?.balance == null ? individualLimit : undefined;
  return (
    <div className="space-y-1.5">
      <div className="flex min-w-0 items-center justify-between gap-2">
        <span className="text-muted-foreground">{monthlyLimit ? "Monthly credits" : "Available credits"}</span>
        <span className="shrink-0 font-medium">{formatCreditBalance({ credits, individualLimit: monthlyLimit })}</span>
      </div>
      {monthlyLimit?.resetsAt ? (
        <p className="text-[12px] text-muted-foreground">Resets: {new Date(monthlyLimit.resetsAt).toLocaleString()}</p>
      ) : null}
    </div>
  );
}

function formatCreditBalance({
  credits,
  individualLimit,
}: Pick<FactoriesCodexUsageLimit, "credits" | "individualLimit">) {
  if (credits?.unlimited) return "Unlimited";
  if (credits?.balance != null) {
    const balance = new Intl.NumberFormat(undefined, { maximumFractionDigits: 20 }).format(Number(credits.balance));
    return `${balance} credits`;
  }
  if (individualLimit) {
    const remaining = Math.max(0, Number(individualLimit.limit) - Number(individualLimit.used));
    return `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(remaining)} credits`;
  }
  return credits?.hasCredits ? "Available" : "0 credits";
}
