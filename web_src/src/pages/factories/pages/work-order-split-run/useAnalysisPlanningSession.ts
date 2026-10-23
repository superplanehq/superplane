import { factoryQueryKeys } from "@/hooks/useFactoryData";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { getApiErrorMessage } from "@/lib/errors";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

import { emptyCreateWithAgentView } from "../createWithAgentDemo";
import {
  answerPlanningSessionSurvey,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
  sendWorkOrderPlanningMessage,
} from "../planningSessionClient";
import {
  createWithAgentViewFromSession,
  mergePlanningSessionHistory,
  type PlanningSessionPayload,
} from "../planningSessionView";
import { usePlanningSessionLiveRun } from "../usePlanningSessionLiveRun";

export function workOrderPlanningSessionQueryKey(organizationId: string, factoryId: string, workOrderId: string) {
  return factoryQueryKeys.planningSession(organizationId, factoryId, workOrderId);
}

export const ANALYSIS_PLANNING_COPY = {
  composerPlaceholder: "Tell the agent more about this task",
  send: "Send",
  sendShortcut: "Enter",
  dictate: "Dictate",
  stopDictation: "Stop dictation",
  microphoneDenied: "Microphone access was denied. Allow access and try again.",
  stopped: "This analysis has stopped.",
  failedSend: "The message did not send. Try again.",
  failedLoad: "The analysis session did not load. Try again.",
  failedLoadRefinement: "The refinement session did not load. Refresh the page to try again.",
};

type AnalysisPlanningSessionArgs = {
  organizationId?: string;
  factoryId?: string;
  workOrderId?: string;
  enabled: boolean;
  canUpdate: boolean;
  /** Poll a missing session only when a refinement run can still start. */
  refinementCanStart?: boolean;
  analysisActive?: boolean;
  analysisQueued?: boolean;
  analysisDelivered?: boolean;
  isUploading?: boolean;
  uploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
};

const LIVE_SESSION_REFETCH_INTERVAL_MS = 15_000;

export function planningSessionCanStart(input: {
  factoryLoaded: boolean;
  planningEnabled: boolean;
  isDraft: boolean;
  analysisActive: boolean;
}): boolean {
  return input.factoryLoaded && input.planningEnabled && (input.isDraft || input.analysisActive);
}

/**
 * Poll a live session until it ends. The first turn has no live connection
 * yet, so the open chat relies on stored activity from this refresh. Poll a
 * missing session only when refinement can still start. An open task that
 * cannot start refinement keeps the first lookup and does not refresh again.
 */
function planningSessionNeedsPolling(
  session: PlanningSessionPayload | null | undefined,
  refinementCanStart: boolean,
): boolean {
  if (session?.id) {
    return session.state !== "ended";
  }
  return refinementCanStart;
}

function usePlanningSessionLookup(
  args: Required<Pick<AnalysisPlanningSessionArgs, "enabled">> & {
    organizationId: string;
    factoryId: string;
    workOrderId: string;
    poll: boolean;
  },
) {
  const { organizationId, factoryId, workOrderId, enabled, poll } = args;
  return useQuery<PlanningSessionPayload | null>({
    queryKey: workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId),
    queryFn: () => findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId),
    enabled: enabled && Boolean(organizationId && factoryId && workOrderId),
    structuralSharing: (previous, next) =>
      mergePlanningSessionHistory(
        previous as PlanningSessionPayload | null | undefined,
        next as PlanningSessionPayload | null,
      ),
    refetchOnWindowFocus: false,
    refetchInterval: poll ? LIVE_SESSION_REFETCH_INTERVAL_MS : undefined,
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
      queryKey: factoryQueryKeys.workOrdersPagePrefix(organizationId, factoryId),
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

function analysisView(session: PlanningSessionPayload | null, composer: string, active: boolean) {
  if (!session) {
    return { ...emptyCreateWithAgentView(), machineStatus: active ? ("starting" as const) : ("waiting" as const) };
  }
  return createWithAgentViewFromSession(session, {
    composer,
    right: emptyCreateWithAgentView().right,
    endConfirmOpen: false,
  });
}

function analysisSendState(
  session: PlanningSessionPayload | null,
  machineStatus: string,
  canUpdate: boolean,
  sendPending: boolean,
  isUploading = false,
) {
  const stopped = machineStatus === "failed" || machineStatus === "passed";
  const isLive = Boolean(session?.id && session.state !== "ended" && !stopped);
  const canRestart = Boolean(session?.id && (session.state === "ended" || stopped));
  return { isLive, canSend: canUpdate && !sendPending && !isUploading && (isLive || canRestart) };
}

function canRecoverAnalysis(
  args: AnalysisPlanningSessionArgs,
  hasSession: boolean,
  lookupPending: boolean,
  sendPending: boolean,
) {
  return (
    !hasSession &&
    !args.analysisActive &&
    !args.analysisQueued &&
    args.enabled &&
    args.refinementCanStart === true &&
    args.canUpdate &&
    !lookupPending &&
    Boolean(args.organizationId && args.factoryId && args.workOrderId) &&
    !args.isUploading &&
    !sendPending
  );
}

export function useAnalysisPlanningSession(args: AnalysisPlanningSessionArgs) {
  const {
    organizationId = "",
    factoryId = "",
    workOrderId = "",
    enabled,
    canUpdate,
    refinementCanStart = false,
    analysisActive = false,
    analysisQueued = false,
    analysisDelivered = false,
    isUploading = false,
    uploadFiles,
  } = args;
  const queryClient = useQueryClient();
  const { composer, composerError, setComposer, setComposerError } = useAnalysisComposer(
    organizationId,
    factoryId,
    workOrderId,
  );
  const queryKey = workOrderPlanningSessionQueryKey(organizationId, factoryId, workOrderId);
  const query = usePlanningSessionLookup({
    organizationId,
    factoryId,
    workOrderId,
    enabled,
    poll: planningSessionNeedsPolling(
      queryClient.getQueryData<PlanningSessionPayload | null>(queryKey),
      refinementCanStart,
    ),
  });
  const session = query.data ?? null;
  useRefreshAnalysisWorkOrder({ queryClient, organizationId, factoryId, workOrderId, session });

  const { sendMessage, answerSurvey } = useAnalysisPlanningMutations({
    organizationId,
    factoryId,
    workOrderId,
    queryClient,
    onSuccess: () => {
      setComposer("");
      setComposerError("");
    },
    onError: (error) => setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedSend)),
  });
  const messageInput = (text: string): AnalysisMessageInput => ({
    organizationId,
    factoryId,
    workOrderId,
    sessionId: session?.id,
    text,
  });

  const liveView = usePlanningSessionLiveRun(
    organizationId,
    analysisView(session, composer, query.isLoading || analysisActive),
    analysisDelivered,
  );
  const view = analysisQueued ? { ...liveView, machineStatus: "waiting" as const, survey: undefined } : liveView;
  const { isLive, canSend: canSendToSession } = analysisSendState(
    session,
    view.machineStatus,
    canUpdate,
    sendMessage.isPending || answerSurvey.isPending,
    isUploading,
  );
  const canRecover = canRecoverAnalysis(
    args,
    Boolean(session),
    query.isLoading || Boolean(query.error),
    sendMessage.isPending,
  );
  const canSend = !analysisQueued && (canSendToSession || canRecover);
  const canAnalyze = canSend && !isLive && !analysisActive && refinementCanStart;
  const submit = (text: string, send: (input: AnalysisMessageInput) => void) => {
    const trimmed = text.trim();
    if (!trimmed || !canSend) {
      return false;
    }
    setComposerError("");
    send(messageInput(trimmed));
    return true;
  };
  const onSend = async (text?: string) => {
    const trimmed = (text ?? composer).trim();
    if (!trimmed || !canSend) {
      return false;
    }
    setComposerError("");
    try {
      await sendMessage.mutateAsync(messageInput(trimmed));
      return true;
    } catch {
      return false;
    }
  };

  return {
    organizationId,
    factoryId,
    session,
    sessionId: session?.id ?? "",
    queryError: query.error,
    isLoading: query.isLoading,
    view,
    composer,
    composerError,
    canSend,
    canAnalyze,
    onAnalyze: () => (canAnalyze ? onSend(composer.trim() || "Analyze this task.") : Promise.resolve(false)),
    isUploading,
    isLive,
    showChat: Boolean(session?.id),
    onComposerChange: setComposer,
    onSend,
    onUploadFiles: uploadFiles
      ? (files: FileList | File[]) => uploadAnalysisFiles(args, queryClient, files)
      : undefined,
    onSubmitSurvey: (text: string) => {
      submit(text, answerSurvey.mutate);
    },
  };
}

function useAnalysisComposer(organizationId: string, factoryId: string, workOrderId: string) {
  const [composer, setComposer] = useState("");
  const [composerError, setComposerError] = useState("");
  useEffect(() => {
    setComposer("");
    setComposerError("");
  }, [organizationId, factoryId, workOrderId]);
  return { composer, composerError, setComposer, setComposerError };
}

type AnalysisMessageInput = {
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  sessionId?: string;
  text: string;
};

function useAnalysisPlanningMutations(args: {
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  queryClient: QueryClient;
  onSuccess: () => void;
  onError: (error: Error) => void;
}) {
  const currentScope = useRef(args);
  currentScope.current = args;
  const isCurrentScope = (scope: AnalysisMessageInput) =>
    scope.organizationId === currentScope.current.organizationId &&
    scope.factoryId === currentScope.current.factoryId &&
    scope.workOrderId === currentScope.current.workOrderId;
  const onSuccess = async (next: PlanningSessionPayload, scope: AnalysisMessageInput) => {
    const responseKey = workOrderPlanningSessionQueryKey(scope.organizationId, scope.factoryId, scope.workOrderId);
    await args.queryClient.cancelQueries({ queryKey: responseKey });
    args.queryClient.setQueryData<PlanningSessionPayload | null>(responseKey, (previous) =>
      mergePlanningSessionHistory(previous, next),
    );
    void args.queryClient.invalidateQueries({ queryKey: ["backlog-analysis-runs", scope.organizationId] });
    void args.queryClient.invalidateQueries({ queryKey: ["organizations", scope.organizationId, "workspace-usage"] });
    if (isCurrentScope(scope)) args.onSuccess();
  };
  const onError = (error: Error, scope: AnalysisMessageInput) => {
    if (isCurrentScope(scope)) args.onError(error);
  };
  const sendMessage = useMutation({
    mutationFn: (input: AnalysisMessageInput) =>
      input.sessionId
        ? sendPlanningSessionMessage(input.organizationId, input.factoryId, input.sessionId, input.text)
        : sendWorkOrderPlanningMessage(input.organizationId, input.factoryId, input.workOrderId, input.text),
    onSuccess,
    onError,
  });
  const answerSurvey = useMutation({
    mutationFn: (input: AnalysisMessageInput) =>
      answerPlanningSessionSurvey(input.organizationId, input.factoryId, input.sessionId ?? "", input.text),
    onSuccess,
    onError,
  });

  return { sendMessage, answerSurvey };
}

async function uploadAnalysisFiles(
  args: AnalysisPlanningSessionArgs,
  queryClient: QueryClient,
  files: FileList | File[],
) {
  if (!args.uploadFiles) return [];
  const uploaded = await args.uploadFiles(files);
  if (uploaded.length > 0) {
    await queryClient.invalidateQueries({
      queryKey: factoryQueryKeys.workOrderDetail(
        args.organizationId ?? "",
        args.factoryId ?? "",
        args.workOrderId ?? "",
      ),
    });
  }
  return uploaded;
}
