import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";

export const CREATE_WITH_AGENT_DEMO_REPOSITORY = "acme/payments";

export function emptyCreateWithAgentView(repository = CREATE_WITH_AGENT_DEMO_REPOSITORY): CreateWithAgentView {
  return {
    repository,
    machineStatus: "starting",
    canvasId: "",
    executionId: "",
    messages: [],
    composer: "",
    created: [],
    right: { kind: "empty" },
    endConfirmOpen: false,
  };
}

export function runningCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return {
    ...emptyCreateWithAgentView(),
    machineStatus: "running",
    canvasId: "canvas-demo",
    executionId: "execution-demo",
    messages: [{ id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting }],
    ...overrides,
  };
}

export function waitingCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return runningCreateWithAgentView({ machineStatus: "waiting", ...overrides });
}

export function setCreateWithAgentComposer(view: CreateWithAgentView, composer: string): CreateWithAgentView {
  return { ...view, composer };
}

export function sendCreateWithAgentMessage(view: CreateWithAgentView): CreateWithAgentView {
  const text = view.composer.trim();
  if (!text) {
    return view;
  }

  const userMessage: CreateWithAgentMessage = {
    id: `user-${view.messages.length + 1}`,
    kind: "text",
    role: "user",
    text,
  };
  return { ...view, composer: "", survey: undefined, messages: [...view.messages, userMessage] };
}

export function updateCreateWithAgentDraft(
  view: CreateWithAgentView,
  draft: { title?: string; description?: string },
): CreateWithAgentView {
  if (view.right.kind !== "draft") {
    return view;
  }
  return {
    ...view,
    right: {
      kind: "draft",
      draft: {
        title: draft.title ?? view.right.draft.title,
        description: draft.description ?? view.right.draft.description,
      },
    },
  };
}

export function createCreateWithAgentDraft(view: CreateWithAgentView): CreateWithAgentView {
  if (view.right.kind !== "draft") {
    return view;
  }
  const order: CreateWithAgentCreatedOrder = {
    id: `wo-${view.created.length + 1}`,
    key: `NEW-${view.created.length + 1}`,
    title: view.right.draft.title.trim() || "Untitled task",
    description: view.right.draft.description,
  };
  return {
    ...view,
    created: [...view.created, order],
    right: { kind: "preview", order },
    messages: [
      ...view.messages,
      { id: `created-${order.id}`, kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.afterCreate },
    ],
  };
}

export function skipCreateWithAgentDraft(view: CreateWithAgentView): CreateWithAgentView {
  if (view.right.kind !== "draft") {
    return view;
  }
  const last = view.created[view.created.length - 1];
  return {
    ...view,
    right: last ? { kind: "preview", order: last } : { kind: "empty" },
    messages: [
      ...view.messages,
      { id: `skipped-${view.messages.length}`, kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.afterSkip },
    ],
  };
}

export function requestCreateWithAgentEnd(view: CreateWithAgentView): CreateWithAgentView {
  return { ...view, endConfirmOpen: true };
}

export function cancelCreateWithAgentEnd(view: CreateWithAgentView): CreateWithAgentView {
  return { ...view, endConfirmOpen: false };
}
