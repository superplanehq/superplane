import { durationLabelMs, formatClockDuration, formatClockDurationLabel } from "@/lib/duration";

import type { SplitRunPhaseStatus } from "./splitRunMocks";

export const RUNNING_SPINNER_FRAMES = ["|", "/", "-", "\\"] as const;

export function runningSpinnerFrame(nowMs: number): string {
  const index = Math.floor(Math.max(0, nowMs) / 250) % RUNNING_SPINNER_FRAMES.length;
  return RUNNING_SPINNER_FRAMES[index];
}

export function tickingRunningClock(duration: string | undefined, sampledAtMs: number, nowMs: number): string {
  const base = duration ? durationLabelMs(duration) : 0;
  const elapsedMs = Math.max(0, nowMs - sampledAtMs);
  return formatClockDuration(base + Math.floor(elapsedMs / 1000) * 1000);
}

export function logStatusTimeLabel(status: SplitRunPhaseStatus, duration?: string): string {
  const word = statusWord(status);
  const clock = duration ? formatClockDurationLabel(duration) : "";
  const hasClock = Boolean(clock && clock !== "—");
  if (!word) {
    return hasClock ? clock : "";
  }
  if (!hasClock) {
    return word;
  }
  return `${word} ${clock}`;
}

function statusWord(status: SplitRunPhaseStatus): string {
  if (status === "passed") {
    return "Passed";
  }
  if (status === "running") {
    return "Running";
  }
  if (status === "failed") {
    return "Failed";
  }
  if (status === "waiting") {
    return "Waiting";
  }
  return "";
}
