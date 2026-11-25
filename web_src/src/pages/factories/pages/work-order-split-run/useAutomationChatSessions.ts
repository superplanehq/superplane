import { useQueries } from "@tanstack/react-query";
import { createContext, useState } from "react";
import { findPlanningSessionByWorkOrder } from "../planningSessionClient";
import type { PlanningSessionPayload } from "../planningSessionView";
import type { SplitRunPhase } from "./splitRunMocks";
import { workOrderPlanningSessionQueryKey } from "./useAnalysisPlanningSession";

export const AutomationChatContext = createContext<ReturnType<typeof useAutomationChatSessions> | null>(null);

/** Runs can share a conversation after an idle restart. Show each session once. */
export function useAutomationChatSessions(
  organizationId: string | undefined,
  factoryId: string | undefined,
  workOrderId: string | undefined,
  phases: SplitRunPhase[],
  planningSession?: PlanningSessionPayload | null,
) {
  const [choice, setChoice] = useState<string>();
  const runs = [...new Map(phases.filter((phase) => phase.runId).map((phase) => [phase.runId, phase])).values()].sort(
    (a, b) => (b.startedAt ?? "").localeCompare(a.startedAt ?? ""),
  );
  const sessions = useQueries({
    queries: runs.map((phase) => ({
      queryKey: workOrderPlanningSessionQueryKey(organizationId ?? "", factoryId ?? "", workOrderId ?? "", phase.runId),
      queryFn: () => findPlanningSessionByWorkOrder(organizationId!, factoryId!, workOrderId!, phase.runId),
      enabled: Boolean(organizationId && factoryId && workOrderId),
      refetchInterval: phase.status === "running" || phase.status === "pending" ? 5000 : false,
    })),
  });
  const seen = new Set<string>();
  const chats = runs.filter((_, index) => {
    const id = sessions[index].data?.id;
    if (!id || seen.has(id)) return false;
    seen.add(id);
    return true;
  });
  const planningPhase = planningSession?.id
    ? (phases.find((phase) => phase.runId === planningSession.canvasRunId) ??
      phases.find((phase) => phase.id.startsWith("backlog-analysis-")))
    : undefined;
  const conversations = planningPhase
    ? [...chats, { ...planningPhase, runId: "planning", componentName: "Planning" }]
    : chats;
  const activeChoice =
    choice === "activity"
      ? choice
      : (conversations.find((phase) => phase.runId === choice)?.runId ?? conversations[0]?.runId ?? "activity");
  const selected = chats.find((phase) => phase.runId === activeChoice);
  return { chats: conversations, selected, choice: activeChoice, setChoice };
}
