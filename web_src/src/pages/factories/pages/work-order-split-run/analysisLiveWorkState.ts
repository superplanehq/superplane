import { CREATE_WITH_AGENT_COPY } from "../createWithAgentCopy";
import type { CreateWithAgentMachineStatus } from "../createWithAgentTypes";
import { groupPlanningSessionLog, isPlanningSessionNoise, isPlanningSessionToolPayload } from "../planningSessionLog";
import type { ClaudeStepEvent, ClaudeStepGroup } from "./PhaseLogCard";
import { toolCallSummary } from "./PhaseLogCard";
import type { SplitRunStreamLine } from "./splitRunMocks";

export const ANALYSIS_THINKING_INTERVAL_MS = 2400;

export const ANALYSIS_THINKING_STATES = [
  CREATE_WITH_AGENT_COPY.machineStarting,
  "Opening the repository",
  "Reading the ticket",
] as const;

export const ANALYSIS_REPLY_THINKING_STATES = ["Reading your reply", "Updating the plan"] as const;

export type AnalysisLiveWorkKind = "idle" | "thinking" | "reasoning";

export type ReasoningItem = {
  id: string;
  text: string;
  details?: string[];
};

const REASONING_LINE_LIMIT = 2;
const REASONING_LINE_MAX_CHARS = 140;

export function analysisLiveWorkKind(args: {
  machineStatus: CreateWithAgentMachineStatus;
  items: ReasoningItem[];
}): AnalysisLiveWorkKind {
  if (args.machineStatus !== "starting" && args.machineStatus !== "running") {
    return "idle";
  }
  if (args.items.length > 0) {
    return "reasoning";
  }
  return "thinking";
}

export function hasAgentReasoning(items: ReasoningItem[]): boolean {
  return items.some((item) => !item.details?.length);
}

export function waitingForAgentReply(messages: { role: string }[]): boolean {
  return lastTalkMessage(messages)?.role !== "agent";
}

export function hidePreviousAgentStream(messages: { role: string }[]): boolean {
  return lastTalkMessage(messages)?.role === "user";
}

function lastTalkMessage(messages: { role: string }[]) {
  return [...messages].reverse().find((message) => message.role === "user" || message.role === "agent");
}

export function previousAgentStreamText(messages: { role: string; text?: string }[]): string | undefined {
  if (!hidePreviousAgentStream(messages)) {
    return undefined;
  }
  const prior = messages
    .filter((message) => message.role === "agent" && message.text)
    .map((message) => message.text)
    .join("\n");
  return prior || undefined;
}

export function liveReasoningForTurn(
  items: ReasoningItem[],
  previousAgentText?: string,
  staleIds: ReadonlySet<string> = new Set(),
): ReasoningItem[] {
  if (!previousAgentText?.trim() && staleIds.size === 0) {
    return items;
  }
  return items.filter((item) => {
    if (item.details?.length) {
      return true;
    }
    if (staleIds.has(item.id)) {
      return false;
    }
    return !previousAgentText || !isPreviousAgentNote(item.text, previousAgentText);
  });
}

export function staleReasoningIds(items: ReasoningItem[]): Set<string> {
  return new Set(items.filter((item) => !item.details?.length).map((item) => item.id));
}

function isPreviousAgentNote(liveText: string, previousText: string): boolean {
  const live = agentNoteFingerprint(liveText);
  const previous = agentNoteFingerprint(previousText);
  if (live.length < 16 || previous.length < 16) {
    return false;
  }
  return previous.includes(live) || live.includes(previous);
}

function agentNoteFingerprint(text: string): string {
  return text
    .toLowerCase()
    .replace(/[.…]+$/g, "")
    .replace(/[^a-z0-9]+/g, "");
}

export function thinkingStatusFor(
  elapsedMs: number,
  intervalMs = ANALYSIS_THINKING_INTERVAL_MS,
  states: readonly string[] = ANALYSIS_THINKING_STATES,
): string {
  const step = Math.max(0, Math.floor(elapsedMs / intervalMs));
  return states[step % states.length];
}

export function reasoningLinesFromPlanningNotes(
  notes: SplitRunStreamLine[],
  limit = REASONING_LINE_LIMIT,
): ReasoningItem[] {
  return reasoningLinesFromPlanningSteps(groupPlanningSessionLog(notes), limit);
}

export function reasoningLinesFromPlanningSteps(
  steps: ClaudeStepGroup[],
  limit = REASONING_LINE_LIMIT,
): ReasoningItem[] {
  const items: ReasoningItem[] = [];
  for (const step of steps) {
    const prompt = clampReasoningLine(step.line.componentName);
    if (prompt && isReasoningNote(step.line)) {
      items.push({ id: step.line.id, text: prompt });
    }
    for (const event of step.events) {
      const item = reasoningItemFromEvent(event);
      if (item) {
        items.push(item);
      }
    }
  }
  return items.slice(-limit);
}

function reasoningItemFromEvent(event: ClaudeStepEvent): ReasoningItem | undefined {
  if (event.kind === "tools") {
    const tools = event.tools.filter((tool) => isReasoningNote(tool));
    if (tools.length === 0) {
      return undefined;
    }
    const text = clampReasoningLine(toolCallSummary(tools));
    if (!text) {
      return undefined;
    }
    const details = tools.map((tool) => clampReasoningLine(tool.componentName)).filter(Boolean);
    return details.length > 0 ? { id: event.id, text, details } : { id: event.id, text };
  }
  if (!isReasoningNote(event.line)) {
    return undefined;
  }
  const text = clampReasoningLine(event.line.componentName);
  return text ? { id: event.line.id, text } : undefined;
}

function isReasoningNote(line: SplitRunStreamLine): boolean {
  if (line.userTalk) {
    return false;
  }
  const text = line.componentName.trim();
  if (!text || text === "Waiting for logs…") {
    return false;
  }
  if (isPlanningSessionNoise(text) || isPlanningSessionToolPayload(text)) {
    return false;
  }
  if (line.componentType === "prompt" && isPlanningSystemPrompt(text)) {
    return false;
  }
  return !isPlanningSystemPrompt(text);
}

function isPlanningSystemPrompt(text: string): boolean {
  return (
    text.startsWith("You are in a SuperPlane") ||
    text.startsWith("This is a SuperPlane") ||
    text.startsWith("Analyze this") ||
    text.startsWith("The user is adding context") ||
    text.startsWith("The user created") ||
    text.startsWith("The user skipped") ||
    text.startsWith("The user started refining") ||
    text === "Plan with the user" ||
    text === "Analyze and score" ||
    text === "Wait for the next user message"
  );
}

function clampReasoningLine(text: string): string {
  const first = text.trim().split(/\n/)[0]?.trim() ?? "";
  if (first.length <= REASONING_LINE_MAX_CHARS) {
    return first;
  }
  return `${first.slice(0, REASONING_LINE_MAX_CHARS - 1).trimEnd()}…`;
}
