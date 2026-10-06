import type { CreateWithAgentView } from "../createWithAgentTypes";

/** A stored survey can outlive its answer until the next agent response arrives. */
export function hasPendingPlanningQuestions(view: CreateWithAgentView): boolean {
  const active = view.machineStatus === "starting" || view.machineStatus === "running";
  return Boolean(view.survey?.questions.length && !active && view.messages.at(-1)?.role === "agent");
}
