import { factoryQueryKeys } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";

import { emptyCreateWithAgentView } from "../createWithAgentDemo";
import {
  endPlanningSession,
  findPlanningSessionByWorkOrder,
  sendPlanningSessionMessage,
} from "../planningSessionClient";
import { createWithAgentViewFromSession, type PlanningSessionPayload } from "../planningSessionView";
import { usePlanningSessionLiveRun } from "../usePlanningSessionLiveRun";

const POLL_MS = 1500;

export const ANALYSIS_PLANNING_COPY = {
  composerPlaceholder: "Add context for this plan",
  send: "Send",
  stopped: "This analysis has stopped.",
  failedSend: "The message did not send. Try again.",
  writing: "The agent is writing the plan.",
};

export function useAnalysisPlanningSession(args: {
  organizationId?: string;
  factoryId?: string;
  workOrderId?: string;
  enabled: boolean;
  canUpdate: boolean;
  analysisDelivered?: boolean;
}) {
  const {
    organizationId = "",
    factoryId = "",
    workOrderId = "",
    enabled,
    canUpdate,
    analysisDelivered = false,
  } = args;
  const queryClient = useQueryClient();
  const [session, setSession] = useState<PlanningSessionPayload | null>(null);
  const [composer, setComposer] = useState("");
  const [composerError, setComposerError] = useState("");
  const [sendBusy, setSendBusy] = useState(false);
  const sessionIdRef = useRef("");
  sessionIdRef.current = session?.id ?? "";

  const applySession = useCallback(
    (next: PlanningSessionPayload | null) => {
      setSession(next);
      if (!next || !organizationId || !factoryId || !workOrderId) {
        return;
      }
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderArtifacts(organizationId, factoryId, workOrderId),
      });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderChecks(organizationId, factoryId, workOrderId),
      });
      void queryClient.invalidateQueries({
        queryKey: factoryQueryKeys.workOrderDetail(organizationId, factoryId, workOrderId),
      });
    },
    [factoryId, organizationId, queryClient, workOrderId],
  );

  useEffect(() => {
    if (!enabled || !organizationId || !factoryId || !workOrderId) {
      setSession(null);
      return;
    }
    let cancelled = false;
    const load = async () => {
      try {
        const next = await findPlanningSessionByWorkOrder(organizationId, factoryId, workOrderId);
        if (!cancelled) {
          applySession(next);
        }
      } catch {
        if (!cancelled) {
          applySession(null);
        }
      }
    };
    void load();
    const timer = window.setInterval(() => {
      void load();
    }, POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [applySession, enabled, factoryId, organizationId, workOrderId]);

  const view = usePlanningSessionLiveRun(
    organizationId,
    session
      ? createWithAgentViewFromSession(session, {
          composer,
          right: emptyCreateWithAgentView().right,
          endConfirmOpen: false,
          analysisDelivered,
        })
      : emptyCreateWithAgentView(),
    analysisDelivered,
  );

  const sessionId = session?.id ?? "";
  const isLive = Boolean(
    sessionId &&
      session?.state !== "ended" &&
      view.machineStatus !== "failed" &&
      view.machineStatus !== "passed",
  );
  const canRestart = Boolean(
    sessionId &&
      (session?.state === "ended" || view.machineStatus === "failed" || view.machineStatus === "passed"),
  );
  const canSend = canUpdate && !sendBusy && (isLive || canRestart);

  const onSend = useCallback(async () => {
    const text = composer.trim();
    if (!text || !canSend || !organizationId || !factoryId || !sessionId) {
      return;
    }
    setSendBusy(true);
    setComposerError("");
    try {
      const next = await sendPlanningSessionMessage(organizationId, factoryId, sessionId, text);
      setComposer("");
      applySession(next);
    } catch (error) {
      setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedSend));
    } finally {
      setSendBusy(false);
    }
  }, [applySession, canSend, composer, factoryId, organizationId, sessionId]);

  const onSubmitSurvey = useCallback(
    async (text: string) => {
      if (!canSend || !organizationId || !factoryId || !sessionId) {
        return;
      }
      setSendBusy(true);
      setComposerError("");
      try {
        applySession(await sendPlanningSessionMessage(organizationId, factoryId, sessionId, text));
      } catch (error) {
        setComposerError(getApiErrorMessage(error, ANALYSIS_PLANNING_COPY.failedSend));
      } finally {
        setSendBusy(false);
      }
    },
    [applySession, canSend, factoryId, organizationId, sessionId],
  );

  const endSession = useCallback(async () => {
    const id = sessionIdRef.current;
    if (!id || !organizationId || !factoryId) {
      return;
    }
    await endPlanningSession(organizationId, factoryId, id).catch(() => undefined);
  }, [factoryId, organizationId]);

  return {
    organizationId,
    sessionId,
    view,
    composer,
    composerError,
    canSend,
    isLive,
    showChat: Boolean(sessionId),
    onComposerChange: setComposer,
    onSend,
    onSubmitSurvey,
    endSession,
  };
}
