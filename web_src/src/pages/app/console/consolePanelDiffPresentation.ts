import { Diff, Minus, Plus } from "lucide-react";

import type { DraftDiffStatus } from "../draftNodeDiff";

export const PANEL_DIFF_BADGE = {
  added: {
    label: "ADDED",
    className: "bg-green-500 text-white",
    borderClassName: "border-green-500",
    Icon: Plus,
  },
  updated: {
    label: "EDITED",
    className: "bg-sky-500 text-white",
    borderClassName: "border-sky-500",
    Icon: Diff,
  },
  removed: {
    label: "REMOVED",
    className: "bg-red-500 text-white",
    borderClassName: "border-red-500",
    Icon: Minus,
  },
} as const;

export function consolePanelDiffBorderClassName(status?: DraftDiffStatus): string | undefined {
  return status ? PANEL_DIFF_BADGE[status].borderClassName : undefined;
}
