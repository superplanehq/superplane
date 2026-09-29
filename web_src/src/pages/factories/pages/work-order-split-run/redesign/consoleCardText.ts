import { formatCheckScore } from "../../../lib/workOrderChecks";
import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { SPLIT_RUN_CLOSURE_PHASE_ID } from "../splitRunMocks";
import type { AgentStep, AutomationStage } from "./automationsViewModel";
import { formatClock } from "./redesignFormat";

/**
 * One line of outcome for the collapsed row. A running stage names the
 * agent step it is on. A pull request activity shows its title, as
 * production records it. A line step shows its description, then the
 * pull request, then the first check, then artifact names. Canvas nodes
 * never appear here: the canvas run page shows those.
 */
export function runResultLine(stage: AutomationStage): string {
  if (stage.status === "running") {
    const live = liveAgentLine(stage);
    if (live) {
      return live;
    }
  }
  const title = runTitle(stage);
  if (title) {
    return plainText(title);
  }
  if (stage.description) {
    return plainText(stage.description);
  }
  const pullRequest = stage.outputs.pullRequests[0];
  if (pullRequest) {
    return `#${pullRequest.number} ${pullRequest.title ?? ""}`.trim();
  }
  const check = stage.checks[0];
  if (check) {
    return check.summary ?? check.name;
  }
  return stage.outputs.artifacts.map(artifactLabel).filter(Boolean).join(" · ");
}

/** The agent step a running stage is on. Empty when the run has no agent. */
export function activeAgentStep(stage: AutomationStage): AgentStep | undefined {
  return stage.agentSteps.find((step) => step.status === "running");
}

/**
 * "Implementation · 49 tool calls" while the agent works. Once the agent
 * has finished and the run is wrapping up, its last step's summary.
 */
function liveAgentLine(stage: AutomationStage): string {
  const active = activeAgentStep(stage);
  if (active) {
    return [active.title, activeStepProgress(active)].filter(Boolean).join(" · ");
  }
  const last = stage.agentSteps.at(-1);
  return last?.summary || last?.title || "";
}

/** Spend and duration of the latest run for the collapsed row. */
export function runMetaLine(stage: AutomationStage): string {
  const duration = stage.status === "running" && stage.duration ? `${stage.duration} so far` : stage.duration;
  return [stage.cost, stage.tokens, duration].filter(Boolean).join(" · ");
}

/**
 * Pull request activity carries the run title in `name`: "Checks passed
 * on 99951af", "@greptile-apps[bot] left a review". Line steps carry the
 * column name there instead, which says nothing about the run.
 */
function runTitle(stage: AutomationStage): string | undefined {
  return stage.pullRequestActivity ? stage.name : undefined;
}

export interface StepOutputSummary {
  artifactCount: number;
  artifactLabels: string[];
  checkCount: number;
  checkLines: string[];
}

/** Artifact and check counts for one step, plus the hover-card lines. */
export function stepOutputSummary(stage: AutomationStage): StepOutputSummary {
  const artifactLabels = stage.outputs.artifacts.map(artifactLabel).filter(Boolean);
  const checkLines = stage.checks.map((check) => {
    const score = formatCheckScore(check);
    const value = `${score.value}${score.scale}`;
    return value ? `${check.name} ${value}` : check.name;
  });
  return {
    artifactCount: stage.outputs.artifacts.length,
    artifactLabels,
    checkCount: stage.checks.length,
    checkLines,
  };
}

export function outputCountLabel(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`;
}

/**
 * Footer of the open card. Only facts the header row does not show: when
 * the run started and which model ran it.
 */
export function runFooterLine(stage: AutomationStage): string {
  const clock = formatClock(stage.startedAt);
  if (stage.id === SPLIT_RUN_CLOSURE_PHASE_ID) {
    return clock ? `Closed ${clock}` : "";
  }
  return [clock ? `Started ${clock}` : "", stage.model].filter(Boolean).join(" · ");
}

/** Markdown and inline HTML down to the words, for a one-line header. */
function plainText(markdown: string): string {
  return markdown
    .replace(/<img[^>]*\balt="([^"]*)"[^>]*>/g, "$1")
    .replace(/<[^>]+>/g, "")
    .replace(/\[((?:[^[\]]|\[[^\]]*\])*)\]\([^)]*\)/g, "$1")
    .replace(/`([^`]*)`/g, "$1")
    .replace(/\*\*([^*]*)\*\*/g, "$1")
    .replace(/^·\s*/, "")
    .replace(/\s+/g, " ")
    .trim();
}

function artifactLabel(artifact: AutomationStage["outputs"]["artifacts"][number]): string {
  const data = toArtifactDataRecord(artifact.data) ?? {};
  const label = data.name ?? data.title ?? data.filename;
  return typeof label === "string" ? label : "";
}

/** "49 tool calls" for the active step, or nothing when it has not called a tool. */
export function activeStepProgress(step: AgentStep): string {
  if (step.toolCount === 0) {
    return "";
  }
  return `${step.toolCount} ${step.toolCount === 1 ? "tool call" : "tool calls"}`;
}
