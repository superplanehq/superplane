export type DraftDiffStatus = "added" | "updated" | "removed";

const DRAFT_DIFF_OUTLINE: Record<DraftDiffStatus, string> = {
  added: "outline-2 outline-green-500",
  updated: "outline-2 outline-sky-400",
  removed: "outline-2 outline-red-400",
};

/** Tailwind green-500 / red-400 — aligned with node diff badge and outline colors. */
export const DRAFT_DIFF_EDGE_STROKE = {
  added: "#22c55e",
  removed: "#f87171",
} as const;

export function getDraftDiffOutlineClassName(status?: DraftDiffStatus): string {
  if (!status) {
    return "outline-1 outline-slate-950/20 dark:outline-gray-600/70";
  }

  return DRAFT_DIFF_OUTLINE[status];
}

export function getDraftDiffEdgeStyle(status: unknown): { stroke: string; strokeDasharray?: string } | undefined {
  if (status === "added") {
    return { stroke: DRAFT_DIFF_EDGE_STROKE.added };
  }

  if (status === "removed") {
    return { stroke: DRAFT_DIFF_EDGE_STROKE.removed, strokeDasharray: "8 4" };
  }

  return undefined;
}
