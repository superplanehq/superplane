import type { CreateWithAgentCreatedOrder, CreateWithAgentTaskMessage } from "./createWithAgentTypes";

/** The fields of a session message that a task-role message uses. */
type PlanningTaskMessagePayload = {
  id?: string;
  text?: string;
  createdAt?: string;
  activityId?: string;
};

/**
 * A task-role message names a task the agent split off this draft. The
 * server snapshots the key and title in the body; the live `created` entry
 * wins when it is still there, so a rename shows through.
 */
export function planningTaskMessageFromPayload(
  message: PlanningTaskMessagePayload,
  createdByID: Map<string, CreateWithAgentCreatedOrder>,
): CreateWithAgentTaskMessage[] {
  const snapshot = parsePlanningTaskBody(message.text);
  if (!snapshot) {
    return [];
  }
  const live = createdByID.get(snapshot.workOrderId);
  const createdAtMs = message.createdAt ? Date.parse(message.createdAt) : Number.NaN;
  return [
    {
      id: message.id ?? snapshot.workOrderId,
      kind: "task",
      role: "task",
      workOrderId: snapshot.workOrderId,
      key: live?.key ?? snapshot.key,
      title: live?.title ?? snapshot.title,
      ...(live?.number === undefined ? {} : { number: live.number }),
      ...(message.activityId?.trim() ? { activityId: message.activityId.trim() } : {}),
      ...(Number.isNaN(createdAtMs) ? {} : { createdAtMs }),
    },
  ];
}

function parsePlanningTaskBody(text: string | undefined): { workOrderId: string; key: string; title: string } | null {
  if (!text?.trim()) {
    return null;
  }
  try {
    const parsed = JSON.parse(text) as { work_order_id?: unknown; key?: unknown; title?: unknown };
    if (typeof parsed.work_order_id !== "string" || !parsed.work_order_id) {
      return null;
    }
    return {
      workOrderId: parsed.work_order_id,
      key: typeof parsed.key === "string" ? parsed.key : "",
      title: typeof parsed.title === "string" ? parsed.title : "",
    };
  } catch {
    return null;
  }
}
