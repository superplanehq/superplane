import {
  extractArtifactContentType,
  extractArtifactFilename,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "../../../lib/workOrderArtifact";
import { SPLIT_RUN_CLOSURE_PHASE_ID, type SplitRunPhase } from "../splitRunMocks";
import { isWorkOrderDescriptionName } from "../splitRunPopupModel";
import { isRunnerComponent } from "../streamNotesFromLiveLog";
import type { AutomationStage } from "./automationsViewModel";

/** The expanded card's fixed pages, in tab order. */
export type ConsolePageId = "agent" | "artifacts" | "checks";

export type StageArtifact = AutomationStage["outputs"]["artifacts"][number];

/**
 * The pages a card can open. Only pages with content exist: a stage where
 * no agent ran has no Agent log page, and most stages report no checks.
 * The stage description is not a page — it always shows above the tabs.
 * The tab bar renders even for a single page, so every card reads the same.
 */
export function consolePages(stage: AutomationStage, phase?: SplitRunPhase): ConsolePageId[] {
  const pages: ConsolePageId[] = [];
  if (hasAgentLog(stage, phase)) {
    pages.push("agent");
  }
  if (artifactsPageCount(stage) > 0) {
    pages.push("artifacts");
  }
  if (stage.checks.length > 0) {
    pages.push("checks");
  }
  return pages;
}

/**
 * What the Artifacts page lists: artifacts plus the pull requests the run
 * opened. The count renders on the tab.
 */
export function artifactsPageCount(stage: AutomationStage): number {
  return stage.outputs.artifacts.length + stage.outputs.pullRequests.length;
}

/**
 * The creation card's description.md or details.md is the task text.
 * Its document row renders the editable task description, not a copy.
 */
export function isTaskDocument(stage: AutomationStage, artifact: StageArtifact): boolean {
  return stage.id === "backlog" && isWorkOrderDescriptionName(artifactLabel(artifact));
}

/** How the Artifacts page renders an artifact: markdown reads inline, media plays inline, the rest are chips. */
export function consoleArtifactKind(artifact: StageArtifact): "markdown" | "image" | "video" | "other" {
  const type = (artifact.type ?? "").replace(/^TYPE_/i, "").toLowerCase();
  if (type === "markdown") {
    return "markdown";
  }
  const contentType = extractArtifactContentType(toArtifactDataRecord(artifact.data));
  if (contentType?.startsWith("image/")) {
    return "image";
  }
  if (contentType?.startsWith("video/")) {
    return "video";
  }
  return "other";
}

export function artifactLabel(artifact: StageArtifact): string {
  const data = toArtifactDataRecord(artifact.data);
  return (
    extractArtifactFilename(data) ??
    extractArtifactTitle(data) ??
    extractArtifactName(data) ??
    fallbackArtifactLabel(artifact.type)
  );
}

/**
 * A transcript exists, or one can still arrive. Any canvas run counts:
 * a running one streams live, and a finished one replays its stored
 * runner log once the live canvas loads — even when the static stream
 * carries no runner lines. The creation and closure rows are plumbing;
 * no agent runs there.
 */
function hasAgentLog(stage: AutomationStage, phase?: SplitRunPhase): boolean {
  if (stage.agentSteps.length > 0) {
    return true;
  }
  if ((phase?.stream ?? []).some((line) => isRunnerComponent(line.component) && Boolean(line.executionId))) {
    return true;
  }
  if (stage.id === "backlog" || stage.id === SPLIT_RUN_CLOSURE_PHASE_ID) {
    return false;
  }
  return Boolean(stage.appId);
}

function fallbackArtifactLabel(type?: string): string {
  const kind = (type ?? "").replace(/^TYPE_/i, "").toLowerCase();
  if (kind === "markdown") {
    return "Note";
  }
  if (kind === "branch") {
    return "Branch";
  }
  if (kind === "link") {
    return "Link";
  }
  return "File";
}
