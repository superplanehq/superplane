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
};

export type PlanningSessionSurveyPayload = {
  id?: string;
  questions?: Array<{ prompt?: string; options?: string[] }>;
};

export function createWithAgentViewFromSession(
  session: PlanningSessionPayload,
  extras: Pick<CreateWithAgentView, "composer" | "right" | "endConfirmOpen">,
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

  return {
    repository: session.repository ?? "",
    machineStatus: createWithAgentMachineStatus(session),
    canvasId: session.canvasId ?? "",
    executionId: session.executionId ?? "",
    messages: (session.messages ?? []).flatMap(planningSessionMessageFromPayload),
    survey: planningSessionSurveyFromPayload(session.survey),
    composer: extras.composer,
    created,
    right,
    endConfirmOpen: extras.endConfirmOpen,
  };
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

function createWithAgentMachineStatus(session: PlanningSessionPayload): CreateWithAgentView["machineStatus"] {
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
    return [
      {
        id: message.id ?? message.text,
        kind: "text",
        role: message.role,
        text: message.text,
        ...(message.role === "user" && isPlanningSurveyReply(message.text) ? { origin: "survey" as const } : {}),
      },
    ];
  }
  return [];
}
