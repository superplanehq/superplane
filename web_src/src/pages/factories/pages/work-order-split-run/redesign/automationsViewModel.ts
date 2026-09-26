import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../../lib/workOrderUsage";
import { groupSplitRunActivities, type PullRequestActivityGroup } from "../splitRunActivityGroups";
import {
  groupClaudeSteps,
  groupSplitRunStream,
  toolCallSummary,
  type ClaudeStepGroup,
  type StreamNodeGroup,
} from "../phaseLogStream";
import {
  splitRunStatusLabel,
  type SplitRunFixture,
  type SplitRunPhase,
  type SplitRunPhaseStatus,
  type SplitRunStreamLine,
} from "../splitRunMocks";

/**
 * One derived model for every Automations tab redesign variant. The
 * variants differ in presentation only, so they all read from here. To
 * wire a variant to live data later, swap the fixture input.
 */

export interface AutomationOutcome {
  statusLabel: string;
  status: SplitRunPhaseStatus;
  duration: string;
  startedLabel: string;
  spend: string;
  tokens: string;
  models: string[];
  pullRequests: FactoriesFactoryPullRequest[];
  checksPassed: number;
  checksTotal: number;
  owner: OrgUserDisplay;
  headline: string;
}

export interface AgentToolRow {
  id: string;
  type: string;
  name: string;
  status: SplitRunPhaseStatus;
  output?: string;
}

export type AgentStepEvent =
  | { kind: "note"; id: string; text: string }
  | { kind: "tools"; id: string; label: string; tools: AgentToolRow[] };

/**
 * One row in a run's step list. `prompt` and `bash` come from the agent
 * transcript. `node` is a canvas node without a transcript, such as a
 * trigger or a wait-for-checks action.
 */
export interface AgentStep {
  id: string;
  title: string;
  type: "prompt" | "bash" | "node";
  status: SplitRunPhaseStatus;
  duration?: string;
  summary: string;
  toolCount: number;
  output?: string;
  events: AgentStepEvent[];
  iconSlug?: string;
}

export interface PlumbingNode {
  id: string;
  at: string;
  name: string;
  componentType: string;
  kind: "trigger" | "action";
  status: SplitRunPhaseStatus;
  duration?: string;
  iconSlug?: string;
}

export interface StageOutputs {
  pullRequests: FactoriesFactoryPullRequest[];
  artifacts: FactoriesWorkOrderArtifact[];
}

export interface AutomationStage {
  id: string;
  name: string;
  description?: string;
  componentName: string;
  /** Automation that ran. Empty for rows no automation made, such as task creation. */
  appId?: string;
  status: SplitRunPhaseStatus;
  statusLabel: string;
  startedAt?: string;
  duration: string;
  cost?: string;
  tokens?: string;
  model?: string;
  checks: WorkOrderCheckPresentation[];
  outputs: StageOutputs;
  plumbing: PlumbingNode[];
  /** Agent transcript steps only. Empty for plumbing-only runs. */
  agentSteps: AgentStep[];
  /**
   * Every step of the run in order: canvas nodes as `node` steps, with the
   * agent node replaced by its transcript steps.
   */
  steps: AgentStep[];
  rawLog: string;
  pullRequestActivity?: SplitRunPhase["pullRequestActivity"];
}

export interface AutomationStageGroups {
  taskStages: AutomationStage[];
  pullRequestGroups: Array<{ id: string; pullRequest?: FactoriesFactoryPullRequest; stages: AutomationStage[] }>;
}

const CHECK_PASS_LEVELS = new Set(["positive", "neutral"]);

export function outcomeSummary(fixture: SplitRunFixture): AutomationOutcome {
  const pullRequests = uniquePullRequests(
    fixture.phases.flatMap((phase) =>
      phase.stream.filter((line) => line.action !== "did not run").map((line) => line.pullRequest),
    ),
  );
  const models = uniqueModels(fixture);
  const checksPassed = fixture.checks.filter((check) => CHECK_PASS_LEVELS.has(check.level)).length;
  return {
    statusLabel: splitRunStatusLabel(fixture.lineStatus),
    status: fixture.lineStatus,
    duration: fixture.elapsed,
    startedLabel: fixture.startedLabel,
    spend: fixture.costUsd,
    tokens: fixture.tokensLabel,
    models,
    pullRequests,
    checksPassed,
    checksTotal: fixture.checks.length,
    owner: fixture.owner,
    headline: fixture.footer.note?.headline ?? splitRunStatusLabel(fixture.lineStatus),
  };
}

export function stagesFromFixture(fixture: SplitRunFixture): AutomationStageGroups {
  const groups = groupSplitRunActivities(fixture.phases);
  return {
    taskStages: groups.taskAutomationPhases.map(stageFromPhase),
    pullRequestGroups: groups.pullRequestActivityGroups.map((group: PullRequestActivityGroup) => ({
      id: group.id,
      pullRequest: group.pullRequest,
      stages: group.phases.map(stageFromPhase),
    })),
  };
}

export function allStages(groups: AutomationStageGroups): AutomationStage[] {
  return [...groups.taskStages, ...groups.pullRequestGroups.flatMap((group) => group.stages)];
}

export function stageFromPhase(phase: SplitRunPhase): AutomationStage {
  const nodes = groupSplitRunStream(phase.stream);
  const agentNotes = phase.stream.filter((line) => line.note);
  const agentSteps = groupClaudeSteps(agentNotes).map(agentStepFromGroup);
  const plumbing = nodes.map(plumbingNodeFromGroup);
  const cost = parseWorkOrderMetric(phase.costCents);
  const tokens = parseWorkOrderMetric(phase.totalTokens);
  return {
    id: phase.id,
    name: phase.name,
    description: phase.description,
    componentName: phase.componentName,
    appId: phase.appId,
    status: phase.status,
    statusLabel: splitRunStatusLabel(phase.status),
    startedAt: phase.startedAt,
    duration: phase.duration,
    cost: cost > 0 ? formatUsdCents(cost) : undefined,
    tokens: tokens > 0 ? formatCompactTokens(tokens) : undefined,
    model: modelDisplayName(phase.model),
    checks: phase.checks ?? [],
    outputs: {
      pullRequests: uniquePullRequests(nodes.map((node) => node.pullRequest)),
      artifacts: uniqueArtifacts([...phase.artifacts, ...nodes.map((node) => node.artifact)]),
    },
    plumbing,
    agentSteps,
    steps: runSteps(nodes),
    rawLog: rawLogFromSteps(agentSteps),
    pullRequestActivity: phase.pullRequestActivity,
  };
}

function plumbingNodeFromGroup({ line }: StreamNodeGroup): PlumbingNode {
  return {
    id: line.id,
    at: line.at,
    name: line.componentName,
    componentType: line.componentType ?? "",
    kind: line.kind === "trigger" ? "trigger" : "action",
    status: line.status,
    duration: line.duration,
    iconSlug: line.iconSlug,
  };
}

/**
 * Flattens a run into one step list. A node with a transcript contributes
 * its transcript steps in its place. Any other node is one `node` step.
 */
function runSteps(nodes: StreamNodeGroup[]): AgentStep[] {
  return nodes.flatMap((node) => {
    const transcript = groupClaudeSteps(node.notes).map(agentStepFromGroup);
    return transcript.length > 0 ? transcript : [nodeStep(node)];
  });
}

function nodeStep({ line }: StreamNodeGroup): AgentStep {
  return {
    id: line.id,
    title: line.componentName,
    type: "node",
    status: line.status,
    duration: line.duration,
    summary: "",
    toolCount: 0,
    events: [],
    iconSlug: line.iconSlug,
  };
}

function agentStepFromGroup(group: ClaudeStepGroup): AgentStep {
  const tools = group.events.flatMap((event) => (event.kind === "tools" ? event.tools : []));
  return {
    id: group.line.id,
    title: group.line.componentName,
    type: group.line.componentType === "bash" ? "bash" : "prompt",
    status: group.line.status,
    duration: group.line.duration,
    summary: tools.length > 0 ? toolCallSummary(tools) : "",
    toolCount: tools.length,
    output: group.line.detail?.trim() || undefined,
    events: group.events.map((event) =>
      event.kind === "note"
        ? { kind: "note", id: event.line.id, text: event.line.componentName }
        : {
            kind: "tools",
            id: event.id,
            label: event.tools.length === 1 ? "1 tool call" : `${event.tools.length} tool calls`,
            tools: event.tools.map(toolRow),
          },
    ),
  };
}

function toolRow(line: SplitRunStreamLine): AgentToolRow {
  return {
    id: line.id,
    type: line.componentType ?? "tool",
    name: line.componentName,
    status: line.status,
    output: line.detail?.trim() || undefined,
  };
}

/** Plain-text log: `$ step`, indented tool lines, then output. */
export function rawLogFromSteps(steps: AgentStep[]): string {
  const lines: string[] = [];
  for (const step of steps) {
    lines.push(`$ ${step.title}`);
    for (const event of step.events) {
      if (event.kind === "note") {
        lines.push(`  ${event.text}`);
        continue;
      }
      for (const tool of event.tools) {
        lines.push(`  [${tool.type}] ${tool.name}`);
        if (tool.output) {
          lines.push(...indent(tool.output, 4));
        }
      }
    }
    if (step.output) {
      lines.push(...indent(step.output, 2));
    }
    lines.push("");
  }
  return lines.join("\n").trimEnd();
}

function indent(text: string, spaces: number): string[] {
  const pad = " ".repeat(spaces);
  return text.split("\n").map((line) => `${pad}${line}`);
}

function artifactName(artifact: FactoriesWorkOrderArtifact): string {
  const data = artifact.data as Record<string, unknown> | undefined;
  const name = data?.name ?? data?.title;
  return typeof name === "string" ? name : "";
}

function uniquePullRequests(items: Array<FactoriesFactoryPullRequest | undefined>): FactoriesFactoryPullRequest[] {
  const seen = new Map<string, FactoriesFactoryPullRequest>();
  for (const item of items) {
    if (!item) continue;
    const key = item.id ?? item.url ?? `${item.repository}#${item.number}`;
    if (!seen.has(key)) {
      seen.set(key, item);
    }
  }
  return [...seen.values()];
}

function uniqueArtifacts(items: Array<FactoriesWorkOrderArtifact | undefined>): FactoriesWorkOrderArtifact[] {
  const seen = new Map<string, FactoriesWorkOrderArtifact>();
  for (const item of items) {
    if (!item) continue;
    const key = item.id ?? artifactName(item);
    if (!seen.has(key)) {
      seen.set(key, item);
    }
  }
  return [...seen.values()];
}

function uniqueModels(fixture: SplitRunFixture): string[] {
  const fromUsage = (fixture.usageByModel ?? []).map((row) => modelDisplayName(row.model));
  const fromPhases = fixture.phases.map((phase) => modelDisplayName(phase.model));
  return [...new Set([...fromUsage, ...fromPhases].filter((model): model is string => Boolean(model)))];
}

export function modelDisplayName(model?: string): string | undefined {
  const raw = model?.trim();
  if (!raw) {
    return undefined;
  }
  const slash = raw.lastIndexOf("/");
  return slash >= 0 && slash < raw.length - 1 ? raw.slice(slash + 1) : raw;
}
