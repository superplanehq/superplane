import { useCallback, useReducer } from "react";

import {
  BUILT_IN_AGENT_COPY,
  SEED_TASKS,
  SEED_TRANSCRIPT,
  applyPlanOrder,
  buildProposedPlan,
  findTaskByTitle,
  newDraftTask,
  type BuiltInAgentMessage,
  type BuiltInAgentProposedPlan,
  type BuiltInAgentTask,
} from "./builtInAgentMocks";

export interface BuiltInAgentPrototypeState {
  panelOpen: boolean;
  tasks: BuiltInAgentTask[];
  transcript: BuiltInAgentMessage[];
  pendingDeleteId: string | null;
  pendingPlan: BuiltInAgentProposedPlan | null;
  hasError: boolean;
}

export interface BuiltInAgentPrototypeSeed {
  panelOpen?: boolean;
  tasks?: BuiltInAgentTask[];
  transcript?: BuiltInAgentMessage[];
  pendingDeleteId?: string | null;
  pendingPlan?: BuiltInAgentProposedPlan | null;
  hasError?: boolean;
}

type PrototypeAction =
  | { type: "openPanel" }
  | { type: "closePanel" }
  | { type: "createTask"; task: BuiltInAgentTask }
  | { type: "requestDelete"; taskId: string }
  | { type: "confirmDelete" }
  | { type: "cancelDelete" }
  | { type: "removeTask"; taskId: string }
  | { type: "appendMessages"; messages: BuiltInAgentMessage[] }
  | { type: "proposePlan"; plan: BuiltInAgentProposedPlan }
  | { type: "acceptPlan" }
  | { type: "dismissPlan" }
  | { type: "setError"; hasError: boolean };

const CREATE_PATTERN = /^(?:create(?:\s+task)?|add(?:\s+task)?)\s+(.+)$/i;
const DELETE_PATTERN = /^(?:delete(?:\s+task)?|remove(?:\s+task)?)\s+(.+)$/i;
const PLAN_PATTERN = /^(?:propose\s+(?:a\s+|an\s+ordered\s+)?plan|plan)$/i;

export function createBuiltInAgentPrototypeState(seed: BuiltInAgentPrototypeSeed = {}): BuiltInAgentPrototypeState {
  return {
    panelOpen: seed.panelOpen ?? true,
    tasks: seed.tasks ?? SEED_TASKS,
    transcript: seed.transcript ?? SEED_TRANSCRIPT,
    pendingDeleteId: seed.pendingDeleteId ?? null,
    pendingPlan: seed.pendingPlan ?? null,
    hasError: seed.hasError ?? false,
  };
}

function removeTaskFromState(state: BuiltInAgentPrototypeState, taskId: string): BuiltInAgentPrototypeState {
  const pendingPlan =
    state.pendingPlan == null
      ? null
      : {
          ...state.pendingPlan,
          taskIds: state.pendingPlan.taskIds.filter((id) => id !== taskId),
        };

  return {
    ...state,
    tasks: state.tasks.filter((task) => task.id !== taskId),
    pendingDeleteId: state.pendingDeleteId === taskId ? null : state.pendingDeleteId,
    pendingPlan: pendingPlan?.taskIds.length ? pendingPlan : null,
  };
}

function reducer(state: BuiltInAgentPrototypeState, action: PrototypeAction): BuiltInAgentPrototypeState {
  switch (action.type) {
    case "openPanel":
      return { ...state, panelOpen: true };
    case "closePanel":
      return { ...state, panelOpen: false };
    case "createTask":
      return { ...state, tasks: [action.task, ...state.tasks] };
    case "requestDelete":
      return { ...state, pendingDeleteId: action.taskId };
    case "confirmDelete":
      if (!state.pendingDeleteId) {
        return state;
      }
      return removeTaskFromState(state, state.pendingDeleteId);
    case "cancelDelete":
      return { ...state, pendingDeleteId: null };
    case "removeTask":
      return removeTaskFromState(state, action.taskId);
    case "appendMessages":
      return { ...state, transcript: [...state.transcript, ...action.messages] };
    case "proposePlan":
      return { ...state, pendingPlan: action.plan };
    case "acceptPlan":
      if (!state.pendingPlan) {
        return state;
      }
      return {
        ...state,
        tasks: applyPlanOrder(state.tasks, state.pendingPlan.taskIds),
        pendingPlan: null,
      };
    case "dismissPlan":
      return { ...state, pendingPlan: null };
    case "setError":
      return { ...state, hasError: action.hasError };
  }
}

function newMessage(role: BuiltInAgentMessage["role"], content: string, createdAt: string): BuiltInAgentMessage {
  return {
    id: `msg-${crypto.randomUUID()}`,
    role,
    content,
    createdAt,
  };
}

type ChatIntent =
  | { type: "create"; title: string }
  | { type: "delete"; task: BuiltInAgentTask }
  | { type: "plan"; plan: BuiltInAgentProposedPlan | null }
  | { type: "unknown" };

function parseChatIntent(content: string, tasks: BuiltInAgentTask[]): ChatIntent {
  const createMatch = content.match(CREATE_PATTERN);
  if (createMatch?.[1]) {
    return { type: "create", title: createMatch[1].trim() };
  }

  const deleteMatch = content.match(DELETE_PATTERN);
  if (deleteMatch?.[1]) {
    const task = findTaskByTitle(tasks, deleteMatch[1]);
    if (task) {
      return { type: "delete", task };
    }
  }

  if (PLAN_PATTERN.test(content)) {
    return { type: "plan", plan: buildProposedPlan(tasks) };
  }

  return { type: "unknown" };
}

function assistantReplyForIntent(intent: ChatIntent): string {
  if (intent.type === "create") {
    return BUILT_IN_AGENT_COPY.createdReply(intent.title);
  }
  if (intent.type === "delete") {
    return BUILT_IN_AGENT_COPY.deletedReply(intent.task.title);
  }
  if (intent.type === "plan") {
    return intent.plan ? BUILT_IN_AGENT_COPY.planReply : BUILT_IN_AGENT_COPY.emptyPlanReply;
  }
  return BUILT_IN_AGENT_COPY.unmatchedReply;
}

export function useBuiltInAgentPrototype(seed: BuiltInAgentPrototypeSeed = {}) {
  const [state, dispatch] = useReducer(reducer, seed, createBuiltInAgentPrototypeState);

  const openPanel = useCallback(() => dispatch({ type: "openPanel" }), []);
  const closePanel = useCallback(() => dispatch({ type: "closePanel" }), []);

  const createTask = useCallback((title: string) => {
    const trimmed = title.trim();
    if (!trimmed) {
      return;
    }
    dispatch({
      type: "createTask",
      task: newDraftTask(trimmed, `task-${crypto.randomUUID()}`, new Date().toISOString()),
    });
  }, []);

  const deleteTask = useCallback((taskId: string) => {
    dispatch({ type: "requestDelete", taskId });
  }, []);

  const confirmDelete = useCallback(() => dispatch({ type: "confirmDelete" }), []);
  const cancelDelete = useCallback(() => dispatch({ type: "cancelDelete" }), []);

  const sendMessage = useCallback(
    (raw: string) => {
      const content = raw.trim();
      if (!content) {
        return;
      }

      const now = new Date().toISOString();
      const intent = parseChatIntent(content, state.tasks);
      dispatch({
        type: "appendMessages",
        messages: [newMessage("user", content, now), newMessage("assistant", assistantReplyForIntent(intent), now)],
      });

      if (intent.type === "create") {
        dispatch({
          type: "createTask",
          task: newDraftTask(intent.title, `task-${crypto.randomUUID()}`, now),
        });
        return;
      }
      if (intent.type === "delete") {
        dispatch({ type: "removeTask", taskId: intent.task.id });
        return;
      }
      if (intent.type === "plan" && intent.plan) {
        dispatch({ type: "proposePlan", plan: intent.plan });
      }
    },
    [state.tasks],
  );

  const acceptPlan = useCallback(() => dispatch({ type: "acceptPlan" }), []);
  const dismissPlan = useCallback(() => dispatch({ type: "dismissPlan" }), []);

  const retryAfterError = useCallback(() => {
    dispatch({ type: "setError", hasError: false });
    dispatch({
      type: "appendMessages",
      messages: [newMessage("assistant", BUILT_IN_AGENT_COPY.retryReply, new Date().toISOString())],
    });
  }, []);

  return {
    ...state,
    openPanel,
    closePanel,
    createTask,
    deleteTask,
    confirmDelete,
    cancelDelete,
    sendMessage,
    acceptPlan,
    dismissPlan,
    retryAfterError,
  };
}
