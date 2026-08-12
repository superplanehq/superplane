import type { FactoryNodeStatus } from "./types";

export function normalizeFactoryNodeStatus(status: string | undefined): FactoryNodeStatus {
  switch (status) {
    case "passed":
    case "created":
    case "success":
    case "finished":
      return "passed";
    case "failed":
    case "error":
      return "failed";
    case "running":
    case "cancelling":
      return "running";
    default:
      return "pending";
  }
}

export function factoryNodeStatusLabel(status: FactoryNodeStatus): string {
  switch (status) {
    case "passed":
      return "Passed";
    case "failed":
      return "Failed";
    case "running":
      return "Running";
    case "pending":
      return "Pending";
  }
}

export function factoryNodeStatusStripClass(status: FactoryNodeStatus): string {
  switch (status) {
    case "passed":
      return "border-[#dcfce7] bg-[#f0fdf4] text-[#166534] dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300";
    case "failed":
      return "border-[#fecaca] bg-[#fef2f2] text-[#991b1b] dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300";
    case "running":
      return "border-[#bfdbfe] bg-[#eff6ff] text-[#1d4ed8] dark:border-blue-500/20 dark:bg-blue-500/10 dark:text-blue-300";
    case "pending":
      return "border-[#e5e5e5] bg-[#fafafa] text-[#737373] dark:border-border dark:bg-muted dark:text-muted-foreground";
  }
}

/** Compact duration: `7s`, `1m 16s`, `9m so far`. */
export function formatFactoryNodeDuration(ms: number, options?: { soFar?: boolean }): string {
  const totalSec = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(totalSec / 3600);
  const minutes = Math.floor((totalSec % 3600) / 60);
  const seconds = totalSec % 60;

  let text: string;
  if (hours > 0) {
    text = minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`;
  } else if (minutes > 0) {
    text = seconds > 0 ? `${minutes}m ${seconds}s` : `${minutes}m`;
  } else {
    text = `${seconds}s`;
  }

  return options?.soFar ? `${text} so far` : text;
}
