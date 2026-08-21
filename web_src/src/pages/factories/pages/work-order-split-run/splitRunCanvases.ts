import type {
  ComponentsEdge,
  FactoriesWorkOrderArtifact,
  SuperplaneComponentsNode as ComponentsNode,
} from "@/api-client";
import { parseCanvasYamlMetadata, parseCanvasYamlToSpec } from "@/pages/app/lib/canvas-yaml-staging";
import type { FactoryNodeStatus } from "@/ui/factoryNodeChrome/types";

import { OPEN_WORK_ORDER_ARTIFACTS } from "../../__fixtures__/factoryPageFixtureVariants";
import { HOUR_AGO, REVIEWER_USER } from "../../__fixtures__/factoryPageResponses";
import issueIntakeYaml from "@/pages/home/factories/line-apps/issue-intake.canvas.yaml?raw";
import planningYaml from "@/pages/home/factories/line-apps/planning.canvas.yaml?raw";
import implementationYaml from "@/pages/home/factories/line-apps/implementation.canvas.yaml?raw";
import prClosureYaml from "@/pages/home/factories/line-apps/pr-closure.canvas.yaml?raw";
import { DESCRIPTION_ARTIFACT, PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import riskAssessmentYaml from "./risk-assessment.canvas.yaml?raw";

import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";

export type SplitRunCanvasKey = "intake" | "planning" | "implementation" | "risk" | "closure";

const CANVAS_KEYS: SplitRunCanvasKey[] = ["intake", "planning", "implementation", "risk", "closure"];

export function parseSplitRunCanvasKey(value: string | null | undefined): SplitRunCanvasKey | undefined {
  if (!value) {
    return undefined;
  }
  return CANVAS_KEYS.find((key) => key === value);
}

export interface SplitRunCanvasModel {
  key: SplitRunCanvasKey;
  title: string;
  nodes: ComponentsNode[];
  edges: ComponentsEdge[];
  statuses: Record<string, FactoryNodeStatus>;
  metrics: Record<string, string>;
}

const CANVAS_YAML: Record<SplitRunCanvasKey, string> = {
  intake: issueIntakeYaml,
  planning: planningYaml,
  implementation: implementationYaml,
  risk: riskAssessmentYaml,
  closure: prClosureYaml,
};

export function canvasKeyForPhase(phase: SplitRunPhase): SplitRunCanvasKey {
  const label = `${phase.id} ${phase.name} ${phase.componentName}`.toLowerCase();
  if (label.includes("backlog") || label.includes("intake") || label.includes("create work order")) {
    return "intake";
  }
  if (label.includes("plan")) {
    return "planning";
  }
  if (label.includes("implement")) {
    return "implementation";
  }
  if (label.includes("verify") || label.includes("risk") || label.includes("ci")) {
    return "risk";
  }
  return "closure";
}

export function splitRunCanvasForPhase(phase: SplitRunPhase): SplitRunCanvasModel {
  const key = canvasKeyForPhase(phase);
  const yaml = CANVAS_YAML[key];
  const spec = parseCanvasYamlToSpec(yaml);
  const metadata = parseCanvasYamlMetadata(yaml);
  if (!spec?.nodes?.length) {
    throw new Error(`Split-run canvas ${key} did not parse`);
  }

  const nodes = spec.nodes;
  const edges = spec.edges ?? [];
  const taken = takenNodeIds(key, nodes, edges);

  return {
    key,
    title: metadata?.name ?? phase.name,
    nodes,
    edges,
    statuses: paintStatuses(nodes, taken, phase.status),
    metrics: paintMetrics(nodes, taken, phase),
  };
}

function takenNodeIds(key: SplitRunCanvasKey, nodes: ComponentsNode[], edges: ComponentsEdge[]): Set<string> {
  const taken = new Set<string>();
  const starts = nodes
    .filter((node) => node.type === "TYPE_TRIGGER")
    .map((node) => node.id)
    .filter(Boolean) as string[];
  const queue = [...starts];

  while (queue.length > 0) {
    const id = queue.shift();
    if (!id || taken.has(id)) {
      continue;
    }
    taken.add(id);
    const outgoing = edges.filter((edge) => edge.sourceId === id);
    const preferred = preferredChannel(key, outgoing);
    for (const edge of outgoing) {
      if (!edge.targetId) {
        continue;
      }
      if (outgoing.length === 1 || edge.channel === preferred || edge.channel === "default") {
        if (edge.channel === "true" && preferred === "false") {
          continue;
        }
        queue.push(edge.targetId);
      }
    }
  }

  return taken;
}

function preferredChannel(key: SplitRunCanvasKey, edges: ComponentsEdge[]): string | undefined {
  if (key === "closure" && edges.some((edge) => edge.channel === "true")) {
    return "true";
  }
  if (edges.some((edge) => edge.channel === "false")) {
    return "false";
  }
  if (edges.some((edge) => edge.channel === "passed")) {
    return "passed";
  }
  return edges[0]?.channel;
}

function paintStatuses(
  nodes: ComponentsNode[],
  taken: Set<string>,
  status: SplitRunPhaseStatus,
): Record<string, FactoryNodeStatus> {
  const painted: Record<string, FactoryNodeStatus> = {};
  const takenList = nodes.filter((node) => node.id && taken.has(node.id));
  const lastTakenId = takenList.at(-1)?.id;

  for (const node of nodes) {
    if (!node.id) {
      continue;
    }
    if (!taken.has(node.id)) {
      painted[node.id] = status === "pending" ? "pending" : "did_not_run";
      continue;
    }
    if (node.type === "TYPE_TRIGGER") {
      painted[node.id] = status === "pending" ? "pending" : "triggered";
      continue;
    }
    if (status === "pending") {
      painted[node.id] = "pending";
      continue;
    }
    if (status === "running" && node.id === lastTakenId) {
      painted[node.id] = "running";
      continue;
    }
    if (status === "failed" && node.id === lastTakenId) {
      painted[node.id] = "failed";
      continue;
    }
    if ((status === "waiting" || status === "pending") && node.id === lastTakenId) {
      painted[node.id] = "pending";
      continue;
    }
    painted[node.id] = "passed";
  }

  return painted;
}

function paintMetrics(nodes: ComponentsNode[], taken: Set<string>, phase: SplitRunPhase): Record<string, string> {
  const metrics: Record<string, string> = {};
  for (const node of nodes) {
    if (!node.id) {
      continue;
    }
    if (!taken.has(node.id)) {
      metrics[node.id] = "—";
      continue;
    }
    if (phase.status === "running" && node.id === nodes.filter((item) => item.id && taken.has(item.id)).at(-1)?.id) {
      metrics[node.id] = phase.duration;
      continue;
    }
    if (phase.status === "pending" || phase.status === "waiting") {
      metrics[node.id] = node.type === "TYPE_TRIGGER" ? phase.duration : "—";
      continue;
    }
    metrics[node.id] = "18h ago";
  }
  return metrics;
}

export function componentPresentation(component?: string): { title: string; iconSlug: string } {
  switch (component) {
    case "onRun":
      return { title: "On Run", iconSlug: "play" };
    case "runnerBash":
      return { title: "Run Bash", iconSlug: "code" };
    case "runnerClaudeCode":
      return { title: "Run Claude Code", iconSlug: "code" };
    case "runnerJS":
      return { title: "Run JavaScript", iconSlug: "code" };
    case "if":
      return { title: "If", iconSlug: "git-branch" };
    case "filter":
      return { title: "Filter", iconSlug: "filter" };
    case "addWorkOrderArtifact":
      return { title: "Add Work Order Artifact", iconSlug: "file-text" };
    case "addRunError":
      return { title: "Add Run Error", iconSlug: "triangle-alert" };
    case "reportWorkOrderCheck":
      return { title: "Report Work Order Check", iconSlug: "clipboard-check" };
    case "github.createIssueComment":
      return { title: "Create Issue Comment", iconSlug: "message-square" };
    case "github.addIssueLabel":
      return { title: "Add Issue Label", iconSlug: "tag" };
    case "github.onIssue":
      return { title: "On Issue", iconSlug: "github" };
    case "github.onPullRequest":
      return { title: "On Pull Request", iconSlug: "git-pull-request" };
    case "findWorkOrder":
      return { title: "Find Work Order", iconSlug: "search" };
    case "updateWorkOrderArtifact":
      return { title: "Update Work Order Artifact", iconSlug: "file-pen" };
    case "createWorkOrder":
      return { title: "Create Work Order", iconSlug: "plus" };
    case "updateWorkOrderStatus":
      return { title: "Update Work Order Status", iconSlug: "circle-check" };
    default:
      return { title: component || "Component", iconSlug: "box" };
  }
}

const PLAN_ARTIFACT: FactoriesWorkOrderArtifact = {
  id: "art-plan-md",
  type: "TYPE_MARKDOWN",
  data: {
    name: "plan.md",
    title: "plan.md",
    body: "Add a focused test for the refund reconciliation worker.\nCover the timeout-then-retry path.",
  },
  createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
  createdAt: HOUR_AGO,
};

const CLOSURE_NOTES: FactoriesWorkOrderArtifact = {
  id: "art-closure-md",
  type: "TYPE_MARKDOWN",
  data: {
    name: "closure.md",
    title: "closure.md",
    body: "The pull request merged. SuperPlane closed the work order.",
  },
};

const MERGE_SCREENSHOT: FactoriesWorkOrderArtifact = {
  id: "art-merge-screenshot",
  type: "TYPE_PREVIEW",
  data: {
    name: "merge-screenshot.png",
    title: "merge-screenshot.png",
    url: "https://github.com/example/ledger/pull/510",
  },
};

/**
 * One log line per canvas node. Claude Code nodes add a short transcript.
 * Work-order artifacts and checks hang their file or score on that line.
 */
export function richStreamForCanvas(canvas: SplitRunCanvasModel): SplitRunStreamLine[] {
  const lines: SplitRunStreamLine[] = [];
  let tick = 0;

  for (const node of canvas.nodes) {
    if (!node.id) {
      continue;
    }
    const nodeStatus = canvas.statuses[node.id] ?? "pending";
    const componentName = node.name ?? componentPresentation(node.component).title;
    const lineStatus = streamStatusForNode(nodeStatus);
    const kind = classifyNode(node);

    if (kind === "agent" && nodeStatus !== "did_not_run" && nodeStatus !== "pending") {
      for (const note of agentNotes(node.id)) {
        lines.push({
          id: `${node.id}-note-${tick}`,
          nodeId: node.id,
          at: clockAt(tick),
          componentName: note,
          status: lineStatus,
          note: true,
        });
        tick += 1;
      }
    }

    lines.push({
      id: node.id,
      nodeId: node.id,
      at: clockAt(tick),
      componentName: kind === "check" ? checkLine(node.id, componentName) : componentName,
      status: lineStatus,
      artifact: kind === "check" || nodeStatus === "did_not_run" ? undefined : artifactForNode(node.id, canvas.key),
    });
    tick += 1;
  }

  return lines;
}

function classifyNode(node: ComponentsNode): "agent" | "check" | "artifact" | "simple" {
  const component = node.component ?? "";
  const name = `${node.name ?? ""} ${componentPresentation(component).title}`.toLowerCase();
  if (component === "runnerClaudeCode") {
    return "agent";
  }
  if (component === "reportWorkOrderCheck") {
    return "check";
  }
  if (component.toLowerCase().includes("workorder") || name.includes("work order")) {
    return "artifact";
  }
  return "simple";
}

function agentNotes(nodeId: string): string[] {
  if (nodeId.startsWith("planner-agent")) {
    return [
      "Reading the work order description.",
      "Drafting plan.md for the refund reconciliation test.",
      "Covering the timeout-then-retry path.",
    ];
  }
  if (nodeId.startsWith("implementation-agent")) {
    return ["Reading plan.md.", "Opening the refund reconciliation worker.", "Adding the timeout-then-retry test."];
  }
  if (nodeId === "assess-pr-risk") {
    return ["Reading the pull request diff.", "Scoring retry-policy risk.", "Writing the risk review."];
  }
  return ["Reading the work order.", "Writing the change.", "Running the local checks."];
}

function checkLine(nodeId: string, fallback: string): string {
  if (nodeId === "report-risk-check") {
    return "Risk review  65/100";
  }
  return fallback;
}

function streamStatusForNode(status: FactoryNodeStatus): SplitRunPhaseStatus {
  if (status === "running") return "running";
  if (status === "passed" || status === "triggered") return "passed";
  if (status === "failed") return "failed";
  return "pending";
}

function artifactForNode(nodeId: string, key: SplitRunCanvasKey): FactoriesWorkOrderArtifact | undefined {
  if (nodeId === "create-work-order") {
    return DESCRIPTION_ARTIFACT;
  }
  if (nodeId === "add-plan-artifact") {
    return PLAN_ARTIFACT;
  }
  if (nodeId === "add-branch-artifact") {
    return OPEN_WORK_ORDER_ARTIFACTS.find((artifact) => artifact.id === "art-branch-1");
  }
  if (nodeId === "stamp-pr-merged") {
    return PR_CLOSURE_PR_ARTIFACT;
  }
  if (nodeId === "find-work-order") {
    return MERGE_SCREENSHOT;
  }
  if (nodeId === "complete-work-order") {
    return CLOSURE_NOTES;
  }
  if (nodeId === "report-risk-check" && key === "risk") {
    return CLOSURE_NOTES;
  }
  return undefined;
}

function clockAt(offset: number): string {
  const total = 13 * 3600 + 50 * 60 + offset;
  const hours = Math.floor(total / 3600)
    .toString()
    .padStart(2, "0");
  const minutes = Math.floor((total % 3600) / 60)
    .toString()
    .padStart(2, "0");
  const seconds = (total % 60).toString().padStart(2, "0");
  return `${hours}:${minutes}:${seconds}`;
}
