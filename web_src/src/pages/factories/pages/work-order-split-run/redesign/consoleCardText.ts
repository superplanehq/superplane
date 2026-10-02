import { durationLabelMs, formatMinutesSecondsDuration } from "@/lib/duration";

import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import { formatWorkOrderDateTime } from "../../../lib/workOrderDateTime";
import { SPLIT_RUN_CLOSURE_PHASE_ID } from "../splitRunMocks";
import type { AgentStep, AutomationStage } from "./automationsViewModel";
import { consoleCardArtifacts } from "./consolePages";

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

/**
 * Duration on the collapsed row: every run on the card, added together.
 * Spend and model sit on the open-card footer, where they can update live.
 */
export function runMetaLine(stage: AutomationStage, runs: AutomationStage[] = [stage]): string {
  const totalMs = runs.reduce((sum, run) => sum + durationLabelMs(run.duration ?? ""), 0);
  const duration =
    formatMinutesSecondsDuration(totalMs) ||
    (runs.some((run) => isSubSecondDuration(run.duration)) ? "<1s" : "") ||
    (runs.length <= 1 ? (stage.duration ?? "") : "");
  if (!duration) {
    return "";
  }
  if (stage.status === "running" || runs.some((run) => run.status === "running")) {
    return `${duration} so far`;
  }
  return duration;
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
  runCount: number;
  artifactCount: number;
  checkCount: number;
}

/** Counts for the collapsed card badges. */
export function stepOutputSummary(stage: AutomationStage, runCount = 0): StepOutputSummary {
  return {
    runCount,
    artifactCount: consoleCardArtifacts(stage).length,
    checkCount: stage.checks.length,
  };
}

export function outputCountLabel(count: number, singular: string, plural: string): string {
  return `${count} ${count === 1 ? singular : plural}`;
}

/**
 * Footer of the open card. Start stamp stays here. Model and spend are
 * rendered beside this lead so spend can stay a live control.
 */
export function runFooterLine(stage: AutomationStage): string {
  const stamp = formatWorkOrderDateTime(new Date(stage.startedAt ?? ""));
  if (stage.id === SPLIT_RUN_CLOSURE_PHASE_ID) {
    return stamp ? `Closed ${stamp}` : "";
  }
  return stamp;
}

/** Dollar amount and compact token count for the run footer. */
export function runFooterSpendLabel(cost?: string, tokens?: string): string {
  return [cost, tokens].filter(Boolean).join(" · ");
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

function isSubSecondDuration(label?: string): boolean {
  return /^<\s*1s$/i.test((label ?? "").replace(/\s+so far$/i, "").trim());
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
