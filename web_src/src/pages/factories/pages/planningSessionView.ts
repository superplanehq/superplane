import type { WorkOrderSurveyView } from "../lib/workOrderSurvey";

import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";

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
  messages?: PlanningSessionMessagePayload[];
  draft?: { title?: string; description?: string } | null;
  created?: Array<{ id?: string; key?: string; title?: string; description?: string }>;
};

export type PlanningSessionMessagePayload = {
  id?: string;
  kind?: string;
  role?: string;
  text?: string;
  answered?: boolean;
  survey?: {
    id?: string;
    status?: string;
    canvasRunId?: string;
    questions?: Array<{ id?: string; prompt?: string; options?: string[]; allowFreeText?: boolean }>;
    expiresAt?: string;
  };
};

export function createWithAgentViewFromSession(
  session: PlanningSessionPayload,
  extras: Pick<CreateWithAgentView, "composer" | "right" | "endConfirmOpen">,
): CreateWithAgentView {
  const created: CreateWithAgentCreatedOrder[] = (session.created ?? [])
    .filter((order): order is { id: string; key: string; title: string; description?: string } => Boolean(order.id && order.key && order.title))
    .map((order) => ({
      id: order.id,
      key: order.key,
      title: order.title,
      description: order.description ?? "",
    }));

  const draftTitle = session.draft?.title?.trim() ?? "";
  const right =
    extras.right.kind === "list" || extras.right.kind === "preview"
      ? extras.right
      : draftTitle
        ? { kind: "draft" as const, draft: { title: draftTitle, description: session.draft?.description ?? "" } }
        : created.length > 0
          ? { kind: "list" as const }
          : { kind: "empty" as const };

  return {
    repository: session.repository ?? "",
    machineStatus: session.canvasRunId ? "running" : "starting",
    messages: (session.messages ?? []).flatMap(planningSessionMessageFromPayload),
    composer: extras.composer,
    created,
    right,
    endConfirmOpen: extras.endConfirmOpen,
  };
}

function planningSessionMessageFromPayload(message: PlanningSessionMessagePayload): CreateWithAgentMessage[] {
  if (message.kind === "survey" && message.survey?.id) {
    const survey = planningSurveyFromPayload(message.survey);
    if (!survey) {
      return [];
    }
    return [{ id: message.id ?? survey.id, kind: "survey", survey, answered: Boolean(message.answered) }];
  }
  if (message.kind === "text" && message.text && (message.role === "user" || message.role === "agent")) {
    return [{ id: message.id ?? message.text, kind: "text", role: message.role, text: message.text }];
  }
  return [];
}

function planningSurveyFromPayload(raw: NonNullable<PlanningSessionMessagePayload["survey"]>): WorkOrderSurveyView | undefined {
  if (!raw.id || !raw.questions?.length) {
    return undefined;
  }
  const questions = raw.questions
    .map((question) => {
      const id = question.id?.trim() ?? "";
      const prompt = question.prompt?.trim() ?? "";
      if (!id || !prompt) {
        return undefined;
      }
      return {
        id,
        prompt,
        options: question.options ?? [],
        allowFreeText: Boolean(question.allowFreeText),
      };
    })
    .filter((question): question is WorkOrderSurveyView["questions"][number] => Boolean(question));
  if (questions.length === 0) {
    return undefined;
  }
  return {
    id: raw.id,
    status: raw.status ?? "pending",
    canvasRunId: raw.canvasRunId,
    questions,
    expiresAt: raw.expiresAt,
  };
}
