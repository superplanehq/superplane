import type { FactoriesCodexUsageWindow } from "@/api-client";
import { useTimeHasPassed } from "@/hooks/useTimeHasPassed";

function useQuotaWindowStatus(window: FactoriesCodexUsageWindow, fallbackLabel: string) {
  const expired = useTimeHasPassed(window.resetsAt);
  const minutes = Number(window.windowDurationMins);
  const windowLabel =
    minutes > 0
      ? minutes % 1440 === 0
        ? `${minutes / 1440}-day window`
        : minutes % 60 === 0
          ? `${minutes / 60}-hour window`
          : `${minutes}-minute window`
      : fallbackLabel;
  const remaining = window.usedPercent === undefined ? undefined : Math.max(0, Math.min(100, 100 - window.usedPercent));
  const status = expired ? "Update needed" : remaining === undefined ? "Usage unavailable" : `${remaining}% remaining`;

  return { windowLabel, expired, status };
}

export function CodexQuotaRemaining({ window, label }: { window: FactoriesCodexUsageWindow; label: string }) {
  const { windowLabel, status } = useQuotaWindowStatus(window, label);
  return (
    <div className="flex min-w-0 items-center justify-between gap-2">
      <span className="text-muted-foreground">{windowLabel}</span>
      <span className="shrink-0 font-medium">{status}</span>
    </div>
  );
}

export function CodexQuotaWindow({ window, label }: { window: FactoriesCodexUsageWindow; label: string }) {
  const { windowLabel, expired, status } = useQuotaWindowStatus(window, label);

  return (
    <div className="space-y-1.5">
      <div className="flex justify-between gap-3">
        <span>{windowLabel}</span>
        <span className="font-medium">{status}</span>
      </div>
      {window.resetsAt ? (
        <p className="text-[12px] text-muted-foreground">
          {expired ? "Reset time passed" : "Resets"}: {new Date(window.resetsAt).toLocaleString()}
        </p>
      ) : null}
    </div>
  );
}
