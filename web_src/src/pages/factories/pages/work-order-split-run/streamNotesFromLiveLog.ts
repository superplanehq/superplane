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

/** Spreads `orderKey` only when known, so untimed lines stay comparable-key-free. */
function orderKeyProps(orderKey: number | undefined): { orderKey?: number } {
  return orderKey === undefined ? {} : { orderKey };
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
    const orderKey = section.started_at ?? undefined;
    notes.push({
      id: stepId,
      nodeId,
      at: "",
      note: true,
      componentType: section.kind,
      componentName: section.preview?.trim() || section.text,
      status: streamStatus(section.status),
      detail: section.kind === "prompt" ? undefined : section.lines.filter((line) => line.trim()).join("\n"),
      ...orderKeyProps(orderKey),
    });
    for (const [eventIndex, event] of section.events.entries()) {
      if (event.kind === "note") {
        if (!event.text.trim()) {
          continue;
        }
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
          ...orderKeyProps(orderKey),
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
          ...orderKeyProps(orderKey),
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
  const orderKey = section.started_at ?? undefined;
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
      ...orderKeyProps(orderKey),
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
      ...orderKeyProps(orderKey),
    });
  }
  return notes;
}

export function mergeLiveStreamNotes(
  live: SplitRunStreamLine[] | undefined,
  extra: SplitRunStreamLine[],
): SplitRunStreamLine[] {
  if (!live?.length) {
    return extra;
  }
  if (extra.length === 0) {
    return live;
  }
  const merged = [...live];
  let insertAt = firstOpenStepIndex(merged);
  for (const line of extra) {
    if (streamAlreadyHasText(merged, line.componentName)) {
      continue;
    }
    merged.splice(insertAt, 0, line);
    insertAt += 1;
  }
  return merged;
}

function firstOpenStepIndex(notes: SplitRunStreamLine[]): number {
  const index = notes.findIndex((note) => !note.noteParentId && note.status === "running");
  return index === -1 ? notes.length : index;
}

function streamAlreadyHasText(notes: SplitRunStreamLine[], text: string): boolean {
  const needle = text.trim();
  if (!needle) {
    return true;
  }
  const prefix = needle.slice(0, 48);
  return notes.some((note) => `${note.componentName}\n${note.detail ?? ""}`.includes(prefix));
}

export function notesForLiveStream(input: {
  nodeId: string;
  sections: CommandSection[];
  orphanLines?: string[];
  error: string | null;
  isStreaming: boolean;
  nodeStatus: SplitRunPhaseStatus;
}): SplitRunStreamLine[] | undefined {
  if (input.sections.length > 0) {
    const notes = notesFromLiveLogSections(input.nodeId, input.sections);
    if (notes.length > 0) {
      return notes;
    }
  }
  const orphanNotes = notesFromOrphanLiveLogLines(input.nodeId, input.orphanLines ?? []);
  if (orphanNotes.length > 0) {
    return orphanNotes;
  }
  if (input.error) {
    return [liveStatusNote(input.nodeId, "Something went wrong while fetching logs.", "failed")];
  }
  if (input.isStreaming || input.nodeStatus === "running") {
    return [liveStatusNote(input.nodeId, "Waiting for logs…", "running")];
  }
  return undefined;
}

function notesFromOrphanLiveLogLines(nodeId: string, lines: string[]): SplitRunStreamLine[] {
  return lines.flatMap((line, index) => {
    const text = line.trim();
    if (!text) {
      return [];
    }
    return [
      {
        id: `${nodeId}-orphan-${index}`,
        nodeId,
        at: "",
        note: true,
        componentType: "note",
        componentName: text,
        status: "passed" as const,
      },
    ];
  });
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
