import { useMemo } from "react";
import type { MentionCandidate } from "./MentionDropdown";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

function timeAgo(dateStr?: string): string {
  if (!dateStr) return "";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

export function useMentionCandidates(
  nodes: SuperplaneComponentsNode[] | undefined,
  runs: CanvasesCanvasRun[] | undefined,
  filter: string,
): MentionCandidate[] {
  return useMemo((): MentionCandidate[] => {
    const filterLower = filter.toLowerCase();
    const result: MentionCandidate[] = [];

    if (nodes) {
      for (const node of nodes) {
        const name = node.name || node.id || "";
        if (filterLower && !name.toLowerCase().includes(filterLower)) continue;
        result.push({
          type: "node",
          id: node.id || "",
          label: name,
          meta: node.component,
          isTrigger: node.type === "TYPE_TRIGGER",
        });
      }
    }

    if (runs) {
      const recentRuns = runs.slice(0, 10);
      for (const run of recentRuns) {
        const label = `Run #${run.id?.slice(0, 6) || "?"}`;
        if (filterLower && !label.toLowerCase().includes(filterLower)) continue;
        result.push({
          type: "run",
          id: run.id || "",
          label,
          meta: run.result || run.state,
          timeAgo: timeAgo(run.createdAt),
        });
      }
    }

    return result;
  }, [nodes, runs, filter]);
}
