import type { WorkOrderBoardLaneId, WorkOrderDisplayStatus } from "../../lib/workOrderProgress";

export type BuiltInAgentLane = WorkOrderBoardLaneId;
export type BuiltInAgentTaskState = WorkOrderDisplayStatus;
export type BuiltInAgentMessageRole = "user" | "assistant";

export interface BuiltInAgentTask {
  id: string;
  title: string;
  state: BuiltInAgentTaskState;
  lane: BuiltInAgentLane;
  owner: string | null;
  updatedAt: string;
}

export interface BuiltInAgentMessage {
  id: string;
  role: BuiltInAgentMessageRole;
  content: string;
  createdAt: string;
}

export interface BuiltInAgentProposedPlan {
  taskIds: string[];
  reason: string;
}

export const BUILT_IN_AGENT_COPY = {
  agentName: "Planning agent",
  openPanel: "Planning agent",
  closePanel: "Close agent",
  boardLabel: "Board",
  velocityLabel: "Velocity",
  workspaceLabel: "RF",
  tasksHeading: "Tasks",
  createTask: "Create task",
  createTaskPlaceholder: "Task title",
  deleteTask: "Delete task",
  keepTask: "Keep task",
  deleteConfirmBody: "This removes the task from the board.",
  emptyTitle: "No tasks yet",
  emptyDescription: "Create a task to start the plan.",
  planTitle: "Proposed plan",
  acceptPlan: "Accept plan",
  dismissPlan: "Dismiss plan",
  composerPlaceholder: "Create a task, delete a task, or propose a plan.",
  composerHint: "Examples: create Review ledger. delete Reconcile. propose a plan.",
  send: "Send",
  errorMessage: "The agent could not complete that request.",
  retry: "Try again",
  unmatchedReply: "I can create a task, delete a task, or propose a plan. Name the action in your message.",
  createdReply: (title: string) => `Created the task "${title}".`,
  deletedReply: (title: string) => `Deleted the task "${title}".`,
  planReply: "Here is a proposed order of work. Accept the plan or dismiss it.",
  emptyPlanReply: "There are no tasks to plan. Create a task first.",
  retryReply: "Ready. Send a message to continue.",
  planReason: "Do waiting work first, then backlog, then running work.",
} as const;

const LANE_PLAN_RANK: Record<BuiltInAgentLane, number> = {
  review: 0,
  backlog: 1,
  running: 2,
  done: 3,
};

export function buildProposedPlan(tasks: BuiltInAgentTask[]): BuiltInAgentProposedPlan | null {
  if (tasks.length === 0) {
    return null;
  }

  const ranked = [...tasks].sort((left, right) => {
    const laneDelta = LANE_PLAN_RANK[left.lane] - LANE_PLAN_RANK[right.lane];
    if (laneDelta !== 0) {
      return laneDelta;
    }
    return right.title.localeCompare(left.title);
  });

  return {
    taskIds: ranked.map((task) => task.id),
    reason: BUILT_IN_AGENT_COPY.planReason,
  };
}

export function applyPlanOrder(tasks: BuiltInAgentTask[], taskIds: string[]): BuiltInAgentTask[] {
  const remaining = new Map(tasks.map((task) => [task.id, task]));
  const ordered: BuiltInAgentTask[] = [];

  for (const taskId of taskIds) {
    const task = remaining.get(taskId);
    if (!task) {
      continue;
    }
    ordered.push(task);
    remaining.delete(taskId);
  }

  for (const task of tasks) {
    if (remaining.has(task.id)) {
      ordered.push(task);
    }
  }

  return ordered;
}

export function findTaskByTitle(tasks: BuiltInAgentTask[], query: string): BuiltInAgentTask | undefined {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return undefined;
  }

  return (
    tasks.find((task) => task.title.toLowerCase() === needle) ??
    tasks.find((task) => task.title.toLowerCase().includes(needle))
  );
}

export function deleteConfirmTitle(title: string): string {
  return `Delete "${title}"?`;
}

export function newDraftTask(title: string, id: string, updatedAt: string): BuiltInAgentTask {
  return {
    id,
    title,
    state: "draft",
    lane: "backlog",
    owner: null,
    updatedAt,
  };
}
