import type { AgentActivity, AgentActivityItem } from "@/lib/agentActivity";
import { isHiddenAgentLiveLogText } from "@/lib/agentRunTelemetry";
import { agentToolLabelText } from "@/lib/agentToolLabels";
import type { CommandSection } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";
import { parseClaudeCodeLog } from "./parseClaudeCodeLog";
import type { SplitRunPhaseStatus, SplitRunStreamLine } from "./splitRunMocks";

const HIDDEN_KINDS = new Set(["setup"]);
const TOOL_LINE = /^-> \[([^\]]+)\]/;

export const RUNNER_COMPONENTS = new Set([
  "runnerSuperPlane",
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

export function streamNoteTextMatches(haystack: string, needle: string): boolean {
  const extra = needle.trim();
  if (!extra) {
    return true;
  }
  const live = haystack.trim();
  if (!live) {
    return false;
  }
  return noteTextCovers(live, extra) || noteTextCovers(extra, live);
}

function noteTextCovers(container: string, part: string): boolean {
  if (container === part) {
    return true;
  }
  if (container.startsWith(part) && isLineBoundary(container, part.length)) {
    return true;
  }
  let from = 0;
  while (from < container.length) {
    const embedded = container.indexOf(`\n${part}`, from);
    if (embedded === -1) {
      return false;
    }
    if (isLineBoundary(container, embedded + 1 + part.length)) {
      return true;
    }
    from = embedded + 1;
  }
  return false;
}

function isLineBoundary(text: string, index: number): boolean {
  return index === text.length || text.charAt(index) === "\n";
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
    notes.push(noteFromCommandSection(nodeId, stepId, orderKey, section));
    notes.push(...notesFromSectionEvents(nodeId, stepId, orderKey, section));
    notes.push(...notesFromAgentActivities(nodeId, stepId, orderKey, section.activities ?? []));
  }
  return notes;
}

function noteFromCommandSection(
  nodeId: string,
  stepId: string,
  orderKey: number | undefined,
  section: CommandSection,
): SplitRunStreamLine {
  return {
    id: stepId,
    nodeId,
    at: "",
    note: true,
    componentType: section.kind,
    componentName: section.preview?.trim() || section.text,
    status: streamStatus(section.status),
    detail:
      section.kind === "prompt"
        ? undefined
        : section.lines.filter((line) => line.trim() && !isHiddenAgentLiveLogText(line)).join("\n"),
    ...orderKeyProps(orderKey),
  };
}

function notesFromSectionEvents(
  nodeId: string,
  stepId: string,
  orderKey: number | undefined,
  section: CommandSection,
): SplitRunStreamLine[] {
  const notes: SplitRunStreamLine[] = [];
  for (const [eventIndex, event] of section.events.entries()) {
    if (event.kind === "note") {
      if (!event.text.trim() || isHiddenAgentLiveLogText(event.text)) {
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
        detail: tool.lines.filter((line) => line.trim() && !isHiddenAgentLiveLogText(line)).join("\n") || undefined,
        ...orderKeyProps(orderKey),
      });
    }
  }
  return notes;
}

function notesFromAgentActivities(
  nodeId: string,
  stepId: string,
  orderKey: number | undefined,
  activities: AgentActivity[],
): SplitRunStreamLine[] {
  const notes: SplitRunStreamLine[] = [];
  for (const activity of activities) {
    for (const item of activity.items) {
      const note = noteFromAgentActivityItem(nodeId, stepId, orderKey, item);
      if (note) {
        notes.push(note);
      }
    }
  }
  return notes;
}

function noteFromAgentActivityItem(
  nodeId: string,
  stepId: string,
  orderKey: number | undefined,
  item: AgentActivityItem,
): SplitRunStreamLine | undefined {
  if (item.type === "content") {
    const text = item.text.trim();
    if (!text || isHiddenAgentLiveLogText(text)) {
      return undefined;
    }
    return {
      id: item.id,
      nodeId,
      at: "",
      note: true,
      noteParentId: stepId,
      noteDepth: 1,
      componentType: "note",
      componentName: text,
      status: item.status === "running" ? "running" : "passed",
      ...orderKeyProps(orderKey),
    };
  }
  if (item.type === "tool") {
    const output = item.output.trim();
    return {
      id: item.id,
      nodeId,
      at: "",
      note: true,
      noteParentId: stepId,
      noteDepth: 1,
      componentType: item.kind,
      componentName: agentToolLabelText(item),
      status: streamStatus(item.status),
      detail: output && !isHiddenAgentLiveLogText(output) ? output : undefined,
      ...orderKeyProps(orderKey),
    };
  }
  const text = item.text.trim();
  if (!text || isHiddenAgentLiveLogText(text)) {
    return undefined;
  }
  return {
    id: item.id,
    nodeId,
    at: "",
    note: true,
    noteParentId: stepId,
    noteDepth: 1,
    componentType: "note",
    componentName: text,
    status: "passed",
    ...orderKeyProps(orderKey),
  };
}

function fallbackNotesFromPlaintext(nodeId: string, section: CommandSection): SplitRunStreamLine[] | undefined {
  if (section.kind !== "prompt" || section.events.length > 0 || (section.activities?.length ?? 0) > 0) {
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
  return notes.some((note) => streamNoteTextMatches(`${note.componentName}\n${note.detail ?? ""}`, text));
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
    if (!text || isHiddenAgentLiveLogText(text)) {
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
