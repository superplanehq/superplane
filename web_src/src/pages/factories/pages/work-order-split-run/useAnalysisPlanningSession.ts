import { factoryQueryKeys } from "@/hooks/useFactoryData";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { getApiErrorMessage } from "@/lib/errors";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { emptyCreateWithAgentView } from "../createWithAgentDemo";
import {
  answerPlanningSessionSurvey,
  endPlanningSession,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import {
  createWithAgentViewFromSession,
  mergePlanningSessionHistory,
  type PlanningSessionPayload,
} from "../planningSessionView";
import { usePlanningSessionLiveRun } from "../usePlanningSessionLiveRun";

const POLL_MS = 1500;

export function workOrderPlanningSessionQueryKey(organizationId: string, factoryId: string, workOrderId: string) {
  return ["planning-session-by-work-order", organizationId, factoryId, workOrderId] as const;
}

export const ANALYSIS_PLANNING_COPY = {
  composerPlaceholder: "Tell the agent more about this task",
  send: "Send",
  sendShortcut: "Enter",
  stopped: "This analysis has stopped.",
  failedSend: "The message did not send. Try again.",
  failedStop: "The analysis did not stop. Try again.",
  failedLoad: "The analysis session did not load. Try again.",
};

type AnalysisPlanningSessionArgs = {
  organizationId?: string;
  factoryId?: string;
  workOrderId?: string;
  enabled: boolean;
  pollForSession?: boolean;
  canUpdate: boolean;
  analysisDelivered?: boolean;
  isUploading?: boolean;
  uploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
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
      queryKey: factoryQueryKeys.workOrders(organizationId, factoryId),
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
}) {
  const { queryClient, organizationId, factoryId, workOrderId, session } = args;
  const refreshKey = analysisWorkOrderRefreshKey(session);
  useEffect(() => {
    if (!refreshKey) {
      return;
    }
    void refreshAnalysisWorkOrder(queryClient, organizationId, factoryId, workOrderId);
  }, [factoryId, organizationId, queryClient, refreshKey, workOrderId]);
}

export function analysisWorkOrderRefreshKey(session: PlanningSessionPayload | null | undefined): string {
  if (!session?.id) {
    return "";
  }
  return JSON.stringify({
    id: session.id,
    state: session.state,
    waitState: session.waitState,
    messages: session.messages?.map(({ id, role, text, createdAt }) => ({ id, role, text, createdAt })),
    draft: session.draft,
    created: session.created,
  });
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

function analysisLiveFlags(session: PlanningSessionPayload | null, machineStatus: string) {
  const stopped = machineStatus === "failed" || machineStatus === "passed";
  return {
    isLive: Boolean(session?.id && session.state !== "ended" && !stopped),
    canRestart: Boolean(session?.id && (session.state === "ended" || stopped)),
  };
}

function analysisSendState(args: {
  session: PlanningSessionPayload | null;
  machineStatus: string;
  canUpdate: boolean;
  sendPending: boolean;
  isUploading?: boolean;
  stopPending?: boolean;
}) {
  const { session, machineStatus, canUpdate, sendPending, isUploading = false, stopPending = false } = args;
  const { isLive, canRestart } = analysisLiveFlags(session, machineStatus);
  const idle = canUpdate && !sendPending && !stopPending;
  return {
    isLive,
    canSend: idle && !isUploading && (isLive || canRestart),
    canStop: idle && isLive,
  };
}

function usePlanningSessionMutations(args: {
  organizationId: string;
  factoryId: string;
  sessionId: string;
  queryKey: ReturnType<typeof workOrderPlanningSessionQueryKey>;
  queryClient: QueryClient;
  setComposer: (value: string) => void;
  setComposerError: (value: string) => void;
}) {
  const { organizationId, factoryId, sessionId, queryKey, queryClient, setComposer, setComposerError } = args;
  const cacheSession = (next: PlanningSessionPayload) => {
    queryClient.setQueryData<PlanningSessionPayload | null>(queryKey, (previous) =>
      mergePlanningSessionHistory(previous, next),
    );
  };
  const onMutationSuccess = (next: PlanningSessionPayload) => {
    cacheSession(next);
    setComposer("");
    setComposerError("");
  };
  const onMutationError = (error: Error) => {
    setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedSend));
  };
  const sendMessage = useMutation({
    mutationFn: (text: string) => sendPlanningSessionMessage(organizationId, factoryId, sessionId, text),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });
  const answerSurvey = useMutation({
    mutationFn: (text: string) => answerPlanningSessionSurvey(organizationId, factoryId, sessionId, text),
    onSuccess: onMutationSuccess,
    onError: onMutationError,
  });
  const stopSession = useMutation({
    mutationFn: () => endPlanningSession(organizationId, factoryId, sessionId),
    onSuccess: (next) => {
      cacheSession(next);
      setComposerError("");
    },
    onError: (error: Error) => {
      setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedStop));
    },
  });
  return { sendMessage, answerSurvey, stopSession };
}

function analysisPlanningArgs(args: AnalysisPlanningSessionArgs) {
  return {
    organizationId: args.organizationId ?? "",
    factoryId: args.factoryId ?? "",
    workOrderId: args.workOrderId ?? "",
    enabled: args.enabled,
    pollForSession: args.pollForSession ?? false,
    canUpdate: args.canUpdate,
    analysisDelivered: args.analysisDelivered ?? false,
    isUploading: args.isUploading ?? false,
    uploadFiles: args.uploadFiles,
  };
}

export function useAnalysisPlanningSession(args: AnalysisPlanningSessionArgs) {
  const {
    organizationId,
    factoryId,
    workOrderId,
    enabled,
    pollForSession,
    canUpdate,
    analysisDelivered,
    isUploading,
    uploadFiles,
  } = analysisPlanningArgs(args);
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
  });

  const { sendMessage, answerSurvey, stopSession } = usePlanningSessionMutations({
    organizationId,
    factoryId,
    sessionId: session?.id ?? "",
    queryKey,
    queryClient,
    setComposer,
    setComposerError,
  });

  const view = usePlanningSessionLiveRun(
    organizationId,
    analysisView(session, composer, analysisDelivered),
    analysisDelivered,
  );
  const sendPending = sendMessage.isPending || answerSurvey.isPending;
  const { isLive, canSend, canStop } = analysisSendState({
    session,
    machineStatus: view.machineStatus,
    canUpdate,
    sendPending,
    isUploading,
    stopPending: stopSession.isPending,
  });
  const actions = useAnalysisComposerActions({
    composer,
    canSend,
    canStop,
    canUpdate,
    sessionId: session?.id ?? "",
    sendMessage,
    answerSurvey,
    stopSession,
    setComposerError,
    queryClient,
    organizationId,
    factoryId,
    workOrderId,
    uploadFiles,
  });

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
    canStop,
    isUploading,
    isLive,
    showChat: Boolean(session?.id),
    onComposerChange: setComposer,
    stopping: stopSession.isPending,
    ...actions,
  };
}

function useAnalysisComposerActions(args: {
  composer: string;
  canSend: boolean;
  canStop: boolean;
  canUpdate: boolean;
  sessionId: string;
  sendMessage: ReturnType<typeof usePlanningSessionMutations>["sendMessage"];
  answerSurvey: ReturnType<typeof usePlanningSessionMutations>["answerSurvey"];
  stopSession: ReturnType<typeof usePlanningSessionMutations>["stopSession"];
  setComposerError: (value: string) => void;
  queryClient: QueryClient;
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  uploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
}) {
  const {
    composer,
    canSend,
    canStop,
    canUpdate,
    sessionId,
    sendMessage,
    answerSurvey,
    stopSession,
    setComposerError,
    queryClient,
    organizationId,
    factoryId,
    workOrderId,
    uploadFiles,
  } = args;
  const submit = (text: string, send: (body: string) => void) => {
    const trimmed = text.trim();
    if (!trimmed || !canSend) {
      return false;
    }
    setComposerError("");
    send(trimmed);
    return true;
  };
  const onSend = async (text?: string) => {
    const trimmed = (text ?? composer).trim();
    if (!trimmed || !canSend) {
      return false;
    }
    setComposerError("");
    try {
      await sendMessage.mutateAsync(trimmed);
      return true;
    } catch {
      return false;
    }
  };
  const onStop = async () => {
    if (!canStop || !sessionId) {
      return;
    }
    setComposerError("");
    try {
      await stopSession.mutateAsync();
    } catch {
      return;
    }
  };
  const onUploadFiles = async (files: FileList | File[]) => {
    if (!uploadFiles) {
      return [];
    }
    const uploaded = await uploadFiles(files);
    if (uploaded.length > 0) {
      await queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, workOrderId),
      });
    }
    return uploaded;
  };
  return {
    onSend,
    onStop: canUpdate ? onStop : undefined,
    onUploadFiles: uploadFiles ? onUploadFiles : undefined,
    onSubmitSurvey: (text: string) => {
      submit(text, answerSurvey.mutate);
    },
  };
}
