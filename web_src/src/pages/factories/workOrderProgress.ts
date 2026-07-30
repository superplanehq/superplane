import type { FactoriesWorkOrder, WorkOrderAttribute } from "@/api-client";

/** Derived lifecycle phase — progress lives on related records; UI derives until API exposes it. */
export type WorkOrderProgressPhase =
  | "unassigned"
  | "needs_attention"
  | "no_plan"
  | "planning"
  | "implementation_in_progress"
  | "verifications_running"
  | "verifications_failed"
  | "ready_for_review"
  | "completed"
  | "rejected";

export interface WorkOrderProgress {
  phase: WorkOrderProgressPhase;
  /** Short pill label shown next to the title (Plan, Verification, …). */
  stageLabel: string;
  /** Human-readable summary for section grouping. */
  summary: string;
}

export type WorkOrderSectionId = "needs_attention" | "unassigned" | "active" | "review" | "closed";

export interface WorkOrderSectionDefinition {
  id: WorkOrderSectionId;
  title: string;
  description: string;
  phases: WorkOrderProgressPhase[];
  tone?: "attention";
}

export const WORK_ORDER_SECTIONS: WorkOrderSectionDefinition[] = [
  {
    id: "needs_attention",
    title: "Needs attention",
    description: "Work paused for a decision, approval, or clarification.",
    phases: ["needs_attention"],
    tone: "attention",
  },
  {
    id: "unassigned",
    title: "Unassigned",
    description: "New intake waiting for someone to take ownership.",
    phases: ["unassigned"],
  },
  {
    id: "active",
    title: "In progress",
    description: "Planning, implementation, or verification underway.",
    phases: ["no_plan", "planning", "implementation_in_progress", "verifications_running", "verifications_failed"],
  },
  {
    id: "review",
    title: "Ready for review",
    description: "Implementation and verifications are done — waiting on a human.",
    phases: ["ready_for_review"],
  },
  {
    id: "closed",
    title: "Closed",
    description: "Completed or rejected work orders.",
    phases: ["completed", "rejected"],
  },
];

const PHASE_META: Record<WorkOrderProgressPhase, Pick<WorkOrderProgress, "stageLabel" | "summary">> = {
  unassigned: { stageLabel: "Intake", summary: "Waiting for an assignee" },
  needs_attention: { stageLabel: "Attention", summary: "Needs a human decision" },
  no_plan: { stageLabel: "Triage", summary: "Assigned — plan not started" },
  planning: { stageLabel: "Plan", summary: "Plan in progress" },
  implementation_in_progress: { stageLabel: "Implementation", summary: "Agent implementation running" },
  verifications_running: { stageLabel: "Verification", summary: "Verifications running" },
  verifications_failed: { stageLabel: "Verification", summary: "Verifications failed" },
  ready_for_review: { stageLabel: "Review", summary: "Ready for human review" },
  completed: { stageLabel: "Done", summary: "Completed" },
  rejected: { stageLabel: "Rejected", summary: "Rejected" },
};

function attributeMap(attributes?: WorkOrderAttribute[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const attribute of attributes ?? []) {
    if (attribute.name && attribute.value !== undefined) {
      map.set(attribute.name, attribute.value);
    }
  }
  return map;
}

export function getWorkOrderAttribute(order: FactoriesWorkOrder, name: string): string | undefined {
  return attributeMap(order.attributes).get(name);
}

export function getWorkOrderDisplayKey(order: FactoriesWorkOrder): string {
  const fromAttribute = getWorkOrderAttribute(order, "key");
  if (fromAttribute) {
    return fromAttribute;
  }
  if (order.source?.key) {
    return order.source.key;
  }
  if (order.id) {
    return order.id.slice(0, 8);
  }
  return "—";
}

export function getWorkOrderBranch(order: FactoriesWorkOrder): string | undefined {
  return getWorkOrderAttribute(order, "branch") ?? getWorkOrderAttribute(order, "git.branch");
}

export function deriveWorkOrderProgress(order: FactoriesWorkOrder): WorkOrderProgress {
  if (order.state === "STATE_CLOSED") {
    const phase: WorkOrderProgressPhase = order.result === "RESULT_REJECTED" ? "rejected" : "completed";
    return { phase, ...PHASE_META[phase] };
  }

  const attrs = attributeMap(order.attributes);
  const planStatus = attrs.get("plan.status");
  const implementationStatus = attrs.get("implementation.status");
  const verificationStatus = attrs.get("verification.status");
  const needsAttention =
    attrs.get("needs_attention") === "true" ||
    Boolean(attrs.get("attention.reason")) ||
    planStatus === "needs_attention";

  if (needsAttention) {
    return { phase: "needs_attention", ...PHASE_META.needs_attention };
  }

  if (!order.assignees?.length) {
    return { phase: "unassigned", ...PHASE_META.unassigned };
  }

  if (verificationStatus === "running") {
    return { phase: "verifications_running", ...PHASE_META.verifications_running };
  }

  if (verificationStatus === "failed") {
    return { phase: "verifications_failed", ...PHASE_META.verifications_failed };
  }

  if (implementationStatus === "in_progress" || implementationStatus === "running") {
    return { phase: "implementation_in_progress", ...PHASE_META.implementation_in_progress };
  }

  if (planStatus === "draft" || planStatus === "in_progress") {
    return { phase: "planning", ...PHASE_META.planning };
  }

  if (verificationStatus === "passed" && (implementationStatus === "completed" || implementationStatus === "done")) {
    return { phase: "ready_for_review", ...PHASE_META.ready_for_review };
  }

  if (!planStatus || planStatus === "none") {
    return { phase: "no_plan", ...PHASE_META.no_plan };
  }

  return { phase: "no_plan", ...PHASE_META.no_plan };
}

export function isAssignedToUser(order: FactoriesWorkOrder, userId?: string): boolean {
  if (!userId) {
    return false;
  }
  return (order.assignees ?? []).some((assignee) => assignee.id === userId);
}

export function filterMyWorkOrders(orders: FactoriesWorkOrder[], userId?: string): FactoriesWorkOrder[] {
  if (!userId) {
    return [];
  }
  return orders.filter((order) => isAssignedToUser(order, userId) && order.state === "STATE_OPEN");
}

export function groupWorkOrdersBySection(
  orders: FactoriesWorkOrder[],
  sections: WorkOrderSectionDefinition[] = WORK_ORDER_SECTIONS,
): Array<{ section: WorkOrderSectionDefinition; orders: FactoriesWorkOrder[] }> {
  const progressByOrderId = new Map<string, WorkOrderProgressPhase>();
  for (const order of orders) {
    if (order.id) {
      progressByOrderId.set(order.id, deriveWorkOrderProgress(order).phase);
    }
  }

  return sections
    .map((section) => ({
      section,
      orders: orders.filter((order) => {
        if (!order.id) {
          return false;
        }
        const phase = progressByOrderId.get(order.id);
        return phase ? section.phases.includes(phase) : false;
      }),
    }))
    .filter((entry) => entry.orders.length > 0);
}

export function countOpenWorkOrders(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => order.state === "STATE_OPEN").length;
}

export function countNeedsAttention(orders: FactoriesWorkOrder[]): number {
  return orders.filter((order) => deriveWorkOrderProgress(order).phase === "needs_attention").length;
}

/** Open work on the factory queue — excludes closed orders. */
export const WORK_ORDER_TAB_SECTIONS = WORK_ORDER_SECTIONS.filter((section) => section.id !== "closed");

/** Assigned open work for the current user — no unassigned intake queue. */
export const MY_WORK_SECTIONS = WORK_ORDER_TAB_SECTIONS.filter((section) => section.id !== "unassigned");
