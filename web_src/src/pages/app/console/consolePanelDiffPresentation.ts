import { Diff, Minus, Plus } from "lucide-react";

import type { DraftDiffStatus } from "../draftNodeDiff";

export const PANEL_DIFF_BADGE = {
  added: {
    label: "ADDED",
    colorClass: "bg-green-500",
    borderClassName: "border-green-500",
    Icon: Plus,
  },
  updated: {
    label: "EDITED",
    colorClass: "bg-sky-500",
    borderClassName: "border-sky-500",
    Icon: Diff,
  },
  removed: {
    label: "REMOVED",
    colorClass: "bg-red-500",
    borderClassName: "border-red-500",
    Icon: Minus,
  },
} as const;

export function consolePanelDiffBorderClassName(status?: DraftDiffStatus): string | undefined {
  return status ? PANEL_DIFF_BADGE[status].borderClassName : undefined;
}
