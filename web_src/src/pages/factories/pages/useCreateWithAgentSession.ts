import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { useCallback, useEffect, useRef, useState } from "react";

import { CREATE_WITH_AGENT_COPY, planningRefineNote } from "./createWithAgentCopy";
import {
  cancelCreateWithAgentEnd,
  emptyCreateWithAgentView,
  requestCreateWithAgentEnd,
  setCreateWithAgentComposer,
  updateCreateWithAgentDraft,
} from "./createWithAgentDemo";
import type { CreateWithAgentView } from "./createWithAgentTypes";
import {
  createPlanningSessionWorkOrder,
  describePlanningSession,
  endPlanningSession,
  sendPlanningSessionMessage,
  skipPlanningSessionDraft,
  startPlanningSession,
  updatePlanningSessionDraft,
} from "./planningSessionClient";
import { isPlanningSurveyReply } from "./planningSessionSurvey";
import { createWithAgentViewFromSession, type PlanningSessionPayload } from "./planningSessionView";

const POLL_MS = 1500;
const DRAFT_SAVE_MS = 400;
const UNMOUNT_END_MS = 100;
const pendingUnmountEnds = new Map<string, number>();

function planningSessionHookKey(organizationId: string, factoryId: string) {
  return `${organizationId}:${factoryId}`;
}

function cancelScheduledPlanningSessionEnd(key: string) {
  const timer = pendingUnmountEnds.get(key);
  if (timer === undefined) {
    return;
  }
  window.clearTimeout(timer);
  pendingUnmountEnds.delete(key);
}

function schedulePlanningSessionEnd(key: string, end: () => void) {
  cancelScheduledPlanningSessionEnd(key);
  pendingUnmountEnds.set(
    key,
    window.setTimeout(() => {
      pendingUnmountEnds.delete(key);
      end();
    }, UNMOUNT_END_MS),
  );
}

function clearDraftSaveTimer(timer: { current: number | undefined }) {
  if (timer.current === undefined) {
    return;
  }
  window.clearTimeout(timer.current);
  timer.current = undefined;
}

export function useCreateWithAgentSession(repository: string, organizationId: string, factoryId: string) {
  const [open, setOpen] = useState(false);
  const [sessionId, setSessionId] = useState("");
  const [view, setView] = useState<CreateWithAgentView>(() => emptyCreateWithAgentView(repository));
  const draftSaveTimer = useRef<number | undefined>(undefined);
  const sessionIdRef = useRef("");
  const openRef = useRef(false);
  const startGenerationRef = useRef(0);
  sessionIdRef.current = sessionId;
  openRef.current = open;

  const resetLocalSession = useCallback(() => {
    clearDraftSaveTimer(draftSaveTimer);
    setOpen(false);
    setSessionId("");
    setView(emptyCreateWithAgentView(repository));
  }, [repository]);

  const applySession = useCallback(
    (session: PlanningSessionPayload, generation: number) => {
      applyPlanningSession({
        session,
        generation,
        organizationId,
        factoryId,
        startGenerationRef,
        openRef,
        setSessionId,
        setView,
      });
    },
    [factoryId, organizationId],
  );

  usePlanningSessionPageLeave({
    open,
    sessionId,
    organizationId,
    factoryId,
    startGenerationRef,
    applySession,
  });

  const stopSession = useCallback(
    (id: string, options?: { keepalive?: boolean }) => {
      if (!id || !organizationId || !factoryId) {
        return;
      }
      const request = options
        ? endPlanningSession(organizationId, factoryId, id, options)
        : endPlanningSession(organizationId, factoryId, id);
      void request.catch(() => undefined);
    },
    [factoryId, organizationId],
  );

  useEffect(() => {
    const key = planningSessionHookKey(organizationId, factoryId);
    cancelScheduledPlanningSessionEnd(key);
    return () => {
      clearDraftSaveTimer(draftSaveTimer);
      const id = sessionIdRef.current;
      if (!openRef.current || !id) {
        return;
      }
      schedulePlanningSessionEnd(key, () => {
        stopSession(id, { keepalive: true });
      });
    };
  }, [factoryId, organizationId, stopSession]);

  const close = useCallback(() => {
    startGenerationRef.current += 1;
    const id = sessionIdRef.current;
    resetLocalSession();
    stopSession(id);
  }, [resetLocalSession, stopSession]);

  const start = useCallback(() => {
    openPlanningSession({
      repository,
      organizationId,
      factoryId,
      sessionIdRef,
      startGenerationRef,
      stopSession,
      applySession,
      resetLocalSession,
      setView,
      setSessionId,
      setOpen,
    });
  }, [applySession, factoryId, organizationId, repository, resetLocalSession, stopSession]);

  const patchDraft = (title: string, description: string) =>
    savePlanningDraft({ title, description, sessionId, organizationId, factoryId, draftSaveTimer, setView });
  const sendSessionText = (text: string, failedCopy: string) =>
    sendPlanningText({
      text,
      failedCopy,
      sessionId,
      organizationId,
      factoryId,
      startGenerationRef,
      setView,
      applySession,
    });

  return {
    open,
    view,
    start,
    close,
    ...createWithAgentViewActions({
      view,
      sessionId,
      organizationId,
      factoryId,
      startGenerationRef,
      setView,
      applySession,
      patchDraft,
      sendSessionText,
      close,
    }),
  };
}

function applyPlanningSession({
  session,
  generation,
  organizationId,
  factoryId,
  startGenerationRef,
  openRef,
  setSessionId,
  setView,
}: {
  session: PlanningSessionPayload;
  generation: number;
  organizationId: string;
  factoryId: string;
  startGenerationRef: { current: number };
  openRef: { current: boolean };
  setSessionId: (id: string) => void;
  setView: (updater: (current: CreateWithAgentView) => CreateWithAgentView) => void;
}) {
  if (generation !== startGenerationRef.current || !openRef.current) {
    const id = session.id ?? "";
    if (id && organizationId && factoryId) {
      void endPlanningSession(organizationId, factoryId, id).catch(() => undefined);
    }
    return;
  }
  setSessionId(session.id ?? "");
  setView((current) =>
    createWithAgentViewFromSession(session, {
      composer: current.composer,
      right: current.right,
      endConfirmOpen: current.endConfirmOpen,
    }),
  );
}

function openPlanningSession({
  repository,
  organizationId,
  factoryId,
  sessionIdRef,
  startGenerationRef,
  stopSession,
  applySession,
  resetLocalSession,
  setView,
  setSessionId,
  setOpen,
}: {
  repository: string;
  organizationId: string;
  factoryId: string;
  sessionIdRef: { current: string };
  startGenerationRef: { current: number };
  stopSession: (id: string) => void;
  applySession: (session: PlanningSessionPayload, generation: number) => void;
  resetLocalSession: () => void;
  setView: (view: CreateWithAgentView) => void;
  setSessionId: (id: string) => void;
  setOpen: (open: boolean) => void;
}) {
  const generation = startGenerationRef.current + 1;
  startGenerationRef.current = generation;
  stopSession(sessionIdRef.current);
  setView(emptyCreateWithAgentView(repository));
  setSessionId("");
  setOpen(true);
  void startPlanningSession(organizationId, factoryId, repository)
    .then((session) => applySession(session, generation))
    .catch((error: unknown) => {
      if (generation !== startGenerationRef.current) {
        return;
      }
      resetLocalSession();
      showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedStart));
    });
}

function usePlanningSessionPageLeave({
  open,
  sessionId,
  organizationId,
  factoryId,
  startGenerationRef,
  applySession,
}: {
  open: boolean;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  startGenerationRef: { current: number };
  applySession: (session: PlanningSessionPayload, generation: number) => void;
}) {
  useEffect(() => {
    if (!open || !sessionId || !organizationId || !factoryId) {
      return;
    }
    const generation = startGenerationRef.current;
    const poll = window.setInterval(() => {
      void describePlanningSession(organizationId, factoryId, sessionId)
        .then((session) => applySession(session, generation))
        .catch(() => undefined);
    }, POLL_MS);
    const warnBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    const endOnLeave = (event: PageTransitionEvent) => {
      if (event.persisted) {
        return;
      }
      void endPlanningSession(organizationId, factoryId, sessionId, { keepalive: true }).catch(() => undefined);
    };
    window.addEventListener("beforeunload", warnBeforeUnload);
    window.addEventListener("pagehide", endOnLeave);
    return () => {
      window.clearInterval(poll);
      window.removeEventListener("beforeunload", warnBeforeUnload);
      window.removeEventListener("pagehide", endOnLeave);
    };
  }, [applySession, factoryId, open, organizationId, sessionId, startGenerationRef]);
}

function savePlanningDraft({
  title,
  description,
  sessionId,
  organizationId,
  factoryId,
  draftSaveTimer,
  setView,
}: {
  title: string;
  description: string;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  draftSaveTimer: { current: number | undefined };
  setView: (updater: (current: CreateWithAgentView) => CreateWithAgentView) => void;
}) {
  setView((current) => updateCreateWithAgentDraft(current, { title, description }));
  if (!sessionId) {
    return;
  }
  clearDraftSaveTimer(draftSaveTimer);
  draftSaveTimer.current = window.setTimeout(() => {
    draftSaveTimer.current = undefined;
    void updatePlanningSessionDraft(organizationId, factoryId, sessionId, { title, description }).catch(
      (error: unknown) => {
        showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedDraft));
      },
    );
  }, DRAFT_SAVE_MS);
}

function sendPlanningText({
  text,
  failedCopy,
  sessionId,
  organizationId,
  factoryId,
  startGenerationRef,
  setView,
  applySession,
}: {
  text: string;
  failedCopy: string;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  startGenerationRef: { current: number };
  setView: (updater: (current: CreateWithAgentView) => CreateWithAgentView) => void;
  applySession: (session: PlanningSessionPayload, generation: number) => void;
}) {
  const body = text.trim();
  if (!body || !sessionId) {
    return;
  }
  const generation = startGenerationRef.current;
  setView((current) => ({
    ...setCreateWithAgentComposer(current, ""),
    survey: undefined,
    messages: [
      ...current.messages,
      {
        id: `local-${current.messages.length + 1}`,
        kind: "text",
        role: "user",
        text: body,
        // Sorts to the end immediately. The server round-trip replaces this
        // with the persisted message, whose created_at keeps the same
        // relative position so there is no visible jump.
        createdAtMs: Date.now(),
        ...(isPlanningSurveyReply(body) ? { origin: "survey" as const } : {}),
      },
    ],
  }));
  void sendPlanningSessionMessage(organizationId, factoryId, sessionId, body)
    .then((session) => applySession(session, generation))
    .catch((error: unknown) => {
      showErrorToast(getApiErrorMessage(error, failedCopy));
    });
}

function createWithAgentViewActions({
  view,
  sessionId,
  organizationId,
  factoryId,
  startGenerationRef,
  setView,
  applySession,
  patchDraft,
  sendSessionText,
  close,
}: {
  view: CreateWithAgentView;
  sessionId: string;
  organizationId: string;
  factoryId: string;
  startGenerationRef: { current: number };
  setView: (updater: (current: CreateWithAgentView) => CreateWithAgentView) => void;
  applySession: (session: PlanningSessionPayload, generation: number) => void;
  patchDraft: (title: string, description: string) => void;
  sendSessionText: (text: string, failedCopy: string) => void;
  close: () => void;
}) {
  return {
    onComposerChange: (value: string) => setView((current) => setCreateWithAgentComposer(current, value)),
    onSend: () => {
      const text = view.composer.trim();
      if (!text || !sessionId) {
        return;
      }
      sendSessionText(text, CREATE_WITH_AGENT_COPY.failedSend);
    },
    onSubmitSurvey: (text: string) => {
      if (!sessionId) {
        return;
      }
      sendSessionText(text, CREATE_WITH_AGENT_COPY.failedSurvey);
    },
    onDraftTitleChange: (title: string) => {
      const description = view.right.kind === "draft" ? view.right.draft.description : "";
      patchDraft(title, description);
    },
    onDraftDescriptionChange: (description: string) => {
      const title = view.right.kind === "draft" ? view.right.draft.title : "";
      patchDraft(title, description);
    },
    onCreateDraft: () => {
      if (!sessionId) {
        return;
      }
      const generation = startGenerationRef.current;
      void createPlanningSessionWorkOrder(organizationId, factoryId, sessionId)
        .then((session) => applySession(session, generation))
        .catch((error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedCreate));
        });
    },
    onSkipDraft: () => {
      if (!sessionId) {
        return;
      }
      const generation = startGenerationRef.current;
      void skipPlanningSessionDraft(organizationId, factoryId, sessionId)
        .then((session) => applySession(session, generation))
        .catch((error: unknown) => {
          showErrorToast(getApiErrorMessage(error, CREATE_WITH_AGENT_COPY.failedSkip));
        });
    },
    onSelectCreated: (order: CreateWithAgentView["created"][number]) => {
      setView((current) => ({ ...current, right: { kind: "preview", order } }));
    },
    onRefineCreated: (order: CreateWithAgentView["created"][number]) => {
      sendSessionText(planningRefineNote(order.key, order.title), CREATE_WITH_AGENT_COPY.failedSend);
    },
    onRequestClose: () => setView((current) => requestCreateWithAgentEnd(current)),
    onCancelEnd: () => setView((current) => cancelCreateWithAgentEnd(current)),
    onConfirmEnd: () => {
      close();
    },
  };
}
