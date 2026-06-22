import { useMemo } from "react";
import type { MentionCandidate } from "./MentionDropdown";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

const emptyMentionCandidates: MentionCandidate[] = [];

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

function buildNodeCandidates(nodes: SuperplaneComponentsNode[], filterLower: string): MentionCandidate[] {
  return nodes
    .map((node): MentionCandidate | null => {
      const name = node.name || node.id || "";
      if (filterLower && !name.toLowerCase().includes(filterLower)) return null;
      return {
        type: "node",
        id: node.id || "",
        label: name,
        meta: node.component,
        isTrigger: node.type === "TYPE_TRIGGER",
      };
    })
    .filter((c): c is MentionCandidate => c !== null);
}

function buildRunCandidates(runs: CanvasesCanvasRun[], filterLower: string): MentionCandidate[] {
  return runs
    .slice(0, 10)
    .map((run): MentionCandidate | null => {
      const label = `Run #${run.id?.slice(0, 6) || "?"}`;
      if (filterLower && !label.toLowerCase().includes(filterLower)) return null;
      return { type: "run", id: run.id || "", label, meta: run.result || run.state, timeAgo: timeAgo(run.createdAt) };
    })
    .filter((c): c is MentionCandidate => c !== null);
}

export function useMentionCandidates(
  nodes: SuperplaneComponentsNode[] | undefined,
  runs: CanvasesCanvasRun[] | undefined,
  filter: string,
  enabled = true,
): MentionCandidate[] {
  return useMemo(() => {
    if (!enabled) {
      return emptyMentionCandidates;
    }

    const filterLower = filter.toLowerCase();
    return [
      ...(nodes ? buildNodeCandidates(nodes, filterLower) : []),
      ...(runs ? buildRunCandidates(runs, filterLower) : []),
    ];
  }, [nodes, runs, filter, enabled]);
}
