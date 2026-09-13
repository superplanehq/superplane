import { factoryQueryKeys } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { emptyCreateWithAgentView } from "../createWithAgentDemo";
import {
  answerPlanningSessionSurvey,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import {
  createWithAgentViewFromSession,
  mergePlanningSessionHistory,
  type PlanningSessionPayload,
} from "../planningSessionView";
import { usePlanningSessionLiveRun } from "../usePlanningSessionLiveRun";
import { workOrderPlanningSessionQueryKey } from "../useWorkOrderPlanningSurvey";

const POLL_MS = 1500;

export const ANALYSIS_PLANNING_COPY = {
  composerPlaceholder: "Add context for this plan",
  send: "Send",
  stopped: "This analysis has stopped.",
  failedSend: "The message did not send. Try again.",
  failedLoad: "The analysis session did not load. Try again.",
  writing: "The agent is writing the plan.",
};

type AnalysisPlanningSessionArgs = {
  organizationId?: string;
  factoryId?: string;
  workOrderId?: string;
  enabled: boolean;
  pollForSession?: boolean;
  canUpdate: boolean;
  analysisDelivered?: boolean;
};

export function analysisSessionPollInterval(
  pollForSession: boolean,
  session: PlanningSessionPayload | null | undefined,
) {
  if (session && session.state !== "ended") {
    return POLL_MS;
  }
  return pollForSession && !session ? POLL_MS : false;
}

function usePlanningSessionLookup(
  args: Required<Pick<AnalysisPlanningSessionArgs, "enabled" | "pollForSession">> & {
    organizationId: string;
    factoryId: string;
    workOrderId: string;
  },
) {
  const { organizationId, factoryId, workOrderId, enabled, pollForSession } = args;
  return useQuery<PlanningSessionPayload | null>({
    queryKey: workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId),
    queryFn: () => findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId),
    enabled: enabled && Boolean(organizationId && factoryId && workOrderId),
    structuralSharing: (previous, next) =>
      mergePlanningSessionHistory(
        previous as PlanningSessionPayload | null | undefined,
        next as PlanningSessionPayload | null,
      ),
    refetchInterval: (current) =>
      analysisSessionPollInterval(pollForSession, current.state.data as PlanningSessionPayload | null | undefined),
  });
}

async function refreshAnalysisWorkOrder(
  queryClient: QueryClient,
  organizationId: string,
  factoryId: string,
  workOrderId: string,
) {
  if (!organizationId || !factoryId || !workOrderId) {
    return;
  }
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, workOrderId),
    }),
    queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderChecks(organizationId, factoryId, workOrderId),
    }),
    queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, workOrderId),
    }),
  ]);
}

function useRefreshAnalysisWorkOrder(args: {
  queryClient: QueryClient;
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  session: PlanningSessionPayload | null;
  sessionUpdatedAt: number;
}) {
  const { queryClient, organizationId, factoryId, workOrderId, session, sessionUpdatedAt } = args;
  useEffect(() => {
    if (!session?.id || !sessionUpdatedAt) {
      return;
    }
    void refreshAnalysisWorkOrder(queryClient, organizationId, factoryId, workOrderId);
  }, [factoryId, organizationId, queryClient, session?.id, sessionUpdatedAt, workOrderId]);
}

function analysisView(session: PlanningSessionPayload | null, composer: string, analysisDelivered: boolean) {
  if (!session) {
    return emptyCreateWithAgentView();
  }
  return createWithAgentViewFromSession(session, {
    composer,
    right: emptyCreateWithAgentView().right,
    endConfirmOpen: false,
    analysisDelivered,
  });
}

function analysisSendState(
  session: PlanningSessionPayload | null,
  machineStatus: string,
  canUpdate: boolean,
  sendPending: boolean,
) {
  const stopped = machineStatus === "failed" || machineStatus === "passed";
  const isLive = Boolean(session?.id && session.state !== "ended" && !stopped);
  const canRestart = Boolean(session?.id && (session.state === "ended" || stopped));
  return { isLive, canSend: canUpdate && !sendPending && (isLive || canRestart) };
}

export function useAnalysisPlanningSession(args: AnalysisPlanningSessionArgs) {
  const {
    organizationId = "",
    factoryId = "",
    workOrderId = "",
    enabled,
    pollForSession = false,
    canUpdate,
    analysisDelivered = false,
  } = args;
  const queryClient = useQueryClient();
  const [composer, setComposer] = useState("");
  const [composerError, setComposerError] = useState("");
  const queryKey = workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId);
  const query = usePlanningSessionLookup({
    organizationId,
    factoryId,
    workOrderId,
    enabled,
    pollForSession,
  });
  const session = query.data ?? null;
  useRefreshAnalysisWorkOrder({
    queryClient,
    organizationId,
    factoryId,
    workOrderId,
    session,
    sessionUpdatedAt: query.dataUpdatedAt,
  });

  const onMutationSuccess = async (next: PlanningSessionPayload) => {
    queryClient.setQueryData<PlanningSessionPayload | null>(queryKey, (previous) =>
      mergePlanningSessionHistory(previous, next),
    );
    setComposer("");
    setComposerError("");
    await refreshAnalysisWorkOrder(queryClient, organizationId, factoryId, workOrderId);
  };
  const onMutationError = (error: Error) => {
    setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedSend));
  };

  const sendMessage = useMutation({
    mutationFn: (text: string) => sendPlanningSessionMessage(organizationId, factoryId, session?.id ?? "", text),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });
  const answerSurvey = useMutation({
    mutationFn: (text: string) => answerPlanningSessionSurvey(organizationId, factoryId, session?.id ?? "", text),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });

  const view = usePlanningSessionLiveRun(
    organizationId,
    analysisView(session, composer, analysisDelivered),
    analysisDelivered,
  );
  const { isLive, canSend } = analysisSendState(
    session,
    view.machineStatus,
    canUpdate,
    sendMessage.isPending || answerSurvey.isPending,
  );
  const submit = (text: string, send: (body: string) => void) => {
    const trimmed = text.trim();
    if (!trimmed || !canSend) {
      return;
    }
    setComposerError("");
    send(trimmed);
  };

  return {
    organizationId,
    session,
    sessionId: session?.id ?? "",
    queryError: query.error,
    isLoading: query.isLoading,
    view,
    composer,
    composerError,
    canSend,
    isLive,
    showChat: Boolean(session?.id),
    onComposerChange: setComposer,
    onSend: () => submit(composer, sendMessage.mutate),
    onSubmitSurvey: (text: string) => submit(text, answerSurvey.mutate),
  };
}
