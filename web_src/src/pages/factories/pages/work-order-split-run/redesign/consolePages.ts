import {
  extractArtifactFilename,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "../../../lib/workOrderArtifact";
import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import type { SplitRunPhase } from "../splitRunMocks";
import { isRunnerComponent } from "../streamNotesFromLiveLog";
import type { AutomationStage } from "./automationsViewModel";

export type ConsolePaneKind = "description" | "log" | "artifact" | "check" | "pullRequest";

/** One selectable item in the expanded card's menu. */
export interface ConsolePane {
  id: string;
  kind: ConsolePaneKind;
  label: string;
  artifact?: AutomationStage["outputs"]["artifacts"][number];
  check?: WorkOrderCheckPresentation;
}

/**
 * Everything an expanded card can show, as one menu. The agent log is an
 * item like any artifact or score, and the selected item fills the body.
 * Order: description, agent log, artifacts, scores, pull request.
 */
export function consolePanes(stage: AutomationStage, phase?: SplitRunPhase): ConsolePane[] {
  const panes: ConsolePane[] = [];
  if (stage.id === "backlog" || stage.description?.trim()) {
    panes.push({ id: "description", kind: "description", label: "Description" });
  }
  if (hasAgentLog(stage, phase)) {
    panes.push({ id: "log", kind: "log", label: "Agent log" });
  }
  for (const artifact of stage.outputs.artifacts) {
    const label = artifactPaneLabel(artifact);
    if (stage.id === "backlog" && label === "description.md") {
      continue; // The Description pane already shows the task text.
    }
    panes.push({ id: `artifact-${artifact.id ?? label}`, kind: "artifact", label, artifact });
  }
  for (const check of stage.checks) {
    panes.push({ id: `check-${check.id}`, kind: "check", label: check.name, check });
  }
  if (stage.outputs.pullRequests.length > 0) {
    panes.push({ id: "pull-request", kind: "pullRequest", label: "Pull request" });
  }
  return panes;
}

/** The agent log starts selected; a card without one starts on its first item. */
export function defaultConsolePaneId(panes: ConsolePane[]): string | undefined {
  return (panes.find((pane) => pane.kind === "log") ?? panes[0])?.id;
}

/**
 * A transcript exists, or one can still arrive: a running canvas run
 * streams live, and stored runner lines replay for finished runs.
 */
function hasAgentLog(stage: AutomationStage, phase?: SplitRunPhase): boolean {
  if (stage.agentSteps.length > 0) {
    return true;
  }
  if (stage.status === "running" && Boolean(stage.appId)) {
    return true;
  }
  return (phase?.stream ?? []).some((line) => isRunnerComponent(line.component) && Boolean(line.executionId));
}

function artifactPaneLabel(artifact: AutomationStage["outputs"]["artifacts"][number]): string {
  const data = toArtifactDataRecord(artifact.data);
  return (
    extractArtifactFilename(data) ??
    extractArtifactTitle(data) ??
    extractArtifactName(data) ??
    fallbackArtifactLabel(artifact.type)
  );
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
