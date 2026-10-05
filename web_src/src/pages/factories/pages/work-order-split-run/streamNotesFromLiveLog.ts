import type { AgentActivity, AgentActivityItem, AgentActivityStatus } from "@/lib/agentActivity";
import { isHiddenAgentLiveLogText } from "@/lib/agentRunTelemetry";
import { agentToolDisplayText, isCommandKind } from "@/lib/agentToolLabels";
import { formatMinutesSecondsDuration } from "@/lib/duration";
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

/** OpenCode emits a bare "Thinking" note before the thought. Do not render it as a message. */
export function isThinkingPlaceholder(text: string): boolean {
  return /^(thinking|thought)[.…]?$/i.test(text.trim());
}

/** Spreads `orderKey` only when known, so untimed lines stay comparable-key-free. */
function orderKeyProps(orderKey: number | undefined): { orderKey?: number } {
  return orderKey === undefined ? {} : { orderKey };
}

export function notesFromLiveLogSections(
  nodeId: string,
  sections: CommandSection[],
  runStatus?: SplitRunPhaseStatus,
): SplitRunStreamLine[] {
  const notes: SplitRunStreamLine[] = [];
  for (const section of sections) {
    if (section.kind && HIDDEN_KINDS.has(section.kind)) {
      continue;
    }
    const fallback = fallbackNotesFromPlaintext(nodeId, section, sections, runStatus);
    if (fallback) {
      notes.push(...fallback);
      continue;
    }
    const stepId = `${nodeId}-step-${section.index}`;
    const orderKey = section.started_at ?? undefined;
    notes.push(noteFromCommandSection(nodeId, stepId, orderKey, section, sections, runStatus));
    notes.push(...notesFromSectionEvents(nodeId, stepId, orderKey, section));
    notes.push(...notesFromAgentActivities(nodeId, stepId, orderKey, section.activities ?? []));
  }
  return notes;
}

export function agentStepStatusFromSection(input: {
  kind?: string;
  sectionStatus: string;
  runStatus?: SplitRunPhaseStatus;
  stoppedTheRun: boolean;
}): SplitRunPhaseStatus {
  const status = knownPhaseStatus(input.sectionStatus);
  if (input.kind !== "prompt" || status !== "failed") {
    return status;
  }
  if (input.runStatus === undefined) {
    return "failed";
  }
  if (input.runStatus === "failed" && input.stoppedTheRun) {
    return "failed";
  }
  return "passed";
}

function knownPhaseStatus(status: string): SplitRunPhaseStatus {
  if (
    status === "failed" ||
    status === "running" ||
    status === "passed" ||
    status === "pending" ||
    status === "waiting" ||
    status === "cancelled"
  ) {
    return status;
  }
  return streamStatus(status);
}

function sectionStoppedTheRun(section: CommandSection, sections: CommandSection[]): boolean {
  const visible = sections.filter((candidate) => !candidate.kind || !HIDDEN_KINDS.has(candidate.kind));
  return visible.at(-1) === section;
}

function noteFromCommandSection(
  nodeId: string,
  stepId: string,
  orderKey: number | undefined,
  section: CommandSection,
  sections: CommandSection[],
  runStatus?: SplitRunPhaseStatus,
): SplitRunStreamLine {
  const name = section.text.trim();
  const preview = section.preview?.trim() ?? "";
  const output = section.kind === "prompt" ? "" : visibleLogText(section.lines);
  const distinctPreview = preview !== name ? preview : "";
  const detail = [distinctPreview, output].filter(Boolean).join("\n\n");
  const bashScript = section.kind === "bash" ? preview : "";
  const sectionStatus = streamStatus(section.status);
  const prompt = section.kind === "prompt";
  return {
    id: stepId,
    nodeId,
    at: "",
    note: true,
    componentType: section.kind,
    componentName: name || preview,
    status: prompt
      ? agentStepStatusFromSection({
          kind: "prompt",
          sectionStatus,
          runStatus,
          stoppedTheRun: sectionStoppedTheRun(section, sections),
        })
      : sectionStatus,
    promptStatus: prompt ? sectionStatus : undefined,
    detail: detail || undefined,
    commandScript: bashScript || undefined,
    commandStdout: bashScript ? output || undefined : undefined,
    duration: commandSectionDuration(section.duration_ms),
    ...orderKeyProps(orderKey),
  };
}

function commandSectionDuration(durationMs: number | null): string | undefined {
  if (durationMs === null || durationMs <= 0) {
    return undefined;
  }
  return formatMinutesSecondsDuration(durationMs) || undefined;
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
        componentName: agentToolDisplayText({ kind: tool.kind, name: tool.kind, input: tool.text }),
        status: streamStatus(tool.status),
        detail: visibleLogText(tool.lines) || undefined,
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
      componentName: agentToolDisplayText(item),
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

function fallbackNotesFromPlaintext(
  nodeId: string,
  section: CommandSection,
  sections: CommandSection[],
  runStatus?: SplitRunPhaseStatus,
): SplitRunStreamLine[] | undefined {
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
      status: agentStepStatusFromSection({
        kind: "prompt",
        sectionStatus: section.status,
        runStatus,
        stoppedTheRun: sectionStoppedTheRun(section, sections),
      }),
      promptStatus: streamStatus(section.status),
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

/**
 * OpenCode and other runners often send command-section events, not
 * schema-v2 activity records. Turn those events into the same activity
 * shape the refinement chat already renders.
 */
export function activitiesFromLiveLogSections(sections: CommandSection[]): AgentActivity[] {
  const items: AgentActivityItem[] = [];
  let running = false;
  for (const section of sections) {
    if (section.kind && HIDDEN_KINDS.has(section.kind)) {
      continue;
    }
    if (section.status === "running") {
      running = true;
    }
    items.push(...itemsFromSectionEvents(section));
  }
  if (items.length === 0) {
    return [];
  }
  return [
    {
      id: "live-log",
      provider: "runner",
      status: running ? "running" : "passed",
      sequence: items.length,
      items,
      truncated: false,
    },
  ];
}

function itemsFromSectionEvents(section: CommandSection): AgentActivityItem[] {
  const items: AgentActivityItem[] = [];
  for (const [eventIndex, event] of section.events.entries()) {
    if (event.kind === "note") {
      const text = event.text.trim();
      if (!text || isHiddenAgentLiveLogText(text) || isThinkingPlaceholder(text)) {
        continue;
      }
      items.push({
        type: "content",
        id: `${section.index}-note-${eventIndex}`,
        kind: "assistant",
        text,
        status: "passed",
        truncated: false,
      });
      continue;
    }
    for (const tool of event.tools) {
      items.push({
        type: "tool",
        id: tool.id,
        kind: tool.kind,
        name: tool.kind,
        input: tool.text,
        output: isCommandKind(tool.kind) ? visibleLogText(tool.lines) : "",
        outputStreams: [],
        status: activityStatus(tool.status),
        durationMs: tool.duration_ms ?? undefined,
        truncated: false,
      });
    }
  }
  return items;
}

function visibleLogText(lines: string[]): string {
  return lines.filter((line) => line.trim() && !isHiddenAgentLiveLogText(line)).join("\n");
}

function activityStatus(status: string): AgentActivityStatus {
  if (status === "failed") {
    return "failed";
  }
  if (status === "cancelled") {
    return "cancelled";
  }
  if (status === "running") {
    return "running";
  }
  return "passed";
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
    const notes = notesFromLiveLogSections(input.nodeId, input.sections, input.nodeStatus);
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
