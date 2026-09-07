import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";
import type { SelectableLLMModel } from "@/lib/selectableLLMModels";
import { isPlanningSurveyReply } from "./planningSessionSurvey";

export const CREATE_WITH_AGENT_DEMO_REPOSITORY = "acme/payments";

export const CREATE_WITH_AGENT_DEMO_MODELS: SelectableLLMModel[] = [
  {
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: "anthropic", name: "Anthropic" },
    model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
    key: "hosted::anthropic::claude-sonnet-4-6",
    label: "anthropic/claude-sonnet-4-6",
  },
  {
    source: { id: "byok", name: "Your keys" },
    provider: { id: "anthropic", name: "Anthropic" },
    model: { id: "claude-sonnet-4-6", name: "claude-sonnet-4-6" },
    key: "byok::anthropic::claude-sonnet-4-6",
    label: "anthropic/claude-sonnet-4-6",
  },
  {
    source: { id: "hosted", name: "SuperPlane" },
    provider: { id: "openai", name: "OpenAI" },
    model: { id: "gpt-5", name: "gpt-5" },
    key: "hosted::openai::gpt-5",
    label: "openai/gpt-5",
  },
];

export function emptyCreateWithAgentView(repository = CREATE_WITH_AGENT_DEMO_REPOSITORY): CreateWithAgentView {
  return {
    repository,
    machineStatus: "starting",
    canvasId: "",
    canvasRunId: "",
    executionId: "",
    messages: [],
    composer: "",
    created: [],
    right: { kind: "empty" },
    endConfirmOpen: false,
    selectableModelKey: "hosted::anthropic::claude-sonnet-4-6",
  };
}

export function runningCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return {
    ...emptyCreateWithAgentView(),
    machineStatus: "running",
    canvasId: "canvas-demo",
    canvasRunId: "run-demo",
    executionId: "execution-demo",
    messages: [{ id: "greet", kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.greeting }],
    ...overrides,
  };
}

export function waitingCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return runningCreateWithAgentView({ machineStatus: "waiting", ...overrides });
}

export function failedCreateWithAgentView(overrides: Partial<CreateWithAgentView> = {}): CreateWithAgentView {
  return runningCreateWithAgentView({ machineStatus: "failed", ...overrides });
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
    ...(isPlanningSurveyReply(text) ? { origin: "survey" as const } : {}),
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
    right: { kind: "empty" },
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
  return {
    ...view,
    right: { kind: "empty" },
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
