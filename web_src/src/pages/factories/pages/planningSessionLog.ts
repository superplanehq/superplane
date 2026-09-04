import { isPlanningSurveyReply } from "./planningSessionSurvey";
import type { ClaudeStepGroup } from "./work-order-split-run/PhaseLogCard";
import type { SplitRunStreamLine } from "./work-order-split-run/splitRunMocks";

const PLANNING_SESSION_NOISE_PREFIXES = [
  "Planning session tools enabled",
  "permission mode:",
  "allowed tools:",
  "Claude Code started",
  "planning tools:",
  "mcp errors:",
  "Retrying API",
];

const COLLAPSED_TOOL_TYPES = new Set(["bash", "read", "edit", "write", "tool"]);

const PREAMBLE_ID = "planning-session-preamble";

export function isPlanningSessionNoise(text: string): boolean {
  const line = text.trim();
  if (!line) {
    return false;
  }
  return PLANNING_SESSION_NOISE_PREFIXES.some((prefix) => line === prefix || line.startsWith(prefix));
}

export function isPlanningSessionToolPayload(text: string): boolean {
  const line = text.trim();
  if (!line.startsWith("{")) {
    return false;
  }
  if (/"title"\s*:/.test(line) && /"description"\s*:/.test(line)) {
    return true;
  }
  if (/"message"\s*:/.test(line) || /"text"\s*:/.test(line)) {
    return true;
  }
  try {
    const value = JSON.parse(line) as unknown;
    if (!value || typeof value !== "object" || Array.isArray(value)) {
      return false;
    }
    const record = value as Record<string, unknown>;
    if (Array.isArray(record.questions) || record.status === "shown") {
      return true;
    }
    return ["message", "text", "title", "description"].some((key) => typeof record[key] === "string");
  } catch {
    return false;
  }
}

type PlanningLogState = {
  notes: SplitRunStreamLine[];
  steps: ClaudeStepGroup[];
  pendingTools: SplitRunStreamLine[];
  toolGroup: number;
};

export function groupPlanningSessionLog(notes: SplitRunStreamLine[]): ClaudeStepGroup[] {
  const state: PlanningLogState = { notes, steps: [], pendingTools: [], toolGroup: 0 };
  for (const line of notes) {
    consumePlanningLogLine(state, line);
  }
  finishPlanningLog(state);
  return state.steps;
}

function flushPlanningTools(state: PlanningLogState, parent: ClaudeStepGroup) {
  if (state.pendingTools.length === 0) {
    return;
  }
  parent.events.push({
    kind: "tools",
    id: `${parent.line.id}-tools-${state.toolGroup}`,
    tools: state.pendingTools,
  });
  state.toolGroup += 1;
  state.pendingTools = [];
}

function ensurePlanningStep(state: PlanningLogState): ClaudeStepGroup {
  const existing = state.steps.at(-1);
  if (existing) {
    return existing;
  }
  const preamble: ClaudeStepGroup = {
    line: {
      id: PREAMBLE_ID,
      nodeId: state.notes[0]?.nodeId,
      at: "",
      note: true,
      componentName: "",
      status: "passed",
    },
    events: [],
  };
  state.steps.push(preamble);
  return preamble;
}

function openHiddenPlanningPrompt(state: PlanningLogState, line: SplitRunStreamLine) {
  const current = state.steps.at(-1);
  if (current) {
    flushPlanningTools(state, current);
  } else if (state.pendingTools.length > 0) {
    flushPlanningTools(state, ensurePlanningStep(state));
  }
  state.toolGroup = 0;
  state.steps.push({
    line: { ...line, componentName: "", componentType: undefined, detail: undefined },
    events: [],
  });
  if (!isPlanningSessionUserTalk(line.componentName)) {
    return;
  }
  attachPlanningNote(state, {
    ...line,
    id: `${line.id}-talk`,
    componentType: "prompt",
    userTalk: line.userTalk ?? (isPlanningSurveyReply(line.componentName) ? "survey" : "message"),
    detail: undefined,
  });
}

function attachPlanningNote(state: PlanningLogState, line: SplitRunStreamLine) {
  if (!line.componentName.trim()) {
    return;
  }
  const parent = parentStep(state.steps, line) ?? ensurePlanningStep(state);
  flushPlanningTools(state, parent);
  parent.events.push({
    kind: "note",
    line: {
      ...line,
      componentType: line.componentType === "prompt" ? "prompt" : "note",
      detail: undefined,
    },
  });
}

function attachPlanningTool(state: PlanningLogState, line: SplitRunStreamLine) {
  if (!parentStep(state.steps, line)) {
    ensurePlanningStep(state);
  }
  state.pendingTools.push({ ...line, componentType: line.componentType || "tool" });
}

function consumePlanningLogLine(state: PlanningLogState, line: SplitRunStreamLine) {
  if (isPlanningSessionNoise(line.componentName)) {
    return;
  }
  if (isPlanningSessionToolPayload(line.componentName) || isCollapsedTool(line)) {
    attachPlanningTool(state, line);
    return;
  }
  if (!line.noteParentId && line.componentType === "prompt") {
    if (isPlanningSessionPromptStep(line)) {
      openHiddenPlanningPrompt(state, line);
      return;
    }
    attachPlanningNote(state, line);
    return;
  }
  if (parentStep(state.steps, line) || !line.noteParentId) {
    attachPlanningNote(state, line);
  }
}

function finishPlanningLog(state: PlanningLogState) {
  const last = state.steps.at(-1);
  if (last) {
    flushPlanningTools(state, last);
    return;
  }
  if (state.pendingTools.length > 0) {
    flushPlanningTools(state, ensurePlanningStep(state));
  }
}

function isCollapsedTool(line: SplitRunStreamLine): boolean {
  const type = (line.componentType ?? "").toLowerCase();
  return (
    COLLAPSED_TOOL_TYPES.has(type) ||
    type.includes("mcp__") ||
    type.endsWith("propose_draft") ||
    type.endsWith("survey") ||
    type.endsWith("say")
  );
}

function isPlanningSessionPromptStep(line: SplitRunStreamLine): boolean {
  if (line.componentType !== "prompt") {
    return false;
  }
  if (/step-\d+$/.test(line.id)) {
    return true;
  }
  return isPlanningSessionSystemPrompt(line.componentName);
}

function isPlanningSessionUserTalk(text: string): boolean {
  const name = text.trim();
  return Boolean(name) && !isPlanningSessionSystemPrompt(name);
}

function isPlanningSessionSystemPrompt(text: string): boolean {
  const name = text.trim();
  return (
    name === "Plan with the user" ||
    name === "Wait for the next user message" ||
    name.startsWith("You are in a SuperPlane planning session") ||
    name.startsWith("This is a SuperPlane planning session") ||
    name.startsWith("Greet the user") ||
    name.startsWith("The user created the draft task") ||
    name.startsWith("The user skipped that draft") ||
    name.startsWith("The user started refining")
  );
}

export function mergePlanningSessionNotes(
  live: SplitRunStreamLine[] | undefined,
  extra: SplitRunStreamLine[],
): SplitRunStreamLine[] {
  if (!live?.length) {
    return extra;
  }
  if (extra.length === 0) {
    return live;
  }

  const merged = live.map((line) => ({ ...line }));
  const { unmatchedUsers, unmatchedOthers } = partitionPlanningExtras(merged, extra);
  const hasLiveOrder = merged.some((line) => typeof line.orderKey === "number");
  const emptyWaits = merged.map((line, index) => ({ line, index })).filter(({ line }) => isEmptyWaitSlot(line));
  const slots = consumeMatchedWaitSlots(extra, unmatchedUsers, emptyWaits);
  const placed = placeUnmatchedUserExtras(merged, unmatchedUsers, emptyWaits, slots, hasLiveOrder);
  placed.insertions.sort((left, right) => right.afterIndex - left.afterIndex || right.order - left.order);
  for (const insertion of placed.insertions) {
    merged.splice(insertion.afterIndex + 1, 0, insertion.line);
  }
  return [...merged, ...unmatchedOthers, ...placed.trailing];
}

type WaitSlot = { line: SplitRunStreamLine; index: number };
type NoteInsertion = { afterIndex: number; order: number; line: SplitRunStreamLine };

function partitionPlanningExtras(
  merged: SplitRunStreamLine[],
  extra: SplitRunStreamLine[],
): { unmatchedUsers: SplitRunStreamLine[]; unmatchedOthers: SplitRunStreamLine[] } {
  const unmatchedUsers: SplitRunStreamLine[] = [];
  const unmatchedOthers: SplitRunStreamLine[] = [];
  for (const line of extra) {
    if (streamNoteHasText(merged, line.componentName)) {
      if (line.userTalk === "survey") {
        markLiveSurveyReply(merged, line.componentName);
      }
      // Live notes only carry the coarse section start time, which every note
      // in an agent turn shares. Stamp the matching live note with the agent
      // message's own created_at so a later user reply can interleave by true
      // chronology instead of landing after the whole turn.
      if (!isSessionUserExtra(line) && typeof line.orderKey === "number") {
        stampLiveNoteOrderKey(merged, line.componentName, line.orderKey);
      }
      continue;
    }
    if (isSessionUserExtra(line)) {
      unmatchedUsers.push(line);
      continue;
    }
    unmatchedOthers.push(line);
  }
  return { unmatchedUsers, unmatchedOthers };
}

function consumeMatchedWaitSlots(
  extra: SplitRunStreamLine[],
  unmatchedUsers: SplitRunStreamLine[],
  emptyWaits: WaitSlot[],
): { waitCursor: number; lastWait?: WaitSlot } {
  let waitCursor = 0;
  let lastWait: WaitSlot | undefined;
  for (const line of extra) {
    if (!isSessionUserExtra(line) || unmatchedUsers.includes(line)) {
      continue;
    }
    if (waitCursor < emptyWaits.length) {
      lastWait = emptyWaits[waitCursor];
      waitCursor += 1;
    }
  }
  return { waitCursor, lastWait };
}

type UserPlacementState = {
  merged: SplitRunStreamLine[];
  emptyWaits: WaitSlot[];
  waitCursor: number;
  lastWait?: WaitSlot;
  turnEnds: number[];
  turnEndCursor: number;
  hasLiveOrder: boolean;
  insertOrder: number;
};

function placeUnmatchedUserExtras(
  merged: SplitRunStreamLine[],
  unmatchedUsers: SplitRunStreamLine[],
  emptyWaits: WaitSlot[],
  slots: { waitCursor: number; lastWait?: WaitSlot },
  hasLiveOrder: boolean,
): { insertions: NoteInsertion[]; trailing: SplitRunStreamLine[] } {
  const insertions: NoteInsertion[] = [];
  const trailing: SplitRunStreamLine[] = [];
  const state: UserPlacementState = {
    merged,
    emptyWaits,
    waitCursor: slots.waitCursor,
    lastWait: slots.lastWait,
    turnEnds: turnEndIndexes(merged),
    turnEndCursor: 0,
    hasLiveOrder,
    insertOrder: 0,
  };
  for (const line of unmatchedUsers) {
    const insertion = nextUserPlacement(state, line);
    if (!insertion) {
      trailing.push(line);
      continue;
    }
    insertions.push(insertion);
    state.insertOrder += 1;
  }
  return { insertions, trailing };
}

function nextUserPlacement(state: UserPlacementState, line: SplitRunStreamLine): NoteInsertion | undefined {
  const turnEnd = state.turnEnds[state.turnEndCursor];
  if (turnEnd !== undefined) {
    state.turnEndCursor += 1;
    return userExtraInsertion(line, turnEnd, state.insertOrder, turnEndParent(state.merged, turnEnd));
  }
  if (state.hasLiveOrder && typeof line.orderKey === "number") {
    return orderKeyUserInsertion(state.merged, line, line.orderKey, state.insertOrder);
  }
  return waitSlotUserInsertion(state, line);
}

function orderKeyUserInsertion(
  merged: SplitRunStreamLine[],
  line: SplitRunStreamLine,
  orderKey: number,
  order: number,
): NoteInsertion {
  const afterIndex = insertionIndexByOrderKey(merged, orderKey);
  const parentId = orderKeyInsertionParent(merged, afterIndex);
  if (!parentId) {
    return { afterIndex, order, line: { ...line } };
  }
  return userExtraInsertion(line, afterIndex, order, parentId);
}

function waitSlotUserInsertion(state: UserPlacementState, line: SplitRunStreamLine): NoteInsertion | undefined {
  const slot = state.emptyWaits[state.waitCursor];
  if (slot) {
    state.lastWait = slot;
    state.waitCursor += 1;
    return userExtraInsertion(line, slot.index, state.insertOrder, slot.line.id);
  }
  if (!state.lastWait) {
    return undefined;
  }
  return userExtraInsertion(
    line,
    lastWaitGroupEndIndex(state.merged, state.lastWait),
    state.insertOrder,
    state.lastWait.line.id,
  );
}

/**
 * Finds the index to insert after so a line with `orderKey` lands in true
 * chronological order. Live notes carry mixed granularity: an agent reply
 * gets its own message time, while the "done" stat and tool notes around it
 * keep the coarser section start time. Tracking the running maximum lets those
 * trailing coarse notes inherit the turn's real time, so a later user message
 * is not pulled back in front of them. Ties resolve to the later index, and a
 * return of -1 means every timestamped line comes after `orderKey`.
 */
function insertionIndexByOrderKey(merged: SplitRunStreamLine[], orderKey: number): number {
  let index = -1;
  let runningMax = Number.NEGATIVE_INFINITY;
  for (let i = 0; i < merged.length; i += 1) {
    const key = merged[i]?.orderKey;
    if (typeof key === "number" && key > runningMax) {
      runningMax = key;
    }
    if (runningMax !== Number.NEGATIVE_INFINITY && runningMax <= orderKey) {
      index = i;
    }
  }
  return index;
}

function turnEndIndexes(merged: SplitRunStreamLine[]): number[] {
  const indexes: number[] = [];
  for (let i = 0; i < merged.length; i += 1) {
    if (isPlanningSessionTurnEnd(merged[i]?.componentName ?? "")) {
      indexes.push(i);
    }
  }
  return indexes;
}

function isPlanningSessionTurnEnd(text: string): boolean {
  const line = text.trim();
  return line.startsWith("✓ done") || line.startsWith("✗ failed");
}

function turnEndParent(merged: SplitRunStreamLine[], afterIndex: number): string {
  return orderKeyInsertionParent(merged, afterIndex) ?? merged[afterIndex]?.id ?? "";
}

/** The note/step group a chronologically placed insertion should nest under. */
function orderKeyInsertionParent(merged: SplitRunStreamLine[], afterIndex: number): string | undefined {
  const target = merged[afterIndex];
  if (!target) {
    return undefined;
  }
  return target.noteParentId ?? target.id;
}

function userExtraInsertion(
  line: SplitRunStreamLine,
  afterIndex: number,
  order: number,
  parentId: string,
): NoteInsertion {
  return {
    afterIndex,
    order,
    line: { ...line, noteParentId: parentId, noteDepth: 1 },
  };
}

function isSessionUserExtra(line: SplitRunStreamLine): boolean {
  return !line.noteParentId && line.componentType === "prompt";
}

function lastWaitGroupEndIndex(notes: SplitRunStreamLine[], wait: { line: SplitRunStreamLine; index: number }): number {
  let end = wait.index;
  for (let index = wait.index + 1; index < notes.length; index += 1) {
    if (notes[index]?.noteParentId === wait.line.id) {
      end = index;
      continue;
    }
    break;
  }
  return end;
}

function isEmptyWaitSlot(line: SplitRunStreamLine): boolean {
  return !line.noteParentId && line.componentType === "prompt" && isPlanningSessionWaitPrompt(line.componentName);
}

function isPlanningSessionWaitPrompt(text: string): boolean {
  const name = text.trim();
  return (
    name === "Wait for the next user message" ||
    name.startsWith("The user created the draft task") ||
    name.startsWith("The user skipped that draft") ||
    name.startsWith("The user started refining")
  );
}

function streamNoteHasText(notes: SplitRunStreamLine[], text: string): boolean {
  const needle = text.trim();
  if (!needle) {
    return true;
  }
  const prefix = needle.slice(0, 48);
  return notes.some((note) => `${note.componentName}\n${note.detail ?? ""}`.includes(prefix));
}

/**
 * Copies an agent message's created_at onto the live note that already renders
 * its text, so the note is ordered by when it was said rather than by the
 * section it streamed under. Only the first matching agent note is stamped.
 */
function stampLiveNoteOrderKey(notes: SplitRunStreamLine[], text: string, orderKey: number): void {
  const prefix = text.trim().slice(0, 48);
  if (!prefix) {
    return;
  }
  for (const note of notes) {
    if (note.componentType === "prompt") {
      continue;
    }
    if (`${note.componentName}\n${note.detail ?? ""}`.includes(prefix)) {
      note.orderKey = orderKey;
      return;
    }
  }
}

// A survey reply is the source of truth for what the user answered. The live
// runner log only proves that the reply reached the agent; its own rendering
// of that turn (for example a truncated preview) is not guaranteed to spell
// out the chosen answer, so the matching live note is rewritten to show the
// submitted reply text instead of whatever the live log recorded for it.
function markLiveSurveyReply(notes: SplitRunStreamLine[], submittedReply: string): void {
  const reply = submittedReply.trim();
  const prefix = reply.slice(0, 48);
  if (!prefix) {
    return;
  }
  const rootPrompts = notes.filter((note) => !note.noteParentId && note.componentType === "prompt");
  const noteText = (note: SplitRunStreamLine): string => `${note.componentName}\n${note.detail ?? ""}`;
  // When several root prompts merely share the 48-character prefix, the turn
  // whose live text still spells out the full submitted reply is the real
  // survey turn. Prefer that exact match so the answer is not attributed to an
  // earlier, unrelated prompt. Only when no prompt carries the full reply (for
  // example the live log recorded a truncated preview) do we fall back to the
  // first prefix match, which stays a single rewrite to avoid duplicating a turn.
  const target =
    rootPrompts.find((note) => noteText(note).includes(reply)) ??
    rootPrompts.find((note) => noteText(note).includes(prefix));
  if (target) {
    target.userTalk = "survey";
    target.componentName = submittedReply;
  }
}

function parentStep(steps: ClaudeStepGroup[], line: SplitRunStreamLine): ClaudeStepGroup | undefined {
  if (line.noteParentId) {
    return steps.find((step) => step.line.id === line.noteParentId);
  }
  return steps.at(-1);
}
