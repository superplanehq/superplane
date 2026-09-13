import { isPlanningRefineNote } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";
import { isPlanningSurveyReply } from "./planningSessionSurvey";

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
  draft?: { title?: string; description?: string; workOrderId?: string } | null;
  created?: Array<{ id?: string; key?: string; title?: string; description?: string }>;
  survey?: PlanningSessionSurveyPayload | null;
};

export type PlanningSessionMessagePayload = {
  id?: string;
  role?: string;
  text?: string;
  /** When the server persisted the message (ISO 8601). Both roles carry this. */
  createdAt?: string;
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
  return { ...next, messages: mergePlanningSessionMessages(previous.messages, next.messages) };
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
      },
    ];
  }
  return [];
}

function parsePlanningMessageCreatedAt(createdAt: string | undefined): number | undefined {
  if (!createdAt) {
    return undefined;
  }
  const parsed = Date.parse(createdAt);
  return Number.isNaN(parsed) ? undefined : parsed;
}
