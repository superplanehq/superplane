import { durationLabelMs, formatClockDuration, formatClockDurationLabel } from "@/lib/duration";

export function tickingRunningClock(duration: string | undefined, sampledAtMs: number, nowMs: number): string {
  const base = duration ? durationLabelMs(duration) : 0;
  const elapsedMs = Math.max(0, nowMs - sampledAtMs);
  return formatClockDuration(base + Math.floor(elapsedMs / 1000) * 1000);
}

export function logStatusTimeLabel(duration?: string): string {
  const clock = duration ? formatClockDurationLabel(duration) : "";
  if (!clock || clock === "—") {
    return "";
  }
  return clock;
}
