import { isRawAgentTurnLiveLogText } from "@/lib/agentRunTelemetry";

import type { CommandSection, CommandSectionEvent, CommandTool, LogState, PendingLiveLogRecord } from "./types";

export type CommandStart = {
  index: number;
  text: string;
  startedAtMs: number | null;
  kind?: string;
  preview?: string;
};

export function emptyCommandSection(start: CommandStart): CommandSection {
  return {
    index: start.index,
    text: start.text,
    kind: start.kind?.trim() || undefined,
    preview: start.preview?.trim() || undefined,
    lines: [],
    events: [],
    status: "running",
    duration_ms: null,
    started_at: start.startedAtMs ?? Date.now(),
    collapsed: false,
  };
}

export function startCommandSection(state: LogState, start: CommandStart): LogState {
  if (state.sections.some((section) => section.index === start.index)) {
    return state;
  }
  return attachPendingRecords(
    {
      ...state,
      sections: [...state.sections, emptyCommandSection(start)],
    },
    start.index,
  );
}

export function completeCommandSection(
  state: LogState,
  index: number,
  status: "passed" | "failed",
  durationMs: number,
): LogState {
  const existing = state.sections.find((section) => section.index === index);
  if (!existing || existing.status !== "running") {
    return state;
  }

  return {
    ...state,
    sections: state.sections.map((section) => {
      if (section.index !== index) {
        return section;
      }
      return {
        ...closeOpenTool(section, status),
        status,
        duration_ms: durationMs,
        collapsed: status === "passed",
      };
    }),
  };
}

export function appendLineToLatestSection(
  state: LogState,
  text: string,
  replayLineSkip?: Map<number, number>,
  commandIndex?: number,
): LogState {
  if (isRawAgentTurnLiveLogText(text)) {
    return state;
  }
  const destination = liveLogRecordDestination(state, commandIndex);
  if (destination.kind === "buffer") {
    return bufferPendingRecord(state, { type: "line", text, commandIndex });
  }
  if (destination.kind === "drop") {
    return state;
  }

  const section = state.sections[destination.pos];
  const skipLeft = replayLineSkip?.get(section.index) ?? 0;
  if (skipLeft > 0) {
    replayLineSkip?.set(section.index, skipLeft - 1);
    return state;
  }

  const nextSections = [...state.sections];
  nextSections[destination.pos] = appendLineToSection(section, text);
  return {
    ...state,
    sections: nextSections,
  };
}

export function startToolOnLatestSection(
  state: LogState,
  kind: string,
  text: string,
  sourceId?: string,
  commandIndex?: number,
): LogState {
  const destination = liveLogRecordDestination(state, commandIndex);
  if (destination.kind === "buffer") {
    return bufferPendingRecord(state, { type: "tool_start", kind, text, sourceId, commandIndex });
  }
  if (destination.kind === "drop") {
    return state;
  }
  const section = state.sections[destination.pos];
  if (sourceId && state.sections.some((candidate) => findToolInSection(candidate, sourceId))) {
    return state;
  }
  const nextSections = [...state.sections];
  nextSections[destination.pos] = startToolOnSection(section, kind, text, sourceId);
  return { ...state, sections: nextSections };
}

export function endToolOnLatestSection(
  state: LogState,
  status: "passed" | "failed",
  durationMs: number,
  sourceId?: string,
  commandIndex?: number,
): LogState {
  const destination = liveLogRecordDestination(state, commandIndex);
  if (destination.kind === "buffer") {
    return bufferPendingRecord(state, { type: "tool_end", status, durationMs, sourceId, commandIndex });
  }
  if (destination.kind === "drop") {
    return state;
  }
  const section = state.sections[destination.pos];
  if (sourceId) {
    const existing = findToolInSection(section, sourceId);
    if (existing && existing.status !== "running") {
      return state;
    }
  }
  const nextSections = [...state.sections];
  nextSections[destination.pos] = endOpenTool(section, status, durationMs, sourceId);
  return { ...state, sections: nextSections };
}

export function shouldSkipUnindexedLiveLogReplay(
  reconnecting: boolean,
  commandIndex: number | undefined,
  hasFinishedSection: boolean,
): boolean {
  return reconnecting && commandIndex === undefined && hasFinishedSection;
}

function commandSectionPosition(state: LogState, commandIndex?: number): number {
  if (commandIndex === undefined) {
    return state.sections.length - 1;
  }
  return state.sections.findIndex((section) => section.index === commandIndex);
}

function liveLogRecordDestination(
  state: LogState,
  commandIndex?: number,
): { kind: "buffer" } | { kind: "drop" } | { kind: "section"; pos: number } {
  const sectionPos = commandSectionPosition(state, commandIndex);
  if (sectionPos < 0) {
    return { kind: "buffer" };
  }
  if (state.sections[sectionPos].status === "running") {
    return { kind: "section", pos: sectionPos };
  }
  if (commandIndex === undefined) {
    return { kind: "buffer" };
  }
  return { kind: "drop" };
}

function pendingRecordsOf(state: LogState): PendingLiveLogRecord[] {
  return state.pendingRecords ?? [];
}

function orphanLinesFromPending(pendingRecords: PendingLiveLogRecord[]): string[] {
  return pendingRecords.filter((record) => record.type === "line").map((record) => record.text);
}

function bufferPendingRecord(state: LogState, record: PendingLiveLogRecord): LogState {
  const pendingRecords = [...pendingRecordsOf(state), record];
  return {
    ...state,
    pendingRecords,
    orphanLines: orphanLinesFromPending(pendingRecords),
  };
}

function recordBelongsToCommand(record: PendingLiveLogRecord, commandIndex: number): boolean {
  return record.commandIndex === undefined || record.commandIndex === commandIndex;
}

function attachPendingRecords(state: LogState, commandIndex: number): LogState {
  const pendingRecords = pendingRecordsOf(state);
  if (pendingRecords.length === 0) {
    return state;
  }
  const taken: PendingLiveLogRecord[] = [];
  const remaining: PendingLiveLogRecord[] = [];
  for (const record of pendingRecords) {
    if (recordBelongsToCommand(record, commandIndex)) {
      taken.push(record);
    } else {
      remaining.push(record);
    }
  }
  if (taken.length === 0) {
    return state;
  }
  let next: LogState = {
    ...state,
    pendingRecords: remaining,
    orphanLines: orphanLinesFromPending(remaining),
  };
  for (const record of taken) {
    next = applyBufferedRecord(next, record, commandIndex);
  }
  return next;
}

function applyBufferedRecord(state: LogState, record: PendingLiveLogRecord, commandIndex: number): LogState {
  const index = record.commandIndex ?? commandIndex;
  if (record.type === "line") {
    return appendLineToLatestSection(state, record.text, undefined, index);
  }
  if (record.type === "tool_start") {
    return startToolOnLatestSection(state, record.kind, record.text, record.sourceId, index);
  }
  return endToolOnLatestSection(state, record.status, record.durationMs, record.sourceId, index);
}

function appendLineToSection(section: CommandSection, text: string): CommandSection {
  const withLine = { ...section, lines: [...section.lines, text] };
  if (!isPromptSection(section) || !text.trim()) {
    return withLine;
  }

  const running = runningToolsInSection(section);
  if (running.length !== 1) {
    return {
      ...withLine,
      events: [...section.events, { kind: "note", text }],
    };
  }

  const open = running[0];
  return {
    ...withLine,
    events: section.events.map((event) => {
      if (event.kind !== "tools" || !event.tools.some((tool) => tool.id === open.id)) {
        return event;
      }
      return {
        ...event,
        tools: event.tools.map((tool) => (tool.id === open.id ? { ...tool, lines: [...tool.lines, text] } : tool)),
      };
    }),
  };
}

function startToolOnSection(section: CommandSection, kind: string, text: string, sourceId?: string): CommandSection {
  const tool: CommandTool = {
    id: sourceId?.trim() || `${section.index}-tool-${toolCount(section)}`,
    sourceId: sourceId?.trim() || undefined,
    kind: kind.trim() || "tool",
    text: text.trim() || kind.trim() || "tool",
    lines: [],
    status: "running",
    duration_ms: null,
  };
  const last = section.events.at(-1);
  if (last?.kind === "tools") {
    return {
      ...section,
      events: [...section.events.slice(0, -1), { ...last, tools: [...last.tools, tool] }],
    };
  }
  const group: CommandSectionEvent = {
    kind: "tools",
    id: `${section.index}-tools-${toolGroupCount(section)}`,
    tools: [tool],
  };
  return {
    ...section,
    events: [...section.events, group],
  };
}

function endOpenTool(
  section: CommandSection,
  status: "passed" | "failed",
  durationMs: number,
  sourceId?: string,
): CommandSection {
  const open = openToolInSection(section, sourceId);
  if (!open) {
    return section;
  }
  return {
    ...section,
    events: section.events.map((event) => {
      if (event.kind !== "tools" || event.id !== open.groupId) {
        return event;
      }
      return {
        ...event,
        tools: event.tools.map((tool) =>
          tool.id === open.toolId ? { ...tool, status, duration_ms: durationMs } : tool,
        ),
      };
    }),
  };
}

export function closeOpenTool(section: CommandSection, status: "passed" | "failed"): CommandSection {
  let next = section;
  let open = openToolInSection(next);
  while (open) {
    next = endOpenTool(next, status, 0);
    open = openToolInSection(next);
  }
  return next;
}

function openToolInSection(
  section: CommandSection,
  sourceId?: string,
): { groupId: string; toolId: string } | undefined {
  const wanted = sourceId?.trim();
  for (let index = section.events.length - 1; index >= 0; index -= 1) {
    const event = section.events[index];
    if (event.kind !== "tools") {
      continue;
    }
    const running = [...event.tools].reverse().find((tool) => {
      if (tool.status !== "running") {
        return false;
      }
      if (!wanted) {
        return true;
      }
      return tool.sourceId === wanted || tool.id === wanted;
    });
    if (running) {
      return { groupId: event.id, toolId: running.id };
    }
  }
  return undefined;
}

function findToolInSection(section: CommandSection, sourceId: string): CommandTool | undefined {
  const wanted = sourceId.trim();
  if (!wanted) {
    return undefined;
  }
  for (const event of section.events) {
    if (event.kind !== "tools") {
      continue;
    }
    const match = event.tools.find((tool) => tool.sourceId === wanted || tool.id === wanted);
    if (match) {
      return match;
    }
  }
  return undefined;
}

function isPromptSection(section: CommandSection): boolean {
  return section.kind === "prompt";
}

function runningToolsInSection(section: CommandSection): CommandTool[] {
  return section.events.flatMap((event) =>
    event.kind === "tools" ? event.tools.filter((tool) => tool.status === "running") : [],
  );
}

function toolCount(section: CommandSection): number {
  return section.events.reduce((count, event) => count + (event.kind === "tools" ? event.tools.length : 0), 0);
}

function toolGroupCount(section: CommandSection): number {
  return section.events.filter((event) => event.kind === "tools").length;
}

export function sectionTitle(section: CommandSection): string {
  return firstNonEmptyLine(section.preview) || section.text;
}

function firstNonEmptyLine(text?: string): string | undefined {
  if (!text) {
    return undefined;
  }
  for (const line of text.split("\n")) {
    const trimmed = line.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return undefined;
}
