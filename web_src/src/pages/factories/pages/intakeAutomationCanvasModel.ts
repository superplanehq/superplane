import type { CanvasesCanvas } from "@/api-client";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import type { SplitRunCanvasModel } from "./work-order-split-run/splitRunCanvases";

/**
 * Builds the Automation tab graph from the intake app's canvas. Returns
 * undefined when the app has no graph yet, so the tab can show an empty state
 * instead of an invented one.
 */
export function intakeAutomationCanvasFromApp(title: string, canvas?: CanvasesCanvas): SplitRunCanvasModel | undefined {
  const nodes = canvas?.spec?.nodes ?? [];
  if (nodes.length === 0) {
    return undefined;
  }

  // The tab shows how the intake is wired, not a run, so only a node that
  // reports a real problem carries a status.
  const statuses: Record<string, FactoryNodeStatus> = {};
  for (const node of nodes) {
    if (node.id && node.errorMessage?.trim()) {
      statuses[node.id] = "failed";
    }
  }

  return {
    key: "intake",
    title,
    nodes,
    edges: canvas?.spec?.edges ?? [],
    statuses,
    metrics: {},
  };
}
