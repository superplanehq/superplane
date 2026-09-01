import type { WorkOrderSurveyAnswerInput } from "../lib/workOrderSurvey";

import { CREATE_WITH_AGENT_COPY } from "./createWithAgentCopy";
import type { CreateWithAgentCreatedOrder, CreateWithAgentMessage, CreateWithAgentView } from "./createWithAgentTypes";

const DEMO_SURVEY_ID = "create-with-agent-area";

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

export function createWithAgentSurveyMessage(id = DEMO_SURVEY_ID): CreateWithAgentMessage {
  return {
    id,
    kind: "survey",
    survey: {
      id,
      status: "pending",
      questions: [
        {
          id: "area",
          prompt: "Which area of the code should this task cover?",
          options: ["Payments", "Checkout", "Skip for now"],
          allowFreeText: true,
        },
      ],
    },
  };
}

export function markCreateWithAgentReady(view: CreateWithAgentView): CreateWithAgentView {
  if (view.machineStatus === "running" && view.messages.length > 0) {
    return view;
  }
  return runningCreateWithAgentView({ composer: view.composer, created: view.created, right: view.right });
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
  const nextMessages = [...view.messages, userMessage];
  const hasOpenSurvey = nextMessages.some((message) => message.kind === "survey" && !message.answered);
  if (hasOpenSurvey || view.right.kind === "draft") {
    return { ...view, composer: "", messages: nextMessages };
  }

  return {
    ...view,
    composer: "",
    messages: [...nextMessages, createWithAgentSurveyMessage(`survey-${nextMessages.length}`)],
  };
}

export function answerCreateWithAgentSurvey(
  view: CreateWithAgentView,
  surveyId: string,
  answers: WorkOrderSurveyAnswerInput[],
): CreateWithAgentView {
  const area = answers.find((answer) => answer.id === "area")?.value?.trim() || "General";
  return {
    ...view,
    messages: view.messages.map((message) =>
      message.kind === "survey" && message.survey.id === surveyId ? { ...message, answered: true } : message,
    ),
    right: {
      kind: "draft",
      draft: {
        title: area === "Skip for now" || area === "skipped" ? "Follow-up task" : `Improve ${area.toLowerCase()}`,
        description: `Draft from the agent session on ${view.repository}.`,
      },
    },
  };
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
    right: { kind: "list" },
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
    right: view.created.length > 0 ? { kind: "list" } : { kind: "empty" },
    messages: [
      ...view.messages,
      { id: `skipped-${view.messages.length}`, kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.afterSkip },
    ],
  };
}

export function workOnNewCreateWithAgent(view: CreateWithAgentView): CreateWithAgentView {
  return {
    ...view,
    right: { kind: "empty" },
    messages: [
      ...view.messages,
      { id: `next-${view.messages.length}`, kind: "text", role: "agent", text: CREATE_WITH_AGENT_COPY.nextPrompt },
    ],
  };
}

export function selectCreateWithAgentCreated(view: CreateWithAgentView, orderId: string): CreateWithAgentView {
  const order = view.created.find((item) => item.id === orderId);
  if (!order) {
    return view;
  }
  return { ...view, right: { kind: "preview", order } };
}

export function showCreateWithAgentList(view: CreateWithAgentView): CreateWithAgentView {
  return { ...view, right: view.created.length > 0 ? { kind: "list" } : { kind: "empty" } };
}

export function requestCreateWithAgentEnd(view: CreateWithAgentView): CreateWithAgentView {
  return { ...view, endConfirmOpen: true };
}

export function cancelCreateWithAgentEnd(view: CreateWithAgentView): CreateWithAgentView {
  return { ...view, endConfirmOpen: false };
}
