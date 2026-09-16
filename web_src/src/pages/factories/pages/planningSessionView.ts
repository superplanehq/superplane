import { isPlanningRefineNote } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";
import { isPlanningSurveyReply } from "./planningSessionSurvey";
import type { AgentActivity, AgentActivityItem, AgentActivityStatus } from "./work-order-split-run/agentActivity";

export type PlanningSessionPayload = {
  id?: string;
  factoryId?: string;
  repository?: string;
  state?: string;
  canvasId?: string;
  canvasRunId?: string;
  waitState?: string;
  executionId?: string;
  selectableModelKey?: string;
  kind?: "PLANNING_SESSION_KIND_TASK_CREATION" | "PLANNING_SESSION_KIND_WORK_ORDER_ANALYSIS";
  messages?: PlanningSessionMessagePayload[];
  activities?: PlanningSessionActivityPayload[];
  draft?: { title?: string; description?: string; workOrderId?: string } | null;
  created?: Array<{ id?: string; key?: string; title?: string; description?: string }>;
  survey?: PlanningSessionSurveyPayload | null;
};

export type PlanningSessionMessagePayload = {
  id?: string;
  role?: string;
  text?: string;
  userId?: string;
  /** When the server persisted the message (ISO 8601). Both roles carry this. */
  createdAt?: string;
  activityId?: string;
};

export type PlanningSessionActivityPayload = {
  id?: string;
  schemaVersion?: number;
  provider?: string;
  status?: string;
  lastSequence?: number | string;
  startedAt?: string;
  completedAt?: string;
  truncated?: boolean;
  turn?: number;
  items?: PlanningSessionActivityItemPayload[];
};

type PlanningSessionActivityItemPayload = {
  type?: string;
  id?: string;
  kind?: string;
  text?: string;
  name?: string;
  input?: string;
  output?: string;
  outputStreams?: Array<{ stream?: string; text?: string }>;
  status?: string;
  code?: string;
  startedAt?: string;
  durationMs?: number | string;
  exitCode?: number;
  signal?: string;
  truncated?: boolean;
};

export type PlanningSessionSurveyPayload = {
  id?: string;
  questions?: Array<{ prompt?: string; options?: string[] }>;
};

/**
 * A planning session is one durable conversation across multiple agent runs.
 * Keep messages that are missing from a partial or stale response so a rewind
 * cannot replace the visible transcript with only the current run.
 */
export function mergePlanningSessionHistory(
  previous: PlanningSessionPayload | null | undefined,
  next: PlanningSessionPayload | null,
): PlanningSessionPayload | null {
  if (!previous || !next || !previous.id || previous.id !== next.id) {
    return next;
  }
  const activities = mergePlanningSessionActivities(previous.activities, next.activities);
  return {
    ...next,
    messages: mergePlanningSessionMessages(previous.messages, next.messages),
    ...(previous.activities || next.activities ? { activities } : {}),
  };
}

function mergePlanningSessionActivities(
  previous: PlanningSessionActivityPayload[] | undefined,
  next: PlanningSessionActivityPayload[] | undefined,
): PlanningSessionActivityPayload[] {
  const activities = new Map((previous ?? []).flatMap((activity) => (activity.id ? [[activity.id, activity]] : [])));
  for (const activity of next ?? []) {
    if (activity.id) {
      activities.set(activity.id, { ...activities.get(activity.id), ...activity });
    }
  }
  return [...activities.values()].sort((left, right) => timestampMs(left.startedAt) - timestampMs(right.startedAt));
}

function mergePlanningSessionMessages(
  previous: PlanningSessionMessagePayload[] | undefined,
  next: PlanningSessionMessagePayload[] | undefined,
): PlanningSessionMessagePayload[] {
  const merged = [...(previous ?? [])];
  const positions = new Map(merged.map((message, index) => [planningSessionMessageKey(message), index]));

  for (const message of next ?? []) {
    const key = planningSessionMessageKey(message);
    const position = positions.get(key);
    if (position === undefined) {
      positions.set(key, merged.length);
      merged.push(message);
      continue;
    }
    merged[position] = { ...merged[position], ...message };
  }

  return merged
    .map((message, index) => ({ message, index, createdAt: planningSessionMessageTime(message) }))
    .sort((left, right) => {
      if (left.createdAt !== undefined && right.createdAt !== undefined && left.createdAt !== right.createdAt) {
        return left.createdAt - right.createdAt;
      }
      return left.index - right.index;
    })
    .map(({ message }) => message);
}

function planningSessionMessageKey(message: PlanningSessionMessagePayload): string {
  const id = message.id?.trim();
  if (id) {
    return `id:${id}`;
  }
  return `content:${message.role ?? ""}\u0000${message.createdAt ?? ""}\u0000${message.text ?? ""}`;
}

function planningSessionMessageTime(message: PlanningSessionMessagePayload): number | undefined {
  if (!message.createdAt) {
    return undefined;
  }
  const parsed = Date.parse(message.createdAt);
  return Number.isNaN(parsed) ? undefined : parsed;
}

export function createWithAgentViewFromSession(
  session: PlanningSessionPayload,
  extras: Pick<CreateWithAgentView, "composer" | "right" | "endConfirmOpen"> & {
    analysisDelivered?: boolean;
  },
): CreateWithAgentView {
  const activities = (session.activities ?? []).flatMap(agentActivityFromPayload);
  return {
    repository: session.repository ?? "",
    machineStatus: createWithAgentMachineStatus(session, extras.analysisDelivered),
    canvasId: session.canvasId ?? "",
    canvasRunId: session.canvasRunId ?? "",
    executionId: session.executionId ?? "",
    messages: (session.messages ?? []).flatMap(planningSessionMessageFromPayload),
    survey: planningSessionSurveyFromPayload(session.survey),
    composer: extras.composer,
    created: createdOrdersFromSession(session),
    right: planningSessionRightPane(session, extras.right),
    endConfirmOpen: extras.endConfirmOpen,
    selectableModelKey: session.selectableModelKey ?? "",
    refining: Boolean(session.draft?.workOrderId?.trim()),
    ...(activities.length > 0 ? { activities } : {}),
  };
}

function createdOrdersFromSession(session: PlanningSessionPayload): CreateWithAgentCreatedOrder[] {
  return (session.created ?? [])
    .filter((order): order is { id: string; key: string; title: string; description?: string } =>
      Boolean(order.id && order.key && order.title),
    )
    .map((order) => ({
      id: order.id,
      key: order.key,
      title: order.title,
      description: order.description ?? "",
    }));
}

function planningSessionRightPane(
  session: PlanningSessionPayload,
  right: CreateWithAgentView["right"],
): CreateWithAgentView["right"] {
  const draftTitle = session.draft?.title?.trim() ?? "";
  if (draftTitle) {
    return { kind: "draft", draft: { title: draftTitle, description: session.draft?.description ?? "" } };
  }
  if (right.kind === "preview") {
    return right;
  }
  return { kind: "empty" };
}

function planningSessionSurveyFromPayload(
  survey: PlanningSessionSurveyPayload | null | undefined,
): CreateWithAgentView["survey"] {
  const questions = (survey?.questions ?? []).flatMap((question) => {
    const prompt = question.prompt?.trim() ?? "";
    const options = (question.options ?? []).map((option) => option.trim()).filter(Boolean);
    if (!prompt || !options.length) {
      return [];
    }
    return [{ prompt, options }];
  });
  if (!questions.length) {
    return undefined;
  }
  return { id: survey?.id, questions };
}

export function planningSessionHasPendingSurvey(
  session: Pick<PlanningSessionPayload, "survey"> | null | undefined,
): boolean {
  return Boolean(planningSessionSurveyFromPayload(session?.survey));
}

/** True when the session is held for the next user message. */
export function planningSessionIsWaiting(
  session: Pick<PlanningSessionPayload, "state" | "waitState"> | null | undefined,
): boolean {
  return Boolean(session && session.state !== "ended" && session.waitState === "pending");
}

/** True when the agent is still running this session. */
export function planningSessionIsWorking(
  session: Pick<PlanningSessionPayload, "state" | "waitState"> | null | undefined,
): boolean {
  return Boolean(session && session.state !== "ended" && session.waitState !== "pending");
}

/**
 * Draft cards show thinking states while the agent works, including
 * follow-up work after a score exists. The meter returns when the
 * session waits for the user.
 */
export function draftCardAgentIsWorking(
  session: Pick<PlanningSessionPayload, "state" | "waitState"> | null | undefined,
  backlogAnalyzing: boolean,
): boolean {
  return planningSessionIsWorking(session) || (backlogAnalyzing && !planningSessionIsWaiting(session));
}

export function isFailedPlanningCanvasRun(run: { result?: string } | null | undefined): boolean {
  return run?.result === "RESULT_FAILED" || run?.result === "RESULT_CANCELLED";
}

export function applyPlanningSessionLiveRun(
  view: CreateWithAgentView,
  run: { result?: string } | null | undefined,
  analysisDelivered = false,
): CreateWithAgentView {
  if (view.machineStatus === "failed" || view.machineStatus === "passed") {
    return view;
  }
  if (isFailedPlanningCanvasRun(run)) {
    return { ...view, machineStatus: analysisStopStatus(analysisDelivered) };
  }
  if (run?.result === "RESULT_PASSED" && view.machineStatus !== "waiting") {
    return { ...view, machineStatus: analysisStopStatus(analysisDelivered) };
  }
  return view;
}

function analysisStopStatus(analysisDelivered: boolean): CreateWithAgentView["machineStatus"] {
  return analysisDelivered ? "passed" : "failed";
}

function createWithAgentMachineStatus(
  session: PlanningSessionPayload,
  analysisDelivered?: boolean,
): CreateWithAgentView["machineStatus"] {
  if (session.state === "ended") {
    return analysisStopStatus(Boolean(analysisDelivered));
  }
  if (!session.executionId) {
    return "starting";
  }
  if (session.waitState === "pending") {
    return "waiting";
  }
  return "running";
}

function planningSessionMessageFromPayload(message: PlanningSessionMessagePayload): CreateWithAgentMessage[] {
  if (message.role === "user" && message.text && isPlanningRefineNote(message.text)) {
    return [];
  }
  if (message.role === "plan") {
    return planningPlanMessageFromPayload(message);
  }
  if (message.text && (message.role === "user" || message.role === "agent")) {
    const createdAtMs = parsePlanningMessageCreatedAt(message.createdAt);
    return [
      {
        id: message.id ?? message.text,
        kind: "text",
        role: message.role,
        text: message.text,
        ...(message.role === "user" && isPlanningSurveyReply(message.text) ? { origin: "survey" as const } : {}),
        ...(createdAtMs === undefined ? {} : { createdAtMs }),
        ...(message.userId?.trim() ? { userId: message.userId.trim() } : {}),
        ...(message.activityId?.trim() ? { activityId: message.activityId.trim() } : {}),
      },
    ];
  }
  return [];
}

function planningPlanMessageFromPayload(message: PlanningSessionMessagePayload): CreateWithAgentMessage[] {
  const score = planningPlanScoreFromPayload(message.text);
  if (score === undefined) {
    return [];
  }
  const createdAtMs = parsePlanningMessageCreatedAt(message.createdAt);
  return [
    {
      id: message.id ?? message.text ?? "plan",
      kind: "plan",
      role: "plan",
      score,
      ...(createdAtMs === undefined ? {} : { createdAtMs }),
    },
  ];
}

function planningPlanScoreFromPayload(text: string | undefined): number | undefined {
  if (!text?.trim()) {
    return undefined;
  }
  try {
    const parsed = JSON.parse(text) as { score?: unknown };
    if (typeof parsed.score === "number" && Number.isFinite(parsed.score)) {
      return parsed.score;
    }
  } catch {
    return undefined;
  }
  return undefined;
}

function agentActivityFromPayload(payload: PlanningSessionActivityPayload): AgentActivity[] {
  const id = payload.id?.trim();
  const status = parsedActivityStatus(payload.status);
  if (!id || !status) {
    return [];
  }
  return [
    {
      id,
      provider: payload.provider?.trim() || "agent",
      status,
      turn: payload.turn,
      sequence: numericValue(payload.lastSequence),
      startedAtMs: optionalTimestampMs(payload.startedAt),
      completedAtMs: optionalTimestampMs(payload.completedAt),
      items: (payload.items ?? []).flatMap(agentActivityItemFromPayload),
      truncated: Boolean(payload.truncated),
    },
  ];
}

function agentActivityItemFromPayload(payload: PlanningSessionActivityItemPayload): AgentActivityItem[] {
  const id = payload.id?.trim();
  if (!id) {
    return [];
  }
  if (payload.type === "content" && (payload.kind === "reasoning" || payload.kind === "assistant")) {
    return [contentActivityItem(payload, id, payload.kind)];
  }
  if (payload.type === "tool") {
    return [toolActivityItem(payload, id)];
  }
  if (payload.type === "notice") {
    return [{ type: "notice", id, code: payload.code ?? "notice", text: payload.text ?? "Agent activity notice" }];
  }
  return [];
}

function contentActivityItem(
  payload: PlanningSessionActivityItemPayload,
  id: string,
  kind: "reasoning" | "assistant",
): AgentActivityItem {
  return {
    type: "content",
    id,
    kind,
    text: payload.text ?? "",
    status: payload.status === "running" ? "running" : "passed",
    startedAtMs: optionalTimestampMs(payload.startedAt),
    durationMs: optionalNumericValue(payload.durationMs),
    truncated: Boolean(payload.truncated),
  };
}

function toolActivityItem(payload: PlanningSessionActivityItemPayload, id: string): AgentActivityItem {
  const kind = payload.kind?.trim() || "tool";
  return {
    type: "tool",
    id,
    kind,
    name: payload.name?.trim() || kind,
    input: payload.input ?? "",
    output: payload.output ?? "",
    outputStreams: (payload.outputStreams ?? []).map((output) => ({
      stream: output.stream ?? "stdout",
      text: output.text ?? "",
    })),
    status: parsedActivityStatus(payload.status) ?? "failed",
    startedAtMs: optionalTimestampMs(payload.startedAt),
    durationMs: optionalNumericValue(payload.durationMs),
    exitCode: payload.exitCode,
    signal: payload.signal,
    truncated: Boolean(payload.truncated),
  };
}

function parsedActivityStatus(status: string | undefined): AgentActivityStatus | undefined {
  if (
    status === "running" ||
    status === "passed" ||
    status === "failed" ||
    status === "cancelled" ||
    status === "timed_out" ||
    status === "interrupted"
  ) {
    return status;
  }
  return undefined;
}

function numericValue(value: number | string | undefined): number {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
}

function optionalNumericValue(value: number | string | undefined): number | undefined {
  return value === undefined ? undefined : numericValue(value);
}

function timestampMs(value: string | undefined): number {
  return optionalTimestampMs(value) ?? 0;
}

function optionalTimestampMs(value: string | undefined): number | undefined {
  if (!value) {
    return undefined;
  }
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? undefined : parsed;
}

function parsePlanningMessageCreatedAt(createdAt: string | undefined): number | undefined {
  if (!createdAt) {
    return undefined;
  }
  const parsed = Date.parse(createdAt);
  return Number.isNaN(parsed) ? undefined : parsed;
}
