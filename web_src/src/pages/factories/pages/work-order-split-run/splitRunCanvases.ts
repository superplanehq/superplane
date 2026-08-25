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
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import slackIcon from "@/assets/icons/integrations/slack.svg";
import { DESCRIPTION_ARTIFACT, PR_CLOSURE_PR_ARTIFACT } from "../work-order-popup-redesign/workOrderPopupMocks";
import riskAssessmentYaml from "./risk-assessment.canvas.yaml?raw";
import sentryIntakeYaml from "./sentry-intake.canvas.yaml?raw";
import slackIntakeYaml from "./slack-intake.canvas.yaml?raw";

import { parseClaudeCodeLog, type ClaudeCodeLogStep } from "./parseClaudeCodeLog";
import implementationClaudeLog from "./implementation-claude-log.txt?raw";
import planningClaudeLog from "./planning-claude-log.txt?raw";
import type { SplitRunPhase, SplitRunPhaseStatus, SplitRunStreamKind, SplitRunStreamLine } from "./splitRunMocks";

export type SplitRunCanvasKey = "intake" | "sentry" | "slack" | "planning" | "implementation" | "risk" | "closure";

const CANVAS_KEYS: SplitRunCanvasKey[] = ["intake", "sentry", "slack", "planning", "implementation", "risk", "closure"];

const CANVAS_HINTS: { needles: string[]; key: SplitRunCanvasKey }[] = [
  { needles: ["sentry"], key: "sentry" },
  { needles: ["slack"], key: "slack" },
  { needles: ["ingest", "intake", "backlog"], key: "intake" },
  { needles: ["plan"], key: "planning" },
  { needles: ["implement"], key: "implementation" },
  { needles: ["verify", "verifier", "risk", "ci"], key: "risk" },
  { needles: ["closure", "done"], key: "closure" },
];

export function parseSplitRunCanvasKey(value: string | null | undefined): SplitRunCanvasKey | undefined {
  if (!value) {
    return undefined;
  }
  return CANVAS_KEYS.find((key) => key === value);
}

function canvasKeyFromLabel(label: string): SplitRunCanvasKey | undefined {
  return CANVAS_HINTS.find((hint) => hint.needles.some((needle) => label.includes(needle)))?.key;
}

/** Map a factory automation onto a split-run canvas when the URL omits `canvas`. */
export function canvasKeyForAutomation(app: { id?: string; name?: string } | undefined): SplitRunCanvasKey | undefined {
  if (!app?.id && !app?.name) {
    return undefined;
  }
  return canvasKeyFromLabel(`${app.id ?? ""} ${app.name ?? ""}`.toLowerCase());
}

const LINE_AUTOMATION_LABEL: Record<SplitRunCanvasKey, { name: string; componentName: string }> = {
  intake: { name: "Backlog", componentName: "Ingest" },
  sentry: { name: "Backlog", componentName: "Sentry" },
  slack: { name: "Backlog", componentName: "Slack" },
  planning: { name: "Plan", componentName: "Planning" },
  implementation: { name: "Implement", componentName: "Implementation" },
  risk: { name: "Verify", componentName: "Risk Assessment" },
  closure: { name: "Done", componentName: "PR Closure" },
};

/** Column + canvas titles for a line step, even when the app still has an old name. */
export function lineAutomationPresentation(
  app: { id?: string; name?: string } | undefined,
  step?: string,
): { name: string; componentName: string } {
  const key = canvasKeyForAutomation(app) ?? canvasKeyFromLabel((step ?? "").toLowerCase());
  if (key) {
    return LINE_AUTOMATION_LABEL[key];
  }
  const fallback = step?.trim() || app?.name?.trim() || "Step";
  return { name: fallback, componentName: app?.name?.trim() || fallback };
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
  sentry: sentryIntakeYaml,
  slack: slackIntakeYaml,
  planning: planningYaml,
  implementation: implementationYaml,
  risk: riskAssessmentYaml,
  closure: prClosureYaml,
};

export function canvasKeyForPhase(phase: SplitRunPhase): SplitRunCanvasKey {
  if (phase.canvasKey) {
    return phase.canvasKey;
  }
  const label = `${phase.id} ${phase.name} ${phase.componentName}`.toLowerCase();
  return canvasKeyFromLabel(label) ?? "closure";
}

export function emptySplitRunCanvas(phase?: SplitRunPhase, title?: string): SplitRunCanvasModel {
  return {
    key: phase ? canvasKeyForPhase(phase) : "implementation",
    title: title ?? phase?.componentName ?? "",
    nodes: [],
    edges: [],
    statuses: {},
    metrics: {},
  };
}

export function splitRunCanvasForPhase(phase: SplitRunPhase): SplitRunCanvasModel {
  if (phase.canvas) {
    return phase.canvas;
  }
  if (phase.canvasKey === null) {
    return emptySplitRunCanvas(undefined, "");
  }
  const key = canvasKeyForPhase(phase);
  const yaml = CANVAS_YAML[key];
  const spec = parseCanvasYamlToSpec(yaml);
  const metadata = parseCanvasYamlMetadata(yaml);
  if (!spec?.nodes?.length) {
    throw new Error(`Split-run canvas ${key} did not parse`);
  }

  const nodes = spec.nodes;
  const edges = spec.edges ?? [];
  const taken = takenNodeIds(key, nodes, edges, phase.status, phase.triggerName);

  return {
    key,
    title: metadata?.name ?? phase.name,
    nodes,
    edges,
    statuses: paintStatuses(nodes, taken, phase.status),
    metrics: paintMetrics(nodes, taken, phase),
  };
}

function takenNodeIds(
  key: SplitRunCanvasKey,
  nodes: ComponentsNode[],
  edges: ComponentsEdge[],
  status: SplitRunPhaseStatus,
  triggerName?: string,
): Set<string> {
  const taken = new Set<string>();
  const starts = triggerIdsToWalk(nodes, triggerName);
  const queue = [...starts];

  while (queue.length > 0) {
    const id = queue.shift();
    if (!id || taken.has(id)) {
      continue;
    }
    taken.add(id);
    const outgoing = edges.filter((edge) => edge.sourceId === id);
    const preferred = preferredChannel(key, outgoing, status);
    for (const edge of outgoing) {
      if (!edge.targetId) {
        continue;
      }
      if (edge.channel === "failed" && status !== "failed") {
        continue;
      }
      const source = nodes.find((node) => node.id === id);
      if (edge.channel === "passed" && status === "running" && source?.component === "runnerClaudeCode") {
        continue;
      }
      if (outgoing.length === 1 || edge.channel === preferred || edge.channel === "default") {
        queue.push(edge.targetId);
      }
    }
  }

  return taken;
}

function triggerIdsToWalk(nodes: ComponentsNode[], triggerName?: string): string[] {
  const triggers = nodes.filter((node) => node.type === "TYPE_TRIGGER" && node.id);
  if (triggerName) {
    const named = triggers.find((node) => node.name === triggerName);
    if (named?.id) {
      return [named.id];
    }
  }
  return triggers.map((node) => node.id).filter((id): id is string => Boolean(id));
}

function preferredChannel(
  _key: SplitRunCanvasKey,
  edges: ComponentsEdge[],
  status: SplitRunPhaseStatus,
): string | undefined {
  if (status === "failed" && edges.some((edge) => edge.channel === "failed")) {
    return "failed";
  }
  if (edges.some((edge) => edge.channel === "true")) {
    return "true";
  }
  if (edges.some((edge) => edge.channel === "passed")) {
    return "passed";
  }
  return edges.find((edge) => edge.channel !== "failed")?.channel ?? edges[0]?.channel;
}

function paintStatuses(
  nodes: ComponentsNode[],
  taken: Set<string>,
  status: SplitRunPhaseStatus,
): Record<string, FactoryNodeStatus> {
  const painted: Record<string, FactoryNodeStatus> = {};
  const lastTakenId = nodes.filter((node) => node.id && taken.has(node.id)).at(-1)?.id;

  for (const node of nodes) {
    if (!node.id) {
      continue;
    }
    painted[node.id] = statusForPaintedNode({
      taken: taken.has(node.id),
      isTrigger: node.type === "TYPE_TRIGGER",
      isLastTaken: node.id === lastTakenId,
      status,
    });
  }

  return painted;
}

function statusForPaintedNode(input: {
  taken: boolean;
  isTrigger: boolean;
  isLastTaken: boolean;
  status: SplitRunPhaseStatus;
}): FactoryNodeStatus {
  if (!input.taken) {
    return input.status === "pending" ? "pending" : "did_not_run";
  }
  if (input.isTrigger) {
    return input.status === "pending" ? "pending" : "triggered";
  }
  if (input.status === "pending") {
    return "pending";
  }
  if (input.isLastTaken && input.status === "running") {
    return "running";
  }
  if (input.isLastTaken && input.status === "failed") {
    return "failed";
  }
  if (input.isLastTaken && input.status === "waiting") {
    return "pending";
  }
  return "passed";
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

const COMPONENT_PRESENTATION: Record<string, { title: string; iconSlug: string; iconSrc?: string }> = {
  onRun: { title: "On Run", iconSlug: "play" },
  runnerBash: { title: "Run Bash", iconSlug: "code" },
  runnerClaudeCode: { title: "Run Claude Code", iconSlug: "code" },
  runnerJS: { title: "Run JavaScript", iconSlug: "code" },
  if: { title: "If", iconSlug: "split" },
  filter: { title: "Filter", iconSlug: "funnel" },
  addWorkOrderArtifact: { title: "Add Work Order Artifact", iconSlug: "factory" },
  addRunError: { title: "Add Run Error", iconSlug: "triangle-alert" },
  reportWorkOrderCheck: { title: "Report Work Order Check", iconSlug: "factory" },
  "github.createIssueComment": { title: "Create Issue Comment", iconSlug: "github" },
  "github.createPullRequest": { title: "Create Pull Request", iconSlug: "github" },
  "github.addIssueLabel": { title: "Add Issue Label", iconSlug: "github" },
  "github.onIssue": { title: "On Issue", iconSlug: "github" },
  "github.onPullRequest": { title: "On Pull Request", iconSlug: "github" },
  "sentry.onIssue": { title: "On Issue", iconSlug: "bug", iconSrc: sentryIcon },
  "slack.onAppMention": { title: "On Mention", iconSlug: "slack", iconSrc: slackIcon },
  "pagerduty.onIncident": { title: "On Incident", iconSlug: "pagerduty" },
  findWorkOrder: { title: "Find Work Order", iconSlug: "factory" },
  updateWorkOrderArtifact: { title: "Update Work Order Artifact", iconSlug: "factory" },
  createWorkOrder: { title: "Create Work Order", iconSlug: "factory" },
  updateWorkOrderStatus: { title: "Update Work Order Status", iconSlug: "factory" },
};

export function componentPresentation(component?: string): { title: string; iconSlug: string; iconSrc?: string } {
  if (!component) {
    return { title: "Component", iconSlug: "box" };
  }
  return COMPONENT_PRESENTATION[component] ?? { title: component, iconSlug: "box" };
}

/** Integration ids stay namespaced. Core components use their catalog label. */
export function componentTypeLabel(component?: string): string {
  if (!component) {
    return "Component";
  }
  if (component.includes(".")) {
    return component;
  }
  return componentPresentation(component).title;
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
  type: "TYPE_LINK",
  data: {
    name: "merge-screenshot.png",
    title: "merge-screenshot.png",
    url: "https://github.com/example/ledger/pull/510",
  },
};

/**
 * One log line per canvas node. Claude Code nodes add their configured steps.
 * Work-order artifacts and checks hang their file or score on that line.
 */
export function richStreamForCanvas(
  canvas: SplitRunCanvasModel,
  description?: FactoriesWorkOrderArtifact,
  options?: { demoArtifacts?: boolean },
): SplitRunStreamLine[] {
  const lines: SplitRunStreamLine[] = [];
  let tick = 0;

  for (const node of canvas.nodes) {
    if (!node.id) {
      continue;
    }
    const nodeStatus = canvas.statuses[node.id] ?? "pending";
    const presentation = componentPresentation(node.component);
    const componentName = node.name ?? presentation.title;
    const lineStatus = streamStatusForNode(nodeStatus);
    const kind = classifyNode(node);
    const streamKind = streamKindForNode(node);

    const artifact =
      kind === "check" || nodeStatus === "did_not_run"
        ? undefined
        : artifactForNode(node.id, canvas.key, description, options?.demoArtifacts !== false);
    const name = kind === "check" ? checkName(node.id, componentName) : componentName;
    lines.push({
      id: node.id,
      nodeId: node.id,
      at: clockAt(tick),
      componentName: name,
      status: lineStatus,
      artifact,
      kind: streamKind,
      componentType: componentTypeLabel(node.component),
      action: actionForStreamLine(streamKind, nodeStatus, node.id),
      iconSlug: presentation.iconSlug,
      iconSrc: presentation.iconSrc,
    });
    tick += 1;

    if (kind === "agent" && nodeStatus !== "did_not_run" && nodeStatus !== "pending") {
      for (const step of claudeCodeChildren(node, canvas.key, options?.demoArtifacts !== false)) {
        const stepId = `${node.id}-note-${tick}`;
        lines.push({
          id: stepId,
          nodeId: node.id,
          at: clockAt(tick),
          componentName: step.name,
          status: step.status,
          detail: step.output,
          note: true,
          componentType: step.type,
        });
        tick += 1;
        for (const command of step.commands) {
          lines.push({
            id: `${node.id}-cmd-${tick}`,
            nodeId: node.id,
            at: clockAt(tick),
            componentName: command.name,
            status: command.status,
            detail: command.output,
            note: true,
            noteParentId: stepId,
            noteDepth: 1,
            componentType: command.type,
          });
          tick += 1;
        }
      }
    }
  }

  return lines;
}

export function streamKindForNode(node: ComponentsNode): SplitRunStreamKind {
  if (node.type === "TYPE_TRIGGER") {
    return "trigger";
  }
  const component = node.component ?? "";
  if (component === "filter") {
    return "filter";
  }
  if (component === "if") {
    return "if";
  }
  if (component === "runnerClaudeCode") {
    return "agent";
  }
  if (component === "reportWorkOrderCheck") {
    return "check";
  }
  return "action";
}

function actionForStreamLine(kind: SplitRunStreamKind, nodeStatus: FactoryNodeStatus, nodeId: string): string {
  if (nodeStatus === "did_not_run") {
    return "did not run";
  }
  if (nodeStatus === "pending") {
    return "—";
  }
  if (nodeStatus === "failed") {
    return "failed";
  }
  if (nodeStatus === "running") {
    return "running";
  }
  if (kind === "check") {
    return checkScore(nodeId);
  }
  if (kind === "trigger" || nodeStatus === "triggered") {
    return "triggered";
  }
  return "passed";
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

export function claudeCodeSteps(node: ComponentsNode): Array<{ name: string; type: string }> {
  const configured = node.configuration?.steps;
  if (!Array.isArray(configured) || configured.length === 0) {
    return agentNotes(node.id).map((name) => ({ name, type: "" }));
  }
  const steps: Array<{ name: string; type: string }> = [];
  for (const step of configured) {
    if (!step || typeof step !== "object") {
      continue;
    }
    const name = "name" in step && typeof step.name === "string" ? step.name.trim() : "";
    const type = "type" in step && typeof step.type === "string" ? step.type.trim() : "";
    if (name) {
      steps.push({ name, type });
    }
  }
  return steps;
}

const CLAUDE_CODE_LOGS: Partial<Record<SplitRunCanvasKey, string>> = {
  planning: planningClaudeLog,
  implementation: implementationClaudeLog,
};

function claudeCodeChildren(
  node: ComponentsNode,
  canvasKey: SplitRunCanvasKey,
  demoArtifacts: boolean,
): ClaudeCodeLogStep[] {
  const configured = claudeCodeSteps(node);
  const log = demoArtifacts ? CLAUDE_CODE_LOGS[canvasKey] : undefined;
  if (log) {
    return parseClaudeCodeLog(log, configured);
  }
  return configured.map((step) => ({ ...step, commands: [] }));
}

function agentNotes(nodeId: string): string[] {
  if (nodeId.startsWith("planner-agent")) {
    return ["Clone Repo", "Write Implementation Plan", "Use plan as output"];
  }
  if (nodeId.startsWith("implementation-agent")) {
    return ["Reading plan.md.", "Opening the refund reconciliation worker.", "Adding the timeout-then-retry test."];
  }
  if (nodeId === "assess-pr-risk") {
    return ["Reading the pull request diff.", "Scoring retry-policy risk.", "Writing the risk review."];
  }
  return ["Reading the work order.", "Writing the change.", "Running the local checks."];
}

function checkName(nodeId: string, fallback: string): string {
  if (nodeId === "report-risk-check") {
    return "Risk review";
  }
  return fallback;
}

function checkScore(nodeId: string): string {
  if (nodeId === "report-risk-check") {
    return "65/100";
  }
  return "passed";
}

function streamStatusForNode(status: FactoryNodeStatus): SplitRunPhaseStatus {
  if (status === "running") return "running";
  if (status === "passed" || status === "triggered") return "passed";
  if (status === "failed") return "failed";
  return "pending";
}

function artifactForNode(
  nodeId: string,
  key: SplitRunCanvasKey,
  description?: FactoriesWorkOrderArtifact,
  demoArtifacts = true,
): FactoriesWorkOrderArtifact | undefined {
  if (nodeId === "create-work-order") {
    return demoArtifacts ? (description ?? DESCRIPTION_ARTIFACT) : description;
  }
  if (!demoArtifacts) {
    return undefined;
  }
  if (nodeId === "add-plan-artifact") {
    return PLAN_ARTIFACT;
  }
  if (nodeId === "add-branch-artifact") {
    return OPEN_WORK_ORDER_ARTIFACTS.find((artifact) => artifact.id === "art-branch-1");
  }
  if (nodeId === "attach-pr-artifact") {
    return OPEN_WORK_ORDER_ARTIFACTS.find((artifact) => artifact.id === "art-pr-1");
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
