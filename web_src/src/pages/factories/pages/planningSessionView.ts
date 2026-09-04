import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";
import { isPlanningSurveyReply } from "./planningSessionSurvey";

export function workspacePlanningRepository(
  factory: { onboarding?: { appRepository?: string | null } } | null | undefined,
): string {
  return factory?.onboarding?.appRepository?.trim() ?? "";
}

export type PlanningSessionPayload = {
  id?: string;
  factoryId?: string;
  repository?: string;
  state?: string;
  canvasId?: string;
  canvasRunId?: string;
  waitState?: string;
  executionId?: string;
  messages?: PlanningSessionMessagePayload[];
  draft?: { title?: string; description?: string } | null;
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

export function createWithAgentViewFromSession(
  session: PlanningSessionPayload,
  extras: Pick<CreateWithAgentView, "composer" | "right" | "endConfirmOpen"> & {
    /**
     * The view's messages before this rebuild. Used to keep optimistic
     * "local-" user bubbles alive across a poll that has not yet observed
     * the persisted message, so sending a message does not flicker.
     */
    previousMessages?: CreateWithAgentMessage[];
  },
): CreateWithAgentView {
  const created: CreateWithAgentCreatedOrder[] = (session.created ?? [])
    .filter((order): order is { id: string; key: string; title: string; description?: string } =>
      Boolean(order.id && order.key && order.title),
    )
    .map((order) => ({
      id: order.id,
      key: order.key,
      title: order.title,
      description: order.description ?? "",
    }));

  const draftTitle = session.draft?.title?.trim() ?? "";
  const right = draftTitle
    ? { kind: "draft" as const, draft: { title: draftTitle, description: session.draft?.description ?? "" } }
    : extras.right.kind === "preview"
      ? extras.right
      : { kind: "empty" as const };

  const serverMessages = (session.messages ?? []).flatMap(planningSessionMessageFromPayload);
  const pendingLocal = pendingLocalUserMessages(extras.previousMessages ?? [], serverMessages);

  return {
    repository: session.repository ?? "",
    machineStatus: createWithAgentMachineStatus(session),
    canvasId: session.canvasId ?? "",
    canvasRunId: session.canvasRunId ?? "",
    executionId: session.executionId ?? "",
    messages: mergeMessagesByCreatedAt(serverMessages, pendingLocal),
    survey: planningSessionSurveyFromPayload(session.survey),
    composer: extras.composer,
    created,
    right,
    endConfirmOpen: extras.endConfirmOpen,
  };
}

function isLocalOptimisticMessage(message: CreateWithAgentMessage): boolean {
  return message.role === "user" && message.id.startsWith("local-");
}

/** Matches a user bubble to its persisted counterpart by trimmed text and survey origin. */
function userMessageKey(message: CreateWithAgentMessage): string {
  return `${message.origin === "survey" ? "survey" : "message"}:${message.text.trim()}`;
}

/**
 * Optimistic "local-" user bubbles that the server payload does not (yet)
 * contain. A poll that races the send round-trip returns a payload without
 * the just-sent message; keeping these alive across that poll prevents the
 * bubble from flickering off and back on. Once the server includes a
 * matching message, one local bubble is dropped so there is no duplicate.
 *
 * Reconciliation is by count, not by set membership, so distinct sends that
 * share the same text are not collapsed. A key drops only as many optimistic
 * bubbles as there are newly-persisted server messages for that key: the
 * current server count minus the count already persisted in the previous
 * render. This keeps a repeated message (text that also exists as an older
 * persisted message) and back-to-back identical sends each visible until the
 * server actually persists that specific send.
 */
function pendingLocalUserMessages(
  previousMessages: CreateWithAgentMessage[],
  serverMessages: CreateWithAgentMessage[],
): CreateWithAgentMessage[] {
  // Server messages that appeared since the previous render, per key. These
  // are the persisted counterparts that should retire an optimistic bubble.
  const newlyPersistedByKey = new Map<string, number>();
  for (const message of serverMessages) {
    if (message.role !== "user") {
      continue;
    }
    const key = userMessageKey(message);
    newlyPersistedByKey.set(key, (newlyPersistedByKey.get(key) ?? 0) + 1);
  }
  for (const message of previousMessages) {
    if (message.role !== "user" || isLocalOptimisticMessage(message)) {
      continue;
    }
    const key = userMessageKey(message);
    const remaining = newlyPersistedByKey.get(key);
    if (remaining !== undefined) {
      newlyPersistedByKey.set(key, remaining - 1);
    }
  }
  return previousMessages.filter((message) => {
    if (!isLocalOptimisticMessage(message)) {
      return false;
    }
    const key = userMessageKey(message);
    const remaining = newlyPersistedByKey.get(key) ?? 0;
    if (remaining > 0) {
      newlyPersistedByKey.set(key, remaining - 1);
      return false;
    }
    return true;
  });
}

/**
 * Concatenates server messages with surviving optimistic bubbles and orders
 * the result by createdAtMs. Messages without a createdAtMs (should only be
 * the seed greeting) sort first, preserving server order; optimistic bubbles
 * always carry Date.now() so they land at the tail without jumping once
 * replaced by the persisted message at the same relative position.
 */
function mergeMessagesByCreatedAt(
  serverMessages: CreateWithAgentMessage[],
  pendingLocal: CreateWithAgentMessage[],
): CreateWithAgentMessage[] {
  if (!pendingLocal.length) {
    return serverMessages;
  }
  return [...serverMessages, ...pendingLocal]
    .map((message, index) => ({ message, index }))
    .sort((a, b) => {
      const aTime = a.message.createdAtMs ?? -Infinity;
      const bTime = b.message.createdAtMs ?? -Infinity;
      if (aTime !== bTime) {
        return aTime - bTime;
      }
      return a.index - b.index;
    })
    .map(({ message }) => message);
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

export function isFailedPlanningCanvasRun(run: { result?: string } | null | undefined): boolean {
  return run?.result === "RESULT_FAILED" || run?.result === "RESULT_CANCELLED";
}

export function applyPlanningSessionLiveRun(
  view: CreateWithAgentView,
  run: { result?: string } | null | undefined,
): CreateWithAgentView {
  if (view.machineStatus === "failed" || !isFailedPlanningCanvasRun(run)) {
    return view;
  }
  return { ...view, machineStatus: "failed" };
}

function createWithAgentMachineStatus(session: PlanningSessionPayload): CreateWithAgentView["machineStatus"] {
  if (session.state === "ended") {
    return "failed";
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
