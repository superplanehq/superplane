import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

import { findClosureAutomationApp } from "../../../lib/linePhaseRuns";
import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { formatCompactTokens, formatUsdCents, parseWorkOrderMetric } from "../../../lib/workOrderUsage";
import { groupSplitRunActivities, type PullRequestActivityGroup } from "../splitRunActivityGroups";
import { groupClaudeSteps, groupSplitRunStream, type ClaudeStepGroup, type StreamNodeGroup } from "../phaseLogStream";
import { agentStepStatusFromSection } from "../streamNotesFromLiveLog";
import {
  SPLIT_RUN_CLOSURE_PHASE_ID,
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
  promptStatus?: SplitRunPhaseStatus;
  duration?: string;
  summary: string;
  toolCount: number;
  output?: string;
  commandScript?: string;
  commandStdout?: string;
  events: AgentStepEvent[];
  iconSlug?: string;
}

/** One automation with every run it made for this task, oldest first. */
export interface ConsoleAutomation {
  id: string;
  name: string;
  latest: AutomationStage;
  runs: AutomationStage[];
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
  /** Board column this app sits on, when it is a column automation. */
  columnKey?: SplitRunPhase["columnKey"];
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

/**
 * A column lists canvas runs, plus the task-creation and close-decision
 * stages. Those stages have no canvas run when a person created or
 * closed the task.
 */
export function isConsoleTaskStage(stage: Pick<AutomationStage, "id" | "appId">): boolean {
  return Boolean(stage.appId) || stage.id === "backlog" || stage.id === SPLIT_RUN_CLOSURE_PHASE_ID;
}

/** The synthetic create-task stage. The console Intake event shows it. */
export function isConsoleCreationStage(stage: Pick<AutomationStage, "id">): boolean {
  return stage.id === "backlog";
}

export function allStages(groups: AutomationStageGroups): AutomationStage[] {
  return [...groups.taskStages, ...groups.pullRequestGroups.flatMap((group) => group.stages)];
}

export type ConsoleColumnId = "backlog" | "implement" | "verify" | "done";

/**
 * Task stages sit in the column named after them. A column app uses the
 * column it is installed on. The creation stage sits in Backlog even
 * when no automation ran. Pull request activity sits in Verify, except
 * a closer run or an app installed on another column. A custom step
 * name still appears, in Implement, so the run does not drop off the
 * timeline.
 */
export function consoleColumnIdForStage(
  stage: Pick<AutomationStage, "id" | "name" | "columnKey" | "appId" | "componentName" | "pullRequestActivity">,
): ConsoleColumnId {
  if (stage.columnKey) {
    return stage.columnKey;
  }
  if (isFactoryClosureStage(stage) || stage.id === SPLIT_RUN_CLOSURE_PHASE_ID || stage.name === "Done") {
    return "done";
  }
  if (stage.name === "Verify" || stage.pullRequestActivity) {
    return "verify";
  }
  if (stage.name === "Backlog" || stage.name === "Analysis") {
    return "backlog";
  }
  return "implement";
}

/** Built-in PR Closure has no column key. It still belongs in Done. */
function isFactoryClosureStage(stage: Pick<AutomationStage, "appId" | "componentName" | "columnKey">): boolean {
  return Boolean(
    findClosureAutomationApp([{ id: stage.appId, name: stage.componentName, columnKey: stage.columnKey }]),
  );
}

/**
 * One card per automation, not per run. Comment replies share a handler
 * name even when each run has its own canvas id, so the name is the key.
 */
export function consoleAutomationKey(stage: Pick<AutomationStage, "id" | "appId" | "componentName">): string {
  return stage.componentName.trim() || stage.appId || stage.id;
}

export function automationsFromStages(stages: AutomationStage[]): ConsoleAutomation[] {
  const byKey = new Map<string, AutomationStage[]>();
  for (const stage of stages) {
    const key = consoleAutomationKey(stage);
    const runs = byKey.get(key) ?? [];
    runs.push(stage);
    byKey.set(key, runs);
  }
  return [...byKey.entries()].map(([key, runs]) => {
    const oldestFirst = [...runs].sort(
      (left, right) => Date.parse(left.startedAt ?? "") - Date.parse(right.startedAt ?? ""),
    );
    const newest = oldestFirst.at(-1);
    const name = newest?.componentName || key;
    const latest = oldestFirst.find((run) => run.status === "running") ?? newest;
    return {
      id: consoleAutomationDomId(name, key),
      name,
      latest: latest ?? oldestFirst[0],
      runs: oldestFirst,
    };
  });
}

function consoleAutomationDomId(name: string, key: string): string {
  const slug = `${name}-${key}`.toLowerCase().replace(/\W+/g, "-").replace(/^-|-$/g, "");
  return slug || "automation";
}

export function stagesByConsoleColumn(
  groups: AutomationStageGroups,
  closerAppId?: string,
): Record<ConsoleColumnId, AutomationStage[]> {
  const pullRequestStages = groups.pullRequestGroups.flatMap((group) => group.stages).filter((stage) => stage.appId);
  const closedBy = (stage: AutomationStage) => Boolean(closerAppId) && stage.appId === closerAppId;
  const columns: Record<ConsoleColumnId, AutomationStage[]> = {
    backlog: [],
    implement: [],
    verify: [],
    done: [],
  };
  for (const stage of groups.taskStages.filter(isConsoleTaskStage)) {
    columns[consoleColumnIdForStage(stage)].push(stage);
  }
  for (const stage of pullRequestStages) {
    if (closedBy(stage)) {
      columns.done.push(stage);
      continue;
    }
    columns[consoleColumnIdForStage(stage)].push(stage);
  }
  return columns;
}

export function stageFromPhase(phase: SplitRunPhase): AutomationStage {
  const nodes = groupSplitRunStream(phase.stream);
  const agentSteps = agentStepsFromNotes(phase.stream, phase.status);
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
    steps: runSteps(nodes, phase.status),
    rawLog: rawLogFromSteps(agentSteps),
    columnKey: phase.columnKey,
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
function runSteps(nodes: StreamNodeGroup[], runStatus?: SplitRunPhaseStatus): AgentStep[] {
  return applyPromptStepRunStatus(flattenRunSteps(nodes), runStatus);
}

function flattenRunSteps(nodes: StreamNodeGroup[]): AgentStep[] {
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
    output: line.detail?.trim() || undefined,
    events: [],
    iconSlug: line.iconSlug,
  };
}

export function agentStepsFromNotes(
  notes: SplitRunStreamLine[],
  runStatus?: SplitRunPhaseStatus,
  canvasLines?: SplitRunStreamLine[],
): AgentStep[] {
  const steps = groupClaudeSteps(notes.filter((line) => line.note)).map(agentStepFromGroup);
  return applyPromptStepRunStatus(steps, runStatus, stoppingIndexForTranscript(notes, steps, canvasLines));
}

function applyPromptStepRunStatus(
  steps: AgentStep[],
  runStatus?: SplitRunPhaseStatus,
  stoppedIndex = stoppingStepIndex(steps),
): AgentStep[] {
  return steps.map((step, index) => {
    if (step.type !== "prompt") {
      return step;
    }
    const sectionFailed = step.promptStatus === "failed" || step.status === "failed";
    const promptStatus = sectionFailed ? "failed" : step.promptStatus;
    const status =
      step.status === "failed"
        ? agentStepStatusFromSection({
            kind: "prompt",
            sectionStatus: "failed",
            runStatus,
            stoppedTheRun: index === stoppedIndex,
          })
        : step.status;
    if (status === step.status && promptStatus === step.promptStatus) {
      return step;
    }
    return { ...step, status, promptStatus };
  });
}

function stoppingIndexForTranscript(
  notes: SplitRunStreamLine[],
  transcript: AgentStep[],
  canvasLines?: SplitRunStreamLine[],
): number {
  const lines = canvasLines ? mergeCanvasLines(canvasLines, notes) : notes;
  const nodes = groupSplitRunStream(lines);
  if (nodes.length === 0) {
    return stoppingStepIndex(transcript);
  }
  const run = flattenRunSteps(nodes);
  const stopped = run[stoppingStepIndex(run)];
  if (!stopped || stopped.type === "node") {
    return -1;
  }
  return transcript.findIndex((step) => step.id === stopped.id);
}

function mergeCanvasLines(canvasLines: SplitRunStreamLine[], notes: SplitRunStreamLine[]): SplitRunStreamLine[] {
  const seen = new Set(canvasLines.map((line) => line.id));
  return [...canvasLines, ...notes.filter((line) => !seen.has(line.id))];
}

function stoppingStepIndex(steps: AgentStep[]): number {
  for (let index = steps.length - 1; index >= 0; index -= 1) {
    const step = steps[index];
    if (!step || step.status !== "failed") {
      continue;
    }
    if (step.type === "prompt" || step.type === "bash" || step.type === "node") {
      return index;
    }
  }
  return -1;
}

/** A canceled or failed stage no longer has a step in progress. */
export function settleStoppedSteps(steps: AgentStep[], status?: SplitRunPhaseStatus): AgentStep[] {
  if (status !== "cancelled" && status !== "failed") {
    return steps;
  }
  return steps.map((step) => (step.status === "running" ? { ...step, status } : step));
}

function agentStepFromGroup(group: ClaudeStepGroup): AgentStep {
  const tools = group.events.flatMap((event) => (event.kind === "tools" ? event.tools : []));
  return {
    id: group.line.id,
    title: group.line.componentName,
    type: group.line.componentType === "bash" ? "bash" : "prompt",
    status: group.line.status,
    promptStatus: group.line.promptStatus,
    duration: group.line.duration,
    summary: "",
    toolCount: tools.length,
    output: group.line.detail?.trim() || undefined,
    commandScript: group.line.commandScript,
    commandStdout: group.line.commandStdout,
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
