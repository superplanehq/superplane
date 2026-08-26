import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";

import { parseClaudeCodeLog } from "./parseClaudeCodeLog";
import type { SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";

const HIDDEN_KINDS = new Set(["setup"]);
const TOOL_LINE = /^-> \[([^\]]+)\]/;

export const RUNNER_COMPONENTS = new Set([
  "runnerClaudeCode",
  "runnerCodex",
  "runnerOpenRouter",
  "runnerBash",
  "runner",
  "runnerJS",
  "runnerPython",
]);

export function isRunnerComponent(component?: string): boolean {
  return Boolean(component && RUNNER_COMPONENTS.has(component));
}

export function notesFromLiveLogSections(nodeId: string, sections: CommandSection[]): SplitRunStreamLine[] {
  const notes: SplitRunStreamLine[] = [];
  for (const section of sections) {
    if (section.kind && HIDDEN_KINDS.has(section.kind)) {
      continue;
    }
    const fallback = fallbackNotesFromPlaintext(nodeId, section);
    if (fallback) {
      notes.push(...fallback);
      continue;
    }
    const stepId = `${nodeId}-step-${section.index}`;
    notes.push({
      id: stepId,
      nodeId,
      at: "",
      note: true,
      componentType: section.kind,
      componentName: section.preview?.trim() || section.text,
      status: streamStatus(section.status),
      detail: section.kind === "prompt" ? undefined : section.lines.filter((line) => line.trim()).join("\n"),
    });
    for (const [eventIndex, event] of section.events.entries()) {
      if (event.kind === "note") {
        notes.push({
          id: `${stepId}-note-${eventIndex}`,
          nodeId,
          at: "",
          note: true,
          noteParentId: stepId,
          noteDepth: 1,
          componentType: "note",
          componentName: event.text,
          status: "passed",
        });
        continue;
      }
      for (const tool of event.tools) {
        notes.push({
          id: tool.id,
          nodeId,
          at: "",
          note: true,
          noteParentId: stepId,
          noteDepth: 1,
          componentType: tool.kind,
          componentName: tool.text,
          status: streamStatus(tool.status),
          detail: tool.lines.filter((line) => line.trim()).join("\n") || undefined,
        });
      }
    }
  }
  return notes;
}

function fallbackNotesFromPlaintext(nodeId: string, section: CommandSection): SplitRunStreamLine[] | undefined {
  if (section.kind !== "prompt" || section.events.length > 0) {
    return undefined;
  }
  if (!section.lines.some((line) => TOOL_LINE.test(line))) {
    return undefined;
  }
  const parsed = parseClaudeCodeLog(`$ ${section.text}\n${section.lines.join("\n")}`, [
    { name: section.text, type: "prompt" },
  ]);
  const step = parsed[0];
  if (!step) {
    return undefined;
  }
  const stepId = `${nodeId}-step-${section.index}`;
  const notes: SplitRunStreamLine[] = [
    {
      id: stepId,
      nodeId,
      at: "",
      note: true,
      componentType: step.type || "prompt",
      componentName: section.preview?.trim() || step.name,
      status: step.status,
      detail: step.output,
    },
  ];
  for (const [index, command] of step.commands.entries()) {
    notes.push({
      id: `${stepId}-cmd-${index}`,
      nodeId,
      at: "",
      note: true,
      noteParentId: stepId,
      noteDepth: 1,
      componentType: command.type,
      componentName: command.name,
      status: command.status,
      detail: command.output,
    });
  }
  return notes;
}

export function notesForLiveStream(input: {
  nodeId: string;
  sections: CommandSection[];
  error: string | null;
  isStreaming: boolean;
  nodeStatus: SplitRunPhaseStatus;
}): SplitRunStreamLine[] | undefined {
  if (input.sections.length > 0) {
    const notes = notesFromLiveLogSections(input.nodeId, input.sections);
    return notes.length > 0 ? notes : undefined;
  }
  if (input.error) {
    return [liveStatusNote(input.nodeId, "Something went wrong while fetching logs.", "failed")];
  }
  if (input.isStreaming || input.nodeStatus === "running") {
    return [liveStatusNote(input.nodeId, "Waiting for logs…", "running")];
  }
  return undefined;
}

function liveStatusNote(nodeId: string, text: string, status: SplitRunPhaseStatus): SplitRunStreamLine {
  return {
    id: `${nodeId}-live-status`,
    nodeId,
    at: "",
    note: true,
    componentName: text,
    status,
  };
}

function streamStatus(status: string): SplitRunPhaseStatus {
  if (status === "failed") {
    return "failed";
  }
  if (status === "running") {
    return "running";
  }
  return "passed";
}
