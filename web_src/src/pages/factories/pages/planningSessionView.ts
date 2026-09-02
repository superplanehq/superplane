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
  waitState?: string;
  executionId?: string;
  messages?: PlanningSessionMessagePayload[];
  draft?: { title?: string; description?: string } | null;
  created?: Array<{ id?: string; key?: string; title?: string; description?: string }>;
};

export type PlanningSessionMessagePayload = {
  id?: string;
  kind?: string;
  role?: string;
  text?: string;
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
  const lastCreated = created[created.length - 1];
  const right = draftTitle
    ? { kind: "draft" as const, draft: { title: draftTitle, description: session.draft?.description ?? "" } }
    : extras.right.kind === "preview"
      ? extras.right
      : lastCreated
        ? { kind: "preview" as const, order: lastCreated }
        : { kind: "empty" as const };

  return {
    repository: session.repository ?? "",
    machineStatus: createWithAgentMachineStatus(session),
    canvasId: session.canvasId ?? "",
    executionId: session.executionId ?? "",
    messages: (session.messages ?? []).flatMap(planningSessionMessageFromPayload),
    composer: extras.composer,
    created,
    right,
    endConfirmOpen: extras.endConfirmOpen,
  };
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
  if (message.kind === "text" && message.text && (message.role === "user" || message.role === "agent")) {
    return [{ id: message.id ?? message.text, kind: "text", role: message.role, text: message.text }];
  }
  return [];
}
