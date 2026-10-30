import { useEffect, useState } from "react";
import type { FactoriesCodexUsageWindow } from "@/api-client";

function useQuotaWindowStatus(window: FactoriesCodexUsageWindow, fallbackLabel: string) {
  const [now, setNow] = useState(Date.now);
  const resetsAt = window.resetsAt ? new Date(window.resetsAt).getTime() : NaN;
  useEffect(() => {
    if (!Number.isFinite(resetsAt)) return;
    let timer: ReturnType<typeof setTimeout>;
    const scheduleReset = () => {
      const current = Date.now();
      setNow(current);
      if (resetsAt > current) {
        timer = setTimeout(scheduleReset, Math.min(resetsAt - current, 2147483647));
      }
    };
    scheduleReset();
    return () => clearTimeout(timer);
  }, [resetsAt]);
  const minutes = Number(window.windowDurationMins);
  const windowLabel =
    minutes > 0
      ? minutes % 1440 === 0
        ? `${minutes / 1440}-day window`
        : minutes % 60 === 0
          ? `${minutes / 60}-hour window`
          : `${minutes}-minute window`
      : fallbackLabel;
  const expired = resetsAt <= Math.max(now, Date.now());
  const remaining = window.usedPercent === undefined ? undefined : Math.max(0, Math.min(100, 100 - window.usedPercent));
  const status = expired ? "Update needed" : remaining === undefined ? "Usage unavailable" : `${remaining}% remaining`;

  return { windowLabel, expired, status };
}

export function CodexQuotaRemaining({
  window,
  label,
  showWindowLabel,
}: {
  window: FactoriesCodexUsageWindow;
  label: string;
  showWindowLabel: boolean;
}) {
  const { windowLabel, status } = useQuotaWindowStatus(window, label);
  return (
    <div className={`flex min-w-0 items-center gap-2 ${showWindowLabel ? "justify-between" : "justify-center"}`}>
      {showWindowLabel ? <span className="text-muted-foreground">{windowLabel}</span> : null}
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
