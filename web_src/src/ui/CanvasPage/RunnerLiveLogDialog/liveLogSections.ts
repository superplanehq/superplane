import type { CommandSection, CommandSectionEvent, CommandTool, LogState } from "./types";

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
  return {
    ...state,
    sections: [...state.sections, emptyCommandSection(start)],
  };
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
): LogState {
  if (state.sections.length === 0) {
    return {
      ...state,
      orphanLines: [...state.orphanLines, text],
    };
  }

  const lastSectionIndex = state.sections.length - 1;
  const section = state.sections[lastSectionIndex];
  const skipLeft = replayLineSkip?.get(section.index) ?? 0;
  if (skipLeft > 0) {
    replayLineSkip?.set(section.index, skipLeft - 1);
    return state;
  }

  const nextSections = [...state.sections];
  nextSections[lastSectionIndex] = appendLineToSection(section, text);
  return {
    ...state,
    sections: nextSections,
  };
}

export function startToolOnLatestSection(state: LogState, kind: string, text: string, sourceId?: string): LogState {
  if (state.sections.length === 0) {
    return state;
  }
  const lastSectionIndex = state.sections.length - 1;
  const section = state.sections[lastSectionIndex];
  if (sourceId && findToolInSection(section, sourceId)) {
    return state;
  }
  const nextSections = [...state.sections];
  nextSections[lastSectionIndex] = startToolOnSection(section, kind, text, sourceId);
  return { ...state, sections: nextSections };
}

export function endToolOnLatestSection(
  state: LogState,
  status: "passed" | "failed",
  durationMs: number,
  sourceId?: string,
): LogState {
  if (state.sections.length === 0) {
    return state;
  }
  const lastSectionIndex = state.sections.length - 1;
  const section = state.sections[lastSectionIndex];
  if (sourceId) {
    const existing = findToolInSection(section, sourceId);
    if (existing && existing.status !== "running") {
      return state;
    }
  }
  const nextSections = [...state.sections];
  nextSections[lastSectionIndex] = endOpenTool(section, status, durationMs, sourceId);
  return { ...state, sections: nextSections };
}

function appendLineToSection(section: CommandSection, text: string): CommandSection {
  const withLine = { ...section, lines: [...section.lines, text] };
  if (!isPromptSection(section)) {
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
        tools: event.tools.map((tool) =>
          tool.id === open.id ? { ...tool, lines: [...tool.lines, text] } : tool,
        ),
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
  return section.preview?.trim() || section.text;
}
